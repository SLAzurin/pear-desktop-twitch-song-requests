package main

import (
	"context"
	"embed"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/appservices"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/helpers"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/staticservices"
	"github.com/coder/websocket"
	_ "github.com/joho/godotenv/autoload"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	helpers.PreflightTest()

	app := NewApp()
	log.Fatalln(app.Run())
}

type App struct {
	pearDesktopService *staticservices.PearDesktopService
	pearDesktopWS      *appservices.PearDesktopService
	currentPlayerState *staticservices.MusicPlayerState
	stateMutex         sync.RWMutex
	// Frontend websocket clients
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	broadcast chan staticservices.MusicPlayerState
	// Connection status tracking
	pearDesktopConnected bool
	pearDesktopStatusMu  sync.RWMutex
	statusBroadcast      chan ConnectionStatus
	ctx                  context.Context
	cancel               context.CancelFunc
}

type ConnectionStatus struct {
	FrontendConnected    bool `json:"frontend_connected"`
	PearDesktopConnected bool `json:"pear_desktop_connected"`
}

func NewApp() *App {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize Pear Desktop static service
	pearDesktopService := staticservices.NewPearDesktopService()

	// Initialize Pear Desktop websocket service
	pearDesktopWS := appservices.NewPearDesktopService()

	return &App{
		pearDesktopService:   pearDesktopService,
		pearDesktopWS:        pearDesktopWS,
		clients:              make(map[*websocket.Conn]bool),
		broadcast:            make(chan staticservices.MusicPlayerState, 10),
		statusBroadcast:      make(chan ConnectionStatus, 10),
		pearDesktopConnected: false,
		ctx:                  ctx,
		cancel:               cancel,
	}
}

//go:embed build/*
var staticControlPanelFS embed.FS

func (a *App) Run() error {
	log.Println("App is running on port 3999...")

	// Ensure services are stopped on exit
	defer a.cancel()

	// Test Pear Desktop connection and get initial state
	if err := a.pearDesktopService.TestConnection(); err != nil {
		log.Printf("Warning: Pear Desktop service not available: %v", err)
	} else {
		log.Println("Pear Desktop service connected successfully")

		// Fetch initial player state
		if initialState, err := a.pearDesktopService.GetInitialPlayerState(); err != nil {
			log.Printf("Warning: Failed to get initial player state: %v", err)
		} else {
			a.stateMutex.Lock()
			a.currentPlayerState = initialState
			a.stateMutex.Unlock()
			log.Printf("Initialized player state: %+v", initialState)
		}
	}

	// Start Pear Desktop websocket service asynchronously
	go func() {
		if err := a.pearDesktopWS.StartCtx(a.ctx); err != nil {
			log.Printf("Failed to start Pear Desktop WS service: %v", err)
			// Don't fail the app, just log the error and continue
			// The service will retry connection periodically
		}
	}()

	// Start goroutine to handle music player state updates
	go a.handleMusicPlayerUpdates()

	// Start goroutine to handle websocket broadcasting
	go a.handleWebSocketBroadcast()

	// Start goroutine to handle status broadcasting
	go a.handleStatusBroadcast()

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Recover())
	e.StaticFS("/", echo.MustSubFS(staticControlPanelFS, "build"))

	apiV1 := e.Group("/api/v1")
	apiV1.POST("/twitch-oauth", a.handleTwitchOAuth)
	apiV1.GET("/music/state", a.handleGetMusicState)
	apiV1.POST("/music/state", a.handleSetMusicState)
	apiV1.GET("/music/ws", a.handleWebSocket)

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

	// Start server with graceful shutdown
	go func() {
		<-a.ctx.Done()
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := e.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	return e.Start("127.0.0.1:3999")
}

func (a *App) handleTwitchOAuth(c echo.Context) error {
	// auth data in url hash string params as get request
	body := c.Request().Body
	rawBodyData, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	defer body.Close()

	authData := struct {
		Something string
	}{}
	err = json.Unmarshal(rawBodyData, &authData)
	return echo.NewHTTPError(http.StatusTeapot, "under construction", err)
}

