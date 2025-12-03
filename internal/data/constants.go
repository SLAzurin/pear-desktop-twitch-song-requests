package data

var twitchClientID = "7k7nl6w8e0owouonj7nb9g3k5s6gs5"
var pearDesktopHostname = "localhost"
var pearDesktopPort = "26538"

func GetTwitchClientID() string {
	return twitchClientID
}

func GetPearDesktopHost() string {
	return pearDesktopHostname + ":" + pearDesktopPort
}
