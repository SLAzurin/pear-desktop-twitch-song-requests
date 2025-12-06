package data

var twitchClientID = "7k7nl6w8e0owouonj7nb9g3k5s6gs5"

func GetTwitchClientID() string {
	return twitchClientID
}

const (
	DB_KEY_TWITCH_ACCESS_TOKEN = "twitch_access_token"
	DB_KEY_TWITCH_LOGIN        = "twitch_login"
	DB_KEY_TWITCH_USER_ID      = "twitch_user_id"

	TWITCH_SERVER_DATE_LAYOUT = "Mon, 02 Jan 2006 15:04:05 MST"
)
