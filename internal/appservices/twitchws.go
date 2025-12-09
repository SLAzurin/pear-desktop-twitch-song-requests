package appservices

import (
	"context"
	"log"
	"os"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/nicklaw5/helix/v2"

	"github.com/joeyak/go-twitch-eventsub/v3"
)

type TwitchWS struct {
	botTwitchChannel  *string
	mainTwitchChannel *string
	mainUserId        *string
	botUserId         *string
	helixMain         *helix.Client
	client            *twitch.Client
	log               *log.Logger
	subs              []twitch.EventSubscription
	setupHandlers     func()
}

func (s *TwitchWS) StartCtx(ctx context.Context) error {
	s.log.Println("Twitch WS service starting...")

	s.client = twitch.NewClient()
	hasSubError := false
	s.client.OnWelcome(func(message twitch.WelcomeMessage) {
		s.log.Printf("WELCOME: subscribing to events...\n")
		clientID := data.GetTwitchClientID()
		accessToken := s.helixMain.GetUserAccessToken()

		for _, event := range s.subs {
			s.log.Printf("subscribing to %s\n", event)

			condition := map[string]string{
				"broadcaster_user_id": *s.mainUserId,
			}
			if event == twitch.SubChannelChatMessage {
				condition["user_id"] = *s.mainUserId
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
				s.log.Printf("Failed to subscribe to %s", event)
				hasSubError = true
			}
		}
		if hasSubError {
			s.log.Printf("There were issues when listening to Twitch events. Please refresh your Twitch token.")
		}
	})
	s.client.OnRevoke(func(message twitch.RevokeMessage) {
		s.log.Printf("REVOKE: %v\n", message)
	})
	s.setupHandlers()

	return s.client.ConnectWithContext(ctx)
}

func (s *TwitchWS) Client() *twitch.Client {
	return s.client
}

func (s *TwitchWS) Log() *log.Logger {
	return s.log
}

func NewTwitchWS(hc *helix.Client, mainUserId *string, mainTwitchChannel *string, helixBot *helix.Client, botUserId *string, botTwitchChannel *string, subs []twitch.EventSubscription, setupHandlers func()) *TwitchWS {
	s := &TwitchWS{
		botTwitchChannel:  botTwitchChannel,
		mainTwitchChannel: mainTwitchChannel,
		mainUserId:        mainUserId,
		botUserId:         botUserId,
		helixMain:         hc,
		log:               log.New(os.Stderr, "TWITCH_WS ", log.Ldate|log.Ltime),
		subs:              subs,
		setupHandlers:     setupHandlers,
	}

	return s
}
