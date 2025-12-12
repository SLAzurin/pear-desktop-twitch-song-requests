package songrequests

import (
	"github.com/joeyak/go-twitch-eventsub/v3"
)

func GetSubscriptions() []twitch.EventSubscription {
	events := []twitch.EventSubscription{
		twitch.SubStreamOnline,
		twitch.SubStreamOffline,
		twitch.SubChannelChatMessage,
		twitch.SubChannelChannelPointsCustomRewardRedemptionAdd, // claim reward points
	}

	return events
}

func GetSubscriptionsBot() []twitch.EventSubscription {
	events := []twitch.EventSubscription{
		twitch.SubChannelChatMessage,
	}

	return events
}
