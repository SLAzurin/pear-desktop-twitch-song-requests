package main

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/appservices"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/helpers"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/songrequests"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nicklaw5/helix/v2"
	"github.com/recws-org/recws"
	"golang.org/x/net/websocket"
)

type twitchData struct {
	accessToken     string
	login           string
	userID          string
	isAuthenticated bool
	expiresDate     time.Time
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	helpers.PreflightTest()
	app := NewApp()

	go func() {
		log.Println(app.Run())
	}()
	<-sigs
	app.cancel()

	fmt.Print("Press 'Enter' to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

type App struct {
	twitchDataStruct        *twitchData
	twitchDataStructBot     *twitchData
	helix                   *helix.Client
	helixBot                *helix.Client
	twitchWSService         *appservices.TwitchWS
	twitchWSBotService      *appservices.TwitchWS
	streamOnline            bool
	twitchWSIncomingMsgs    chan []byte
	pearDesktopIncomingMsgs chan []byte
	ctx                     context.Context
	cancel                  context.CancelFunc
	clients                 map[*websocket.Conn]struct{}
	clientsMu               sync.RWMutex
	clientsBroadcast        chan string
	songRequestRewardID     string
}

func NewApp() *App {
	ctx, cancel := context.WithCancel(context.Background())
	c, _ := helix.NewClient(&helix.Options{
		ClientID: data.GetTwitchClientID(),
	})
	c2, _ := helix.NewClient(&helix.Options{
		ClientID: data.GetTwitchClientID(),
	})
	return &App{
		twitchDataStruct:        &twitchData{},
		twitchDataStructBot:     &twitchData{},
		ctx:                     ctx,
		cancel:                  cancel,
		helix:                   c,
		helixBot:                c2,
		clientsBroadcast:        make(chan string),
		twitchWSIncomingMsgs:    make(chan []byte),
		clientsMu:               sync.RWMutex{},
		clients:                 make(map[*websocket.Conn]struct{}),
		pearDesktopIncomingMsgs: make(chan []byte),
	}
}

//go:embed build/*
var staticControlPanelFS embed.FS

func (a *App) Run() error {
	// load sqlite
	err := a.loadSqliteSettings()
	if err != nil {
		return err
	}

	// Auto reconnect pear desktop and funnel mesasges to channel
	log.Println("Pear Desktop WS service starting...")
	ws := recws.RecConn{
		RecIntvlFactor: 1,               // multiplier backoff
		RecIntvlMin:    3 * time.Second, // start time
		NonVerbose:     true,
		SubscribeHandler: func() error {
			log.Println("Connected to Pear Desktop")
			return nil
		},
	}
	ws.Dial("ws://"+songrequests.GetPearDesktopHost()+"/api/v1/ws", nil)
	go func() {
		for {
			select {
			case <-a.ctx.Done():
				go ws.Close()
				return
			default:
				if !ws.IsConnected() {
					time.Sleep(3 * time.Second)
					continue
				}
				_, message, err := ws.Conn.ReadMessage()
				if err != nil {
					time.Sleep(3 * time.Second)
					continue
				}

				a.pearDesktopIncomingMsgs <- message
			}

		}
	}()

	// Handle Pear desktop messages
	go a.handlePearDesktopMsgs()

	// Auto reconnect twitch ws
	go func() {
		for {
			a.twitchWSService = appservices.NewTwitchWS(a.helix, &a.twitchDataStruct.userID, &a.twitchDataStruct.login, nil, nil, nil, songrequests.GetSubscriptions(), a.SetSubscriptionHandlers, false)
			if a.helix.GetUserAccessToken() != "" {
				valid, _, _ := a.helix.ValidateToken(a.helix.GetUserAccessToken())
				if valid {
					err := a.twitchWSService.StartCtx(a.ctx)
					if err == nil {
						// graceful shutdown
						return
					}
					log.Println("Twitch WS disconnected, attempt to reconnect")
				}
				// always sleep 5s after token validation
				time.Sleep(5 * time.Second)
			} else {
				time.Sleep(5 * time.Second)
			}
		}
	}()

	// Auto reconnect twitch ws
	go func() {
		for {
			a.twitchWSBotService = appservices.NewTwitchWS(a.helixBot, &a.twitchDataStructBot.userID, &a.twitchDataStructBot.login, a.helix, &a.twitchDataStruct.userID, &a.twitchDataStruct.login, songrequests.GetSubscriptionsBot(), a.SetSubscriptionHandlersBot, true)
			if a.helixBot.GetUserAccessToken() != "" {
				valid, _, _ := a.helixBot.ValidateToken(a.helixBot.GetUserAccessToken())
				if valid {
					err := a.twitchWSBotService.StartCtx(a.ctx)
					if err == nil {
						// graceful shutdown
						return
					}
					log.Println("Twitch WS disconnected, attempt to reconnect")
				}
				// always sleep 5s after token validation
				time.Sleep(5 * time.Second)
			} else {
				time.Sleep(5 * time.Second)
			}
		}
	}()

	// Send msgs to ws clients
	go func() {
		for {
			data := <-a.clientsBroadcast
			a.clientsMu.Lock()
			for ws := range a.clients {
				websocket.Message.Send(ws, data)
			}
			a.clientsMu.Unlock()
		}
	}()

	// Process song requests
	go func() {
		for msg := range srChan {
			a.songRequestLogic(msg.song, msg.event)
		}
	}()

	// Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       "build",
		Index:      "index.html",
		Filesystem: http.FS(staticControlPanelFS),
		HTML5:      true,
	}))

	apiV1 := e.Group("/api/v1")
	apiV1.POST("/twitch-oauth", a.processTwitchOAuth)
	apiV1.PATCH("/settings", a.processTwitchSettings)
	apiV1.GET("/ws", a.handleAppWs)

	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, "http://localhost:3999/") // must use localhost here because twitch does not allow 127.0.0.1
	twitchTokenExpiresSoon := a.twitchDataStruct.isAuthenticated && time.Now().Add(-15*24*time.Hour).After(a.twitchDataStruct.expiresDate)
	if a.twitchDataStruct.isAuthenticated && twitchTokenExpiresSoon {
		log.Println("ALERT! Token expiry is soon, consider refreshing token.")
	}
	twitchTokenBotExpiresSoon := a.twitchDataStructBot.isAuthenticated && time.Now().Add(-15*24*time.Hour).After(a.twitchDataStructBot.expiresDate)
	if a.twitchDataStructBot.isAuthenticated && twitchTokenBotExpiresSoon {
		log.Println("ALERT! Bot Token expiry is soon, consider refreshing token.")
	}
	if !a.twitchDataStruct.isAuthenticated || a.songRequestRewardID == "" || twitchTokenExpiresSoon || twitchTokenBotExpiresSoon {
		exec.Command(cmd, args...).Start()
	} else {
		time.Sleep(5 * time.Second)
		log.Println("Friendly reminder, the control panel is available at http://localhost:3999/")
	}
	return e.Start("127.0.0.1:3999")
}
