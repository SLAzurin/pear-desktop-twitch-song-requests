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
	VideoId          string `json:"videoId"`
	ImageSrc         string `json:"imageSrc"`
	Artist           string `json:"artist"`
	SongDuration     int    `json:"songDuration"`
	AlternativeTitle string `json:"alternativeTitle"`
}

func (s playerSonginfo) GetUrl() string {
	return "https://youtu.be/" + s.VideoId
}

var playerInfo = struct {
	Position  int
	IsPlaying bool
	Song      playerSonginfo `json:"song"`
}{
	Song: playerSonginfo{},
}
