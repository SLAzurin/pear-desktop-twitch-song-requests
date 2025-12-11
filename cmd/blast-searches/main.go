package main

import (
	"log"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/songrequests"
)

func main() {
	songs := []string{
		"!sr yena nemonemo",
		"!sr yena good morning",
		"!sr yena being a good girl hurts",
		"!sr yena smartphone",
	}
	for _, v := range songs {
		v = songrequests.ParseSearchQuery(v)
		song, err := songrequests.SearchSong(v, 60, 600)
		if err != nil {
			panic(err)
		}
		log.Println(song.Title, song.SearchOrigin, song.ImageUrl)
	}
}
