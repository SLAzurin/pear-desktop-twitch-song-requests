package appservices

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/gorilla/websocket"
	"github.com/nicklaw5/helix/v2"
)

// TODO: NOT VALIDATED, NEED TO CHECK AGAIN AFTER FINISHING IRC

type twitchWS struct {
	stopChan          chan struct{}
	mainTwitchChannel string
	helixMain         *helix.Client
	conn              *websocket.Conn
	log               *log.Logger
	msgChan           chan struct{}
	rcvChan           chan []byte
}

func (s *twitchWS) StartCtx(ctx context.Context) error {
	const twitchWsHost = "wss://eventsub.wss.twitch.tv/ws"
	// TODO: get helixMain channel name
	s.log.Println("Twitch IRC service starting...")
	r, err := s.helixMain.GetUsers(&helix.UsersParams{})
	if err != nil {
		s.log.Println("Error getting main Twitch user info:", err)
		return err
	}
	s.mainTwitchChannel = r.Data.Users[0].Login

	// Connect to Twitch IRC WebSocket
	connectRetries := 0
	for s.conn == nil {
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, twitchWsHost, nil)
		if err != nil {
			if connectRetries > 128 {
				s.log.Println("Last retry took 128s and still didn't reconnect")
				s.log.Println("Force closing")
				s.Stop() // no error to handle because this was never initialized
				return errors.New("failed to retry 128s later")
			}
			s.log.Println("Failed to connect", err)
			s.log.Println("Retrying")
			if connectRetries == 0 {
				connectRetries = 1
			} else {
				connectRetries *= 2
			}
			continue
		}
		connectRetries = 0
		s.conn = conn
	}
	go func() {
		for {
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				s.log.Println("Error reading Twitch WS message:", err)
				break
			}
			s.log.Println("Received Twitch WS message", string(message))
		}
	}()
	return nil
}
func (s *twitchWS) Stop() error {
	s.log.Println("Twitch IRC service stopping...")
	s.stopChan <- struct{}{}
	return nil
}

func (s *twitchWS) MsgChan() chan struct{} {
	return s.msgChan
}

func (s *twitchWS) RcvChan() chan []byte {
	return s.rcvChan
}

func (s *twitchWS) Log() *log.Logger {
	return s.log
}

func TwitchWS(helixMain *helix.Client) appService[struct{}, []byte] {
	stopChan := make(chan struct{}, 1)
	s := &twitchWS{
		helixMain:         helixMain,
		log:               log.New(os.Stderr, "TWITCH_WS ", log.Ldate|log.Ltime),
		msgChan:           make(chan struct{}),
		rcvChan:           make(chan []byte),
		stopChan:          stopChan,
		mainTwitchChannel: "",  // filled duing startup
		conn:              nil, // filled during startup
	}

	go func() {
		<-stopChan
		close(stopChan)
		close(s.msgChan)
		err := s.conn.Close()
		if err != nil {
			s.log.Println("Error closing Twitch IRC connection:", err)
			s.log.Println("Ignoring error...")
		}
	}()

	return s
}