func (a *App) handleGetMusicState(c echo.Context) error {
	// First try to get real-time data from websocket updates
	a.stateMutex.RLock()
	if a.currentPlayerState != nil {
		state := a.currentPlayerState
		a.stateMutex.RUnlock()
		log.Printf("Returning real-time player state: %+v", state)
		return c.JSON(http.StatusOK, state)
	}
	a.stateMutex.RUnlock()

	// Fall back to REST API call
	state, err := a.pearDesktopService.GetMusicPlayerState()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get music player state", err)
	}
	log.Printf("Returning REST API player state: %+v", state)
	return c.JSON(http.StatusOK, state)
}

func (a *App) handleSetMusicState(c echo.Context) error {
	var state staticservices.MusicPlayerState
	if err := c.Bind(&state); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body", err)
	}

	if err := a.pearDesktopService.SetMusicPlayerState(&state); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to set music player state", err)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) handleWebSocket(c echo.Context) error {
	ws, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // Allow any origin for development
	})
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return err
	}
	defer ws.Close(websocket.StatusInternalError, "connection closed")

	// Add client to the map
	a.clientsMu.Lock()
	a.clients[ws] = true
	a.clientsMu.Unlock()

	log.Printf("Frontend WebSocket client connected. Total clients: %d", len(a.clients))

	// Send current connection status
	a.updateConnectionStatus()

	// Send current state immediately if available
	a.stateMutex.RLock()
	if a.currentPlayerState != nil {
		if err := ws.Write(a.ctx, websocket.MessageText, a.stateToJSON(a.currentPlayerState)); err != nil {
			log.Printf("Failed to send initial state: %v", err)
		}
	}
	a.stateMutex.RUnlock()

	// Handle client disconnection
	defer func() {
		a.clientsMu.Lock()
		delete(a.clients, ws)
		a.clientsMu.Unlock()
		log.Printf("Frontend WebSocket client disconnected. Total clients: %d", len(a.clients))
	}()

	// Keep connection alive and handle any incoming messages
	for {
		_, _, err := ws.Read(a.ctx)
		if err != nil {
			break
		}
		// For now, we don't handle incoming messages from frontend
		// This could be extended for bidirectional communication
	}

	return nil
}

func (a *App) handleWebSocketBroadcast() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case state := <-a.broadcast:
			a.clientsMu.RLock()
			clients := make([]*websocket.Conn, 0, len(a.clients))
			for client := range a.clients {
				clients = append(clients, client)
			}
			a.clientsMu.RUnlock()

			// Send to all connected clients
			for _, client := range clients {
				if err := client.Write(a.ctx, websocket.MessageText, a.stateToJSON(&state)); err != nil {
					log.Printf("Failed to broadcast to client: %v", err)
					// Remove broken connection
					a.clientsMu.Lock()
					delete(a.clients, client)
					a.clientsMu.Unlock()
				}
			}
		}
	}
}

func (a *App) stateToJSON(state *staticservices.MusicPlayerState) []byte {
	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Failed to marshal state to JSON: %v", err)
		return []byte("{}")
	}
	return data
}

func (a *App) handleStatusBroadcast() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case status := <-a.statusBroadcast:
			a.clientsMu.RLock()
			clients := make([]*websocket.Conn, 0, len(a.clients))
			for client := range a.clients {
				clients = append(clients, client)
			}
			a.clientsMu.RUnlock()

			// Send status to all connected clients
			for _, client := range clients {
				if err := client.Write(a.ctx, websocket.MessageText, a.statusToJSON(status)); err != nil {
					log.Printf("Failed to broadcast status to client: %v", err)
					// Remove broken connection
					a.clientsMu.Lock()
					delete(a.clients, client)
					a.clientsMu.Unlock()
				}
			}
		}
	}
}

