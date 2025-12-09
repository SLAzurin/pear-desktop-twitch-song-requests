package main

import (
	"sync"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/songrequests"
)

var songQueueMutex sync.RWMutex

var songQueue = []struct {
	requestedBy string
	song        songrequests.SongResult
}{}

type playerSonginfo struct {
	ImageSrc     string `json:"imageSrc"`
	Artist       string `json:"artist"`
	SongDuration string `json:"songDuration"`
	Title        string `json:"title"`
	Url          string `json:"url"`
}

var playerInfo = struct {
	Position  int
	IsPlaying bool
	Song      playerSonginfo `json:"song"`
}{
	Song: playerSonginfo{},
}
