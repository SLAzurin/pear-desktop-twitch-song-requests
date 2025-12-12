package appservices

import (
	"context"
	"log"
	"os"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/nicklaw5/helix/v2"

	"github.com/joeyak/go-twitch-eventsub/v3"
)

// mistake in the code, bot means secondary, main means main. so you will see the inverse for bot mode
type TwitchWS struct {
	botTwitchChannel  *string
	mainTwitchChannel *string
	mainUserId        *string
	botUserId         *string
	helixMain         *helix.Client
	helixBot          *helix.Client
	client            *twitch.Client
	log               *log.Logger
	subs              []twitch.EventSubscription
	setupHandlers     func()
	isBotMode         bool
}

func (s *TwitchWS) StartCtx(ctx context.Context) error {
	if s.isBotMode {
		s.log.Println("Twitch WS bot service starting...")
	} else {
		s.log.Println("Twitch WS main service starting...")
	}

	s.client = twitch.NewClient()
	hasSubError := false
	s.client.OnWelcome(func(message twitch.WelcomeMessage) {
		clientID := data.GetTwitchClientID()
		accessToken := s.helixMain.GetUserAccessToken()

		for _, event := range s.subs {
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
			s.log.Printf("There were issues when listening to Twitch events. Please refresh your Twitch token(s) and restart the app.")
		} else {
			if s.isBotMode {
				s.log.Println("Connected to Twitch as bot")
			} else {
				s.log.Println("Connected to Twitch as main account")
			}
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

func NewTwitchWS(hc *helix.Client, mainUserId *string, mainTwitchChannel *string, helixBot *helix.Client, botUserId *string, botTwitchChannel *string, subs []twitch.EventSubscription, setupHandlers func(), isBotMode bool) *TwitchWS {
	s := &TwitchWS{
		botTwitchChannel:  botTwitchChannel,
		mainTwitchChannel: mainTwitchChannel,
		mainUserId:        mainUserId,
		botUserId:         botUserId,
		helixMain:         hc,
		helixBot:          helixBot,
		log:               log.New(os.Stderr, "", log.Ldate|log.Ltime),
		subs:              subs,
		setupHandlers:     setupHandlers,
		isBotMode:         isBotMode,
	}

	return s
}
