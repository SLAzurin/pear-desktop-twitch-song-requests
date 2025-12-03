package appservices

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/coder/websocket"
)

// SongInfo represents detailed song information from Pear Desktop websocket
type SongInfo struct {
	Title          string `json:"title"`
	Artist         string `json:"artist"`
	URL            string `json:"url"`
	SongDuration   int    `json:"songDuration"`
	ImageSrc       string `json:"imageSrc"`
	ElapsedSeconds int    `json:"elapsedSeconds"`
	IsPaused       bool   `json:"isPaused"`
	VideoID        string `json:"videoId"`
}

// PositionChangedMessage represents a position update from the websocket
type PositionChangedMessage struct {
	Type     string `json:"type"` // "POSITION_CHANGED"
	Position int    `json:"position"`
}

// VideoChangedMessage represents a video change from the websocket
type VideoChangedMessage struct {
	Type     string   `json:"type"` // "VIDEO_CHANGED"
	Song     SongInfo `json:"song"`
	Position int      `json:"position"`
}

// WebSocketMessage represents any websocket message from Pear Desktop
type WebSocketMessage struct {
	Type     string   `json:"type"`
	Position int      `json:"position,omitempty"`
	Song     SongInfo `json:"song,omitempty"`
}

// WebSocketStateUpdate represents a music player state update from the websocket
type WebSocketStateUpdate struct {
	IsPlaying      bool      `json:"isPlaying"`
	CurrentSong    string    `json:"currentSong,omitempty"`
	Artist         string    `json:"artist,omitempty"`
	URL            string    `json:"url,omitempty"`
	SongDuration   int       `json:"songDuration,omitempty"`
	ImageSrc       string    `json:"imageSrc,omitempty"`
	ElapsedSeconds int       `json:"elapsedSeconds,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// PearDesktopService handles websocket connection to Pear Desktop for music player state updates
type PearDesktopService struct {
	stopChan          chan struct{}
	wsURL             string
	conn              *websocket.Conn
	log               *log.Logger
	msgChan           chan WebSocketStateUpdate
	rcvChan           chan WebSocketStateUpdate
	reconnectInterval time.Duration
}

// StartCtx starts the websocket connection and begins listening for state updates
func (s *PearDesktopService) StartCtx(ctx context.Context) error {
	s.log.Println("Pear Desktop WS service starting...")

	// Start reconnection handler (this will handle initial connection and retries)
	go s.handleReconnection(ctx)

	// Stop handler
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	s.log.Println("Pear Desktop WS service started.")
	return nil
}

// connect establishes the websocket connection
func (s *PearDesktopService) connect() error {
	u, err := url.Parse(s.wsURL)
	if err != nil {
		return err
	}

	// Create headers with Authorization (Bearer token can be empty)
	headers := http.Header{}
	headers.Set("Authorization", "Bearer ")

	s.log.Printf("Attempting websocket connection to: %s", u.String())
	conn, _, err := websocket.Dial(context.Background(), u.String(), &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		s.log.Printf("WebSocket connection failed: %v", err)
		return err
	}

	s.log.Println("WebSocket connection established successfully")
	s.conn = conn
	return nil
}

// handleMessages processes incoming websocket messages
func (s *PearDesktopService) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			s.log.Println("Recovered in handleMessages:", r)
		}
	}()

	for {
		select {
		case <-s.stopChan:
			return
		default:
			if s.conn == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			_, message, err := s.conn.Read(context.Background())
			if err != nil {
				s.log.Printf("WebSocket read error: %v", err)
				s.conn = nil
				continue
			}

			// Parse the message to determine its type
			var wsMsg WebSocketMessage
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				s.log.Printf("Failed to unmarshal websocket message: %v", err)
				continue
			}

			// Create state update based on message type
			var update WebSocketStateUpdate
			update.Timestamp = time.Now()

			switch wsMsg.Type {
			case "POSITION_CHANGED":
				// Position changed - update elapsed seconds
				update.ElapsedSeconds = wsMsg.Position

			case "VIDEO_CHANGED":
				// Video changed - extract song information
				if wsMsg.Song.Title != "" {
					update.CurrentSong = wsMsg.Song.Title
					update.Artist = wsMsg.Song.Artist
					update.URL = wsMsg.Song.URL
					update.SongDuration = wsMsg.Song.SongDuration
					update.ImageSrc = wsMsg.Song.ImageSrc
					update.ElapsedSeconds = wsMsg.Position
					update.IsPlaying = !wsMsg.Song.IsPaused

					s.log.Printf("Video changed - Title: %s, Artist: %s, Duration: %d",
						wsMsg.Song.Title, wsMsg.Song.Artist, wsMsg.Song.SongDuration)
				}

			default:
				s.log.Printf("Unknown message type: %s", wsMsg.Type)
				continue
			}

			// Send to receive channel for external listeners
			select {
			case s.rcvChan <- update:
			default:
				s.log.Println("Receive channel full, dropping message")
			}
		}
	}
}

// handleReconnection manages reconnection logic
func (s *PearDesktopService) handleReconnection(ctx context.Context) {
	// Initial connection attempt
	s.log.Println("Attempting initial connection to Pear Desktop WS...")
	if err := s.connect(); err != nil {
		s.log.Printf("Initial connection failed: %v", err)
	} else {
		s.log.Println("Connected to Pear Desktop WS")
		// Start message handling goroutine when connected
		go s.handleMessages()
	}

	ticker := time.NewTicker(s.reconnectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			if s.conn == nil {
				s.log.Println("Attempting to reconnect to Pear Desktop WS...")
				if err := s.connect(); err != nil {
					s.log.Printf("Reconnection failed: %v", err)
				} else {
					s.log.Println("Reconnected to Pear Desktop WS")
					// Start message handling goroutine when reconnected
					go s.handleMessages()
				}
			}
		}
	}
}

// Stop closes the websocket connection and stops the service
func (s *PearDesktopService) Stop() error {
	defer func() {
		if r := recover(); r != nil {
			s.log.Println("Recovered in PearDesktopService Stop():", r)
		}
	}()

	s.log.Println("Pear Desktop WS service stopping...")
	close(s.stopChan)

	if s.conn != nil {
		err := s.conn.Close(websocket.StatusNormalClosure, "service stopping")
		if err != nil {
			s.log.Printf("Error closing websocket connection: %v", err)
		}
		s.conn = nil
	}

	s.log.Println("Pear Desktop WS service stopped.")
	return nil
}

// MsgChan returns the message channel (for sending messages if needed)
func (s *PearDesktopService) MsgChan() chan WebSocketStateUpdate {
	return s.msgChan
}

// RcvChan returns the receive channel for incoming music player state updates
func (s *PearDesktopService) RcvChan() chan WebSocketStateUpdate {
	return s.rcvChan
}

// Log returns the logger
func (s *PearDesktopService) Log() *log.Logger {
	return s.log
}

// NewPearDesktopService creates a new Pear Desktop websocket service
func NewPearDesktopService() *PearDesktopService {
	stopChan := make(chan struct{})
	return &PearDesktopService{
		stopChan:          stopChan,
		wsURL:             "ws://" + data.GetPearDesktopHost() + "/api/v1/ws",
		log:               log.New(os.Stderr, "PEAR_DESKTOP_WS ", log.Ldate|log.Ltime),
		msgChan:           make(chan WebSocketStateUpdate, 100),
		rcvChan:           make(chan WebSocketStateUpdate, 100),
		reconnectInterval: 5 * time.Second,
	}
}
