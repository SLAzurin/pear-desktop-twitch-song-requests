package main

import (
	"log"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/songrequests"
	"github.com/valyala/fastjson"
)

func (a *App) handlePearDesktopMsgs() {
	var p fastjson.Parser
	for {
		select {
		case <-a.ctx.Done():
			return
		case msg := <-a.pearDesktopIncomingMsgs:
			v, err := p.ParseBytes(msg)
			if err != nil {
				log.Printf("Received non-json: %s\n", msg)
				continue
			}
			msgType := string(v.GetStringBytes("type"))
			switch msgType {
			case "POSITION_CHANGED":
				playerInfoMutex.Lock()
				playerInfo.Position = v.GetInt("position")
				playerInfoMutex.Unlock()
			case "PLAYER_INFO":
				playerInfoMutex.Lock()
				playerInfo.IsPlaying = v.GetBool("isPlaying")
				playerInfo.Position = v.GetInt("position")
				songinfo := playerSonginfo{
					ImageSrc:         string(v.GetStringBytes("song", "imageSrc")),
					Artist:           string(v.GetStringBytes("song", "artist")),
					SongDuration:     v.GetInt("song", "songDuration"),
					AlternativeTitle: string(v.GetStringBytes("song", "alternativeTitle")),
					VideoId:          string(v.GetStringBytes("song", "videoId")),
				}
				playerInfo.Song = songinfo
				playerInfoMutex.Unlock()
			case "VIDEO_CHANGED":
				playerInfoMutex.Lock()
				songQueueMutex.Lock()
				newVideoId := string(v.GetStringBytes("song", "videoId"))
				playerInfo.Position = v.GetInt("position")
				if playerInfo.Song.VideoId != newVideoId {
					songinfo := playerSonginfo{
						ImageSrc:         string(v.GetStringBytes("song", "imageSrc")),
						Artist:           string(v.GetStringBytes("song", "artist")),
						SongDuration:     v.GetInt("song", "songDuration"),
						AlternativeTitle: string(v.GetStringBytes("song", "alternativeTitle")),
						VideoId:          newVideoId,
					}
					playerInfo.Song = songinfo
					if len(songQueue) > 1 {
						songQueue = songQueue[1:]
					}
					if len(songQueue) > 1 && songQueue[0].song.VideoID != newVideoId {
						// queue invalid now, wiping queue
						songQueue = []struct {
							requestedBy string
							song        songrequests.SongResult
						}{}
						log.Println("queue was wiped because it was out of sync with Pear Desktop")
					}
				}
				playerInfoMutex.Unlock()
				songQueueMutex.Unlock()
			case "PLAYER_STATE_CHANGED":
				playerInfoMutex.Lock()
				playerInfo.Position = v.GetInt("position")
				playerInfo.IsPlaying = v.GetBool("isPlaying")
				playerInfoMutex.Unlock()
			default:
				// Nothing, ignore non important
			}
		}
	}
}
