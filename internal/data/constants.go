package data

import "time"

var twitchClientID = "7k7nl6w8e0owouonj7nb9g3k5s6gs5"

func GetTwitchClientID() string {
	return twitchClientID
}

const (
	DB_KEY_TWITCH_ACCESS_TOKEN           = "twitch_access_token"
	DB_KEY_TWITCH_ACCESS_TOKEN_BOT       = "twitch_access_token_bot"
	DB_KEY_TWITCH_SONG_REQUEST_REWARD_ID = "twitch_song_request_reward_id"
	TWITCH_SERVER_DATE_LAYOUT            = time.RFC1123
)
