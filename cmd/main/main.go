package main

//lint:file-ignore ST1001 Dot imports by jet
import (
	"context"
	"embed"
	"errors"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/appservices"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/databaseconn"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/helpers"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nicklaw5/helix/v2"
	"golang.org/x/net/websocket"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/gen/model"
	. "github.com/azuridayo/pear-desktop-twitch-song-requests/gen/table"
	. "github.com/go-jet/jet/v2/sqlite"
)

type twitchData = struct {
	accessToken     string
	login           string
	userID          string
	isAuthenticated bool
	expiresDate     time.Time
}

func (a *App) loadSqliteSettings() (*helix.Client, error) {
	db, err := databaseconn.NewDBConnection()
	if err != nil {
		return nil, err
	}

	twitchDataStruct := twitchData{}
	results := []model.Settings{}
	stmt := SELECT(Settings.Value).FROM(Settings).WHERE(Settings.Key.EQ(String(data.DB_KEY_TWITCH_ACCESS_TOKEN))).LIMIT(1)
	err = stmt.QueryContext(a.ctx, db, &results)
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		if result.Key == data.DB_KEY_TWITCH_ACCESS_TOKEN {
			twitchDataStruct.accessToken = result.Value
		}
	}

	c, err := helix.NewClient(&helix.Options{
		ClientID: data.GetTwitchClientID(),
	})
	if err != nil {
		return nil, err
	}

	if twitchDataStruct.accessToken != "" {
		isValid, response, err := c.ValidateToken(twitchDataStruct.accessToken)
		if err != nil {
			// req error
			return nil, err
		}
		if response.StatusCode == http.StatusOK && isValid {
			c.SetUserAccessToken(twitchDataStruct.accessToken)
			twitchDataStruct.isAuthenticated = true

			twitchDataStruct.userID = response.Data.UserID
			twitchDataStruct.login = response.Data.Login
			expiresIn := response.Data.ExpiresIn
			strDate := response.Header.Get("Date")
			t, err := time.Parse(data.TWITCH_SERVER_DATE_LAYOUT, strDate)
			if err != nil {
				return nil, errors.New("Failed to validate server date time expiry, original error:\n" + err.Error())
			}
			t = t.Add(time.Duration(expiresIn) * time.Second)
			twitchDataStruct.expiresDate = t
		}
	}

	a.twitchDataStruct = twitchDataStruct

	return c, nil
}

func main() {
	helpers.PreflightTest()
	app := NewApp()
	log.Fatalln(app.Run())
}

type App struct {
	twitchDataStruct twitchData
	helix            helix.Client
	twitchWSService  appservices.TwitchWS
	ctx              context.Context
	cancel           context.CancelFunc
	clients          map[*websocket.Conn]bool
	clientsMu        sync.RWMutex
	clientsBroadcast chan string
}

func NewApp() *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		ctx:    ctx,
		cancel: cancel,
	}
}

//go:embed build/*
var staticControlPanelFS embed.FS

func (a *App) Run() error {
	// load sqlite
	a.loadSqliteSettings()

	log.Println("App is running on port 3999...")
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.StaticFS("/", echo.MustSubFS(staticControlPanelFS, "build"))

	apiV1 := e.Group("/api/v1")
	apiV1.POST("/twitch-oauth", a.handleTwitchOAuth)
	apiV1.GET("/ws", a.handleAppWs)

	e.Pre(middleware.Rewrite(map[string]string{
		"/proxy-pear-desktop/*": "/$1",
	}))

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