func (a *App) statusToJSON(status ConnectionStatus) []byte {
	data, err := json.Marshal(status)
	if err != nil {
		log.Printf("Failed to marshal status to JSON: %v", err)
		return []byte("{}")
	}
	return data
}

func (a *App) updateConnectionStatus() {
	a.pearDesktopStatusMu.RLock()
	status := ConnectionStatus{
		FrontendConnected:    len(a.clients) > 0,
		PearDesktopConnected: a.pearDesktopConnected,
	}
	a.pearDesktopStatusMu.RUnlock()

	select {
	case a.statusBroadcast <- status:
	default:
		log.Printf("Status broadcast channel full, skipping update")
	}
}

func (a *App) handleMusicPlayerUpdates() {
	// Set initial Pear Desktop connection status
	a.pearDesktopStatusMu.Lock()
	a.pearDesktopConnected = false
	a.pearDesktopStatusMu.Unlock()
	a.updateConnectionStatus()

	for {
		select {
		case <-a.ctx.Done():
			return
		case update := <-a.pearDesktopWS.RcvChan():
			// Mark Pear Desktop as connected when we receive updates
			a.pearDesktopStatusMu.Lock()
			wasConnected := a.pearDesktopConnected
			a.pearDesktopConnected = true
			a.pearDesktopStatusMu.Unlock()

			// Send status update if connection state changed
			if !wasConnected {
				a.updateConnectionStatus()
			}

			a.stateMutex.Lock()
			// If we don't have a current state, initialize it
			if a.currentPlayerState == nil {
				a.currentPlayerState = &staticservices.MusicPlayerState{}
			}

			// Update only the fields that changed based on the websocket message
			stateChanged := false

			// Handle different types of updates
			if update.CurrentSong != "" {
				// VIDEO_CHANGED message - update all song info
				a.currentPlayerState.IsPlaying = *update.IsPlaying
				a.currentPlayerState.CurrentSong = update.CurrentSong
				a.currentPlayerState.Artist = update.Artist
				a.currentPlayerState.URL = update.URL
				a.currentPlayerState.SongDuration = update.SongDuration
				a.currentPlayerState.ImageSrc = update.ImageSrc
				a.currentPlayerState.ElapsedSeconds = update.ElapsedSeconds
				log.Printf("Updated song info: %s by %s (%ds)", update.CurrentSong, update.Artist, update.SongDuration)
				stateChanged = true
			} else if update.ElapsedSeconds >= 0 {
				// POSITION_CHANGED or PLAYER_STATE_CHANGED message
				a.currentPlayerState.ElapsedSeconds = update.ElapsedSeconds
				// For PLAYER_STATE_CHANGED, also update IsPlaying if it's different
				if update.IsPlaying != nil && *update.IsPlaying != a.currentPlayerState.IsPlaying {
					a.currentPlayerState.IsPlaying = *update.IsPlaying
					log.Printf("Updated playing state: %t (from PLAYER_STATE_CHANGED)", update.IsPlaying)
				}
				stateChanged = true
			} else if update.IsPlaying != nil && *update.IsPlaying != a.currentPlayerState.IsPlaying {
				// PLAYER_STATE_CHANGED message with only IsPlaying change
				a.currentPlayerState.IsPlaying = *update.IsPlaying
				log.Printf("Updated playing state only: %t", update.IsPlaying)
				stateChanged = true
			}

			// Broadcast update to frontend websocket clients
			if stateChanged {
				log.Printf("Broadcasting state update - Playing: %t, Song: %s", a.currentPlayerState.IsPlaying, a.currentPlayerState.CurrentSong)
				select {
				case a.broadcast <- *a.currentPlayerState:
					log.Printf("State update broadcast successfully")
				default:
					log.Printf("Broadcast channel full, skipping update")
				}
			} else {
				log.Printf("No state changes detected in update")
			}
			a.stateMutex.Unlock()
		}
	}
}
