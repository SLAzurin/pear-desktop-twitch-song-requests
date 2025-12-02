package appservices

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/staticservices"

	"github.com/joeyak/go-twitch-eventsub/v3"
)

type TwitchWS struct {
	stopChan          chan struct{}
	mainTwitchChannel string
	helixMain         *staticservices.TwitchHelixService
	client            *twitch.Client
	log               *log.Logger
	msgChan           chan []byte
	rcvChan           chan []byte
	subs              []twitch.EventSubscription
	setupHandlers     func(log *log.Logger, client *twitch.Client)
}

func (s *TwitchWS) StartCtx(ctx context.Context) error {
	s.log.Println("Twitch WS service starting...")
	s.mainTwitchChannel = strings.ToLower(s.helixMain.GetNickname())

	s.client = twitch.NewClient()
	s.client.OnWelcome(func(message twitch.WelcomeMessage) {
		s.log.Printf("WELCOME: subscribing to events...\n")
		clientID := data.GetTwitchClientID()
		accessToken := s.helixMain.Client().GetUserAccessToken()
		userID := s.helixMain.GetUserID()

		for _, event := range s.subs {
			s.log.Printf("subscribing to %s\n", event)

			condition := map[string]string{
				"broadcaster_user_id": userID,
			}
			if event == twitch.SubChannelChatMessage {
				condition["user_id"] = userID
			}

			_, err := twitch.SubscribeEvent(twitch.SubscribeRequest{
				SessionID:   message.Payload.Session.ID,
				ClientID:    clientID,
				AccessToken: accessToken,
				Event:       event,
				Condition:   condition,
			})
			if err != nil {
				s.log.Printf("ERROR: %v\n", err)
			}
		}
	})
	s.client.OnRevoke(func(message twitch.RevokeMessage) {
		s.log.Printf("REVOKE: %v\n", message)
	})
	s.setupHandlers(s.log, s.client)

	go func() {
		err := s.client.Connect()
		if err != nil {
			s.Stop()
		}
	}()

	// stop handler
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	s.log.Println("Twitch WS service started.")
	return nil
}
func (s *TwitchWS) Stop() error {
	defer func() {
		if r := recover(); r != nil {
			s.log.Println("Recovered in Twitch WS Stop():", r)
			// possibly already stopped and sending into a closed channel
		}
	}()
	s.log.Println("Twitch WS service stopping...")
	s.stopChan <- struct{}{}
	return nil
}

func (s *TwitchWS) Client() *twitch.Client {
	return s.client
}

func (s *TwitchWS) MsgChan() chan any {
	return nil
}

func (s *TwitchWS) RcvChan() chan any {
	return nil
}

func (s *TwitchWS) Log() *log.Logger {
	return s.log
}

func NewTwitchWS(helixMain *staticservices.TwitchHelixService, helixBot *staticservices.TwitchHelixService, subs []twitch.EventSubscription, setupHandlers func(log *log.Logger, client *twitch.Client)) *TwitchWS {
	stopChan := make(chan struct{})
	s := &TwitchWS{
		helixMain:         helixMain,
		log:               log.New(os.Stderr, "TWITCH_WS ", log.Ldate|log.Ltime),
		msgChan:           nil,
		rcvChan:           nil,
		stopChan:          stopChan,
		mainTwitchChannel: "", // filled duing startup
		subs:              subs,
		setupHandlers:     setupHandlers,
	}

	go func() {
		<-stopChan
		close(stopChan)
		s.log.Println("Twitch WS service stopping...")
		if s.client != nil {
			err := s.client.Close()
			if err != nil {
				s.log.Printf("ERROR closing Twitch WS client: %v\n", err)
			}
			s.client = nil
		}
		s.log.Println("Twitch WS service stopped.")
	}()

	return s
}
