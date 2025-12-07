package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/songrequests"
	"github.com/joeyak/go-twitch-eventsub/v3"
	"github.com/labstack/echo/v4"
	"github.com/nicklaw5/helix/v2"
)

func (a *App) SetSubscriptionHandlers() {
	a.twitchWSService.Client().OnEventStreamOnline(func(event twitch.EventStreamOnline) {
		a.streamOnline = true

		j, _ := json.Marshal(echo.Map{
			"stream_online": true,
		})
		a.clientsBroadcast <- string(j)
		log.Println("STREAM_ONLINE")
	})
	a.twitchWSService.Client().OnEventStreamOffline(func(event twitch.EventStreamOffline) {
		a.streamOnline = false
		j, _ := json.Marshal(echo.Map{
			"stream_online": false,
		})
		a.clientsBroadcast <- string(j)
		log.Println("STREAM_OFFLINE")
	})
	a.twitchWSService.Client().OnEventChannelChatMessage(func(event twitch.EventChannelChatMessage) {
		// Song request
		isSub := false
		isBroadcaster := false
		isModerator := false
		for _, v := range event.Badges {
			if v.SetId == "subscriber" {
				isSub = true
			}
			if v.SetId == "broadcaster" {
				isBroadcaster = true
			}
			if v.SetId == "moderator" {
				isModerator = true
				isSub = true
			}
		}

		log.Printf("Chat message from %s: %s %s\n", event.ChatterUserLogin, event.Message.Text, event.ChannelPointsCustomRewardId)
		if a.songRequestRewardID == event.ChannelPointsCustomRewardId || (strings.HasPrefix(event.Message.Text, "!sr ") && isSub) {
			if a.streamOnline || isBroadcaster {
				s := songrequests.ParseSearchQuery(event.Message.Text)
				song, err := songrequests.SearchSong(s, 600)
				if err != nil {
					a.helix.SendChatMessage(&helix.SendChatMessageParams{
						BroadcasterID:        event.BroadcasterUserId,
						SenderID:             a.twitchDataStruct.userID,
						Message:              err.Error(),
						ReplyParentMessageID: event.MessageId,
					})
					return
				}
				b := echo.Map{
					"videoId":        song.VideoID,
					"insertPosition": "INSERT_AFTER_CURRENT_VIDEO",
				}
				bb, _ := json.Marshal(b)
				http.Post("http://"+songrequests.GetPearDesktopHost()+"/api/v1/queue", "application/json", bytes.NewBuffer(bb))
				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              "Added song: " + song.Title + " - " + song.Artist + " " + "https://youtu.be/" + song.VideoID,
					ReplyParentMessageID: event.MessageId,
				})
			} else {
				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              "Stream is offline. If this is a mistake, please ask streamer to reboot the song requests app!",
					ReplyParentMessageID: event.MessageId,
				})
			}
			return
		}

		if strings.HasPrefix(event.Message.Text, "!skip") && isModerator {
			hasSkipped := false
			skipMutex.Lock()
			if time.Now().After(lastSkipped.Add(time.Second * -5)) {
				hasSkipped = true
				http.Post("http://"+songrequests.GetPearDesktopHost()+"/api/v1/next", "application/json", nil)
				lastSkipped = time.Now()
			}
			skipMutex.Unlock()
			if hasSkipped {
				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              "Skipped song!",
					ReplyParentMessageID: event.MessageId,
				})
			}
			return
		}

		if strings.HasPrefix(event.Message.Text, "!song") {
			failed := false
			song := songrequests.SongResult{}
			var rootErr error = nil
			currentSongMutex.Lock()
			if time.Now().After(lastUsedCurrentSong.Add(time.Second * -10)) {
				lastUsedCurrentSong = time.Now()
				resp, err := http.Get("http://" + songrequests.GetPearDesktopHost() + "/api/v1/song")
				if err == nil {
					bb, err := io.ReadAll(resp.Body)
					if err == nil {
						rootErr = json.Unmarshal(bb, &song)
						if rootErr != nil {
							failed = true
						}
					} else {
						failed = true
						rootErr = err
					}
				} else {
					failed = true
					rootErr = err
				}
			}
			currentSongMutex.Unlock()
			if failed {
				log.Println("Failed to get song info from !song", rootErr)
				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              "Internal failure to get song details!",
					ReplyParentMessageID: event.MessageId,
				})
			} else {
				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              "Song: " + song.Title + " - " + song.Artist + " https://youtu.be/" + song.VideoID,
					ReplyParentMessageID: event.MessageId,
				})
			}
			return
		}
	})
}

var skipMutex = sync.Mutex{}
var lastSkipped = time.Now().Add(time.Second * -5)

var currentSongMutex = sync.Mutex{}
var lastUsedCurrentSong = time.Now().Add(time.Second * -5)
