package main

import (
	"context"
	"embed"
	"log"
	"os/exec"
	"runtime"
	"sync"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/appservices"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/helpers"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nicklaw5/helix/v2"
	"golang.org/x/net/websocket"
)

type runtimeSettings = struct {
	twitchAccessToken    string
	twitchLogin          string
	twitchUserID         string
	twitchTokenExpiresIn string
}

var runtimeSettingsData = runtimeSettings{
	twitchAccessToken:    "",
	twitchLogin:          "",
	twitchUserID:         "",
	twitchTokenExpiresIn: "",
}

func loadSqliteSettings() {

}

func main() {
	helpers.PreflightTest()

	loadSqliteSettings()

	app := NewApp()
	log.Fatalln(app.Run())
}

type App struct {
	helix            helix.Client
	twitchService    appservices.TwitchWS
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
