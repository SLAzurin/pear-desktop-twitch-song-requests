package appservices

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/staticservices"
	"github.com/gorilla/websocket"
)

type twitchIRC struct {
	stopChan          chan struct{}
	mainTwitchChannel string
	helixMain         *staticservices.TwitchHelixService
	helixBot          *staticservices.TwitchHelixService
	conn              *websocket.Conn
	log               *log.Logger
	msgChan           chan struct {
		UseSlashMe bool
		Message    string
	}
	rcvChan chan []byte
}

func (s *twitchIRC) StartCtx(ctx context.Context) error {
	const twitchIRCHost = "wss://irc-ws.chat.twitch.tv"
	s.log.Println("Twitch IRC service starting...")
	s.mainTwitchChannel = strings.ToLower(s.helixMain.GetNickname())

	oauthToken := s.helixMain.Client().GetUserAccessToken()
	loginNickname := s.helixMain.GetNickname()

	if s.helixBot != nil {
		oauthToken = s.helixBot.Client().GetUserAccessToken()
		loginNickname = s.helixBot.GetNickname()
	}

	// Connect to Twitch IRC WebSocket
	connectRetries := 0
	for s.conn == nil {
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, twitchIRCHost, nil)
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

	// rcv msg loop
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Println("Recovered in Twitch IRC receive loop:", r)
				s.Stop()
			}
		}()
		part := ""
		for {
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				s.log.Println("Error reading Twitch WS message:", err)
				break
			}
			// handle message processing here
			stringmsg := string(message)
			part += stringmsg
			index := strings.Index(part, "\r\n")
			for index != -1 {
				fullIRCMsg := part[:index]
				if fullIRCMsg != "" {
					s.rcvChan <- []byte(fullIRCMsg)
				}
				part = part[index+2:]
				index = strings.Index(part, "\r\n")
			}
		}
	}()

	// send msg loop
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Println("Recovered in Twitch IRC send loop:", r)
				s.Stop()
			}
		}()
		for msg := range s.msgChan {
			var toSend string
			if msg.UseSlashMe {
				toSend = "PRIVMSG #" + s.mainTwitchChannel + " :/me " + msg.Message
			} else {
				toSend = "PRIVMSG #" + s.mainTwitchChannel + " :" + msg.Message
			}
			err := s.conn.WriteMessage(websocket.TextMessage, []byte(toSend))
			if err != nil {
				s.log.Println("Error sending Twitch IRC message:", err)
				continue
			}
		}
	}()

	// stop handler
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	// Authenticate with Twitch IRC
	s.conn.WriteMessage(websocket.TextMessage, []byte("CAP REQ :twitch.tv/membership twitch.tv/tags twitch.tv/commands"))
	s.conn.WriteMessage(websocket.TextMessage, []byte("PASS oauth:"+oauthToken))
	s.conn.WriteMessage(websocket.TextMessage, []byte("NICK "+loginNickname))
	s.conn.WriteMessage(websocket.TextMessage, []byte("JOIN #"+s.mainTwitchChannel))

	s.log.Println("Twitch IRC service started.")
	return nil
}
func (s *twitchIRC) Stop() error {
	defer func() {
		if r := recover(); r != nil {
			s.log.Println("Recovered in Twitch IRC Stop():", r)
			// possibly already stopped and sending into a closed channel
		}
	}()
	s.log.Println("Twitch IRC service stopping...")
	s.stopChan <- struct{}{}
	return nil
}

func (s *twitchIRC) MsgChan() chan struct {
	UseSlashMe bool
	Message    string
} {
	return s.msgChan
}

func (s *twitchIRC) RcvChan() chan []byte {
	return s.rcvChan
}

func (s *twitchIRC) Log() *log.Logger {
	return s.log
}

func TwitchIRC(helixMain *staticservices.TwitchHelixService, helixBot *staticservices.TwitchHelixService) appService[struct {
	UseSlashMe bool
	Message    string
}, []byte] {
	stopChan := make(chan struct{}, 1)
	s := &twitchIRC{
		helixMain: helixMain,
		helixBot:  helixBot,
		log:       log.New(os.Stderr, "TWITCH_IRC ", log.Ldate|log.Ltime),
		msgChan: make(chan struct {
			UseSlashMe bool
			Message    string
		}),
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
