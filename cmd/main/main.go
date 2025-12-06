package main

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/appservices"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/helpers"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/songrequests"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nicklaw5/helix/v2"
	"golang.org/x/net/websocket"
)

type twitchData = struct {
	accessToken     string
	login           string
	userID          string
	isAuthenticated bool
	expiresDate     time.Time
}

func main() {
	helpers.PreflightTest()
	app := NewApp()

	log.Println(app.Run())
	app.cancel()
	fmt.Print("Press 'Enter' to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

type App struct {
	twitchDataStruct     *twitchData
	helix                *helix.Client
	twitchWSService      *appservices.TwitchWS
	twitchWSIncomingMsgs chan []byte
	ctx                  context.Context
	cancel               context.CancelFunc
	clients              map[*websocket.Conn]struct{}
	clientsMu            sync.RWMutex
	clientsBroadcast     chan string
}

func NewApp() *App {
	ctx, cancel := context.WithCancel(context.Background())
	c, _ := helix.NewClient(&helix.Options{
		ClientID: data.GetTwitchClientID(),
	})
	return &App{
		twitchDataStruct:     &twitchData{},
		ctx:                  ctx,
		cancel:               cancel,
		helix:                c,
		clientsBroadcast:     make(chan string),
		twitchWSIncomingMsgs: make(chan []byte),
		clientsMu:            sync.RWMutex{},
		clients:              make(map[*websocket.Conn]struct{}),
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

	// Auto reconnect twitch ws
	go func() {
		for {
			a.twitchWSService = appservices.NewTwitchWS(a.helix, &a.twitchDataStruct.userID, &a.twitchDataStruct.login, nil, nil, nil, songrequests.GetSubscriptions(), songrequests.SetSubscriptionHandlers)
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

	log.Println("App is running on port 3999...")
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.StaticFS("/", echo.MustSubFS(staticControlPanelFS, "build"))

	apiV1 := e.Group("/api/v1")
	apiV1.POST("/twitch-oauth", a.processTwitchOAuth)
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
	exec.Command(cmd, args...).Start()
	return e.Start("127.0.0.1:3999")
}
