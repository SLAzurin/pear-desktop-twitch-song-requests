package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
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
		isSub := false
		isBroadcaster := false
		isModerator := false
		for _, v := range event.Badges {
			if v.SetId == "subscriber" {
				isSub = true
			}
			if v.SetId == "broadcaster" {
				isBroadcaster = true
				isModerator = true
				isSub = true
			}
			if v.SetId == "moderator" {
				isModerator = true
				isSub = true
			}
		}

		log.Printf("Chat message from %s: %s %s\n", event.ChatterUserLogin, event.Message.Text, event.ChannelPointsCustomRewardId)
		if a.songRequestRewardID == event.ChannelPointsCustomRewardId || (strings.HasPrefix(event.Message.Text, "!sr ") && isSub) {
			if !a.streamOnline && !isBroadcaster {
				return
			}
			s := songrequests.ParseSearchQuery(event.Message.Text)
			song, err := songrequests.SearchSong(s, 60, 600)
			if err != nil {
				return
			}

			// Loop through queue state to check if song is queued already
			queue := struct {
				Items []struct {
					PlaylistPanelVideoRenderer struct {
						VideoId         string `json:"videoId"`
						Selected        bool   `json:"selected"`
						ShortByLineText struct {
							Runs []struct {
								Text string `json:"text"`
							} `json:"runs"`
						} `json:"shortByLineText"`
						Title struct {
							Runs []struct {
								Text string `json:"text"`
							} `json:"runs"`
						} `json:"title"`
						NavigationEndpoint struct {
							WatchEndpoint struct {
								Index int `json:"index"`
							} `json:"watchEndpoint"`
						} `json:"navigationEndpoint"`
					} `json:"playlistPanelVideoRenderer"`
				} `json:"items"`
			}{}

			preResponse, err := http.Get("http://" + songrequests.GetPearDesktopHost() + "/api/v1/queue")
			if err != nil || preResponse.StatusCode != http.StatusOK {
				emsg := "Internal error when checking if song is already in queue"
				log.Println(emsg, err)
				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              emsg,
					ReplyParentMessageID: event.MessageId,
				})
				return
			}
			qb, err := io.ReadAll(preResponse.Body)
			if err != nil {
				emsg := "Internal error processing data to check if song is already in queue"
				log.Println(emsg, err)
				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              emsg,
					ReplyParentMessageID: event.MessageId,
				})
				return
			}
			err = json.Unmarshal(qb, &queue)
			preResponse.Body.Close()
			if err != nil {
				emsg := "Failed to check if song exists in queue."
				log.Println(emsg, err)
				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              emsg,
					ReplyParentMessageID: event.MessageId,
				})
				return
			}

			afterSelected := false
			songExistsInQueue := false
			for _, v := range queue.Items {
				if v.PlaylistPanelVideoRenderer.Selected {
					afterSelected = true
				}
				if afterSelected && song.VideoID == v.PlaylistPanelVideoRenderer.VideoId {
					songExistsInQueue = true
					break
				}
			}

			if songExistsInQueue {
				msg := "Song is already in queue!"
				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              msg,
					ReplyParentMessageID: event.MessageId,
				})
				return
			}

			// Committing to adding song to q
			a.helix.SendChatMessage(&helix.SendChatMessageParams{
				BroadcasterID:        event.BroadcasterUserId,
				SenderID:             a.twitchDataStruct.userID,
				Message:              "Added song: " + song.Title + " - " + song.Artist + " " + "https://youtu.be/" + song.VideoID,
				ReplyParentMessageID: event.MessageId,
			})
			srChan <- struct {
				song  *songrequests.SongResult
				event twitch.EventChannelChatMessage
			}{
				song:  song,
				event: event,
			}
			return
		}

		if strings.HasPrefix(event.Message.Text, "!skip") && isModerator {
			if !a.streamOnline && !isBroadcaster {
				return
			}
			hasSkipped := false
			skipMutex.Lock()
			if time.Now().After(lastSkipped.Add(time.Second * -10)) {
				hasSkipped = true
				songQueueMutex.Lock()
				http.Post("http://"+songrequests.GetPearDesktopHost()+"/api/v1/next", "application/json", nil)
				songQueueMutex.Unlock()
				lastSkipped = time.Now()
			}
			skipMutex.Unlock()
			if hasSkipped {
				s := "Skipped song!"
				if songQueueMutex.TryRLock() {
					s = "Skipped " + playerInfo.Song.AlternativeTitle + "!"
					songQueueMutex.RUnlock()
				}
				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              s,
					ReplyParentMessageID: event.MessageId,
				})
			}
			return
		}

		if strings.HasPrefix(event.Message.Text, "!song") {
			if !a.streamOnline && !isBroadcaster {
				return
			}
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

		if strings.HasPrefix(event.Message.Text, "!queue") {
			if !a.streamOnline && !isBroadcaster {
				return
			}
			failed := false
			queue := struct {
				Items []struct {
					PlaylistPanelVideoRenderer struct {
						VideoId         string `json:"videoId"`
						Selected        bool   `json:"selected"`
						ShortByLineText struct {
							Runs []struct {
								Text string `json:"text"`
							} `json:"runs"`
						} `json:"shortByLineText"`
						// LongBylineText struct {
						// 	Runs []struct {
						// 		Text string `json:"text"`
						// 	} `json:"runs"`
						// } `json:"longBylineText"`
						Title struct {
							Runs []struct {
								Text string `json:"text"`
							} `json:"runs"`
						} `json:"title"`
					} `json:"playlistPanelVideoRenderer"`
				} `json:"items"`
			}{}
			var rootErr error = nil
			queueCmdMutex.Lock()
			if time.Now().After(lastUsedQueueCmd.Add(time.Second * -10)) {
				lastUsedQueueCmd = time.Now()
				resp, err := http.Get("http://" + songrequests.GetPearDesktopHost() + "/api/v1/queue")
				if err == nil {
					bb, err := io.ReadAll(resp.Body)
					if err == nil {
						rootErr = json.Unmarshal(bb, &queue)
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
			queueCmdMutex.Unlock()
			if failed {
				log.Println("Failed to get queue info from !queue", rootErr)
				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              "Internal failure to get queue detail!",
					ReplyParentMessageID: event.MessageId,
				})
			} else {
				s := "Now: "
				n := 0
				foundSelected := false
				for _, v := range queue.Items {
					if v.PlaylistPanelVideoRenderer.Selected {
						foundSelected = true
					}
					if !v.PlaylistPanelVideoRenderer.Selected && !foundSelected {
						continue
					}
					if n > 5 {
						break
					}
					n++
					title := v.PlaylistPanelVideoRenderer.Title.Runs[0].Text
					artist := v.PlaylistPanelVideoRenderer.ShortByLineText.Runs[0].Text
					sl := "#" + strconv.Itoa(n-1) + ": " + title + " - " + artist + ", "
					if n == 1 {
						sl = strings.TrimPrefix(sl, "#"+strconv.Itoa(n-1)+": ")
					}
					s += sl
				}
				s = strings.TrimSuffix(s, ", ")

				a.helix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             a.twitchDataStruct.userID,
					Message:              s,
					ReplyParentMessageID: event.MessageId,
				})
			}
			return
		}
	})
}

var skipMutex = sync.Mutex{}
var lastSkipped = time.Now().Add(time.Second * -10)

var currentSongMutex = sync.Mutex{}
var lastUsedCurrentSong = time.Now().Add(time.Second * -10)

var queueCmdMutex = sync.Mutex{}
var lastUsedQueueCmd = time.Now().Add(time.Second * -10)
