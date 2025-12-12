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
	"github.com/nicklaw5/helix/v2"
)

var checkMainChannelUserStatusMutex = sync.RWMutex{}
var checkMainChannelUserStatus = map[string]struct {
	isSub       bool
	isModerator bool
	timeExpiry  time.Time
}{}

func (a *App) SetSubscriptionHandlersBot() {
	a.twitchWSBotService.Client().OnEventChannelChatMessage(func(event twitch.EventChannelChatMessage) {
		isSub := false
		isBroadcaster := false
		isModerator := false
		useProperHelix := a.helixBot
		properUserID := a.twitchDataStructBot.userID
		realBroadcasterID := a.twitchDataStruct.userID

		if strings.EqualFold(event.ChatterUserLogin, a.twitchDataStruct.login) {
			isSub = true
			isBroadcaster = true
			isModerator = true
		} else {
			checkMainChannelUserStatusMutex.RLock()
			if v, ok := checkMainChannelUserStatus[event.ChatterUserLogin]; ok && !time.Now().After(v.timeExpiry) {
				isSub = v.isSub
				isModerator = v.isModerator
				checkMainChannelUserStatusMutex.RUnlock()
			} else {
				checkMainChannelUserStatusMutex.RUnlock()

				subsResponse, err := a.helix.GetSubscriptions(&helix.SubscriptionsParams{
					UserID:        []string{event.ChatterUserId},
					BroadcasterID: realBroadcasterID,
				})
				if err != nil {
					emsg := "Internal error when checking if you are a sub"
					log.Println(emsg, err)
					a.helixBot.SendChatMessage(&helix.SendChatMessageParams{
						BroadcasterID:        event.BroadcasterUserId,
						SenderID:             properUserID,
						Message:              emsg,
						ReplyParentMessageID: event.MessageId,
					})
					return
				}
				if len(subsResponse.Data.Subscriptions) > 0 {
					isSub = true
				}

				modsResponse, err := a.helix.GetModerators(&helix.GetModeratorsParams{
					UserIDs:       []string{event.ChatterUserId},
					BroadcasterID: realBroadcasterID,
				})
				if err != nil {
					emsg := "Internal error when checking if you are a moderator"
					log.Println(emsg, err)
					a.helixBot.SendChatMessage(&helix.SendChatMessageParams{
						BroadcasterID:        event.BroadcasterUserId,
						SenderID:             properUserID,
						Message:              emsg,
						ReplyParentMessageID: event.MessageId,
					})
					return
				}
				if len(modsResponse.Data.Moderators) > 0 {
					isSub = true
					isModerator = true
				}

				checkMainChannelUserStatusMutex.Lock()
				checkMainChannelUserStatus[event.ChatterUserLogin] = struct {
					isSub       bool
					isModerator bool
					timeExpiry  time.Time
				}{
					isSub:       isSub,
					isModerator: isModerator,
					timeExpiry:  time.Now().Add(time.Hour * 2),
				}
				checkMainChannelUserStatusMutex.Unlock()
			}
		}

		if strings.HasPrefix(event.Message.Text, "!sr ") && isSub {
			if !a.streamOnline && !isBroadcaster {
				return
			}
			a.songRequestSubmit(useProperHelix, properUserID, event)
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
				a.helixBot.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             properUserID,
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
			currentSongMutexBot.Lock()
			if time.Now().After(lastUsedCurrentSongBot.Add(time.Second * -10)) {
				lastUsedCurrentSongBot = time.Now()
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
			currentSongMutexBot.Unlock()
			if failed {
				log.Println("Failed to get song info from !song", rootErr)
				a.helixBot.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             properUserID,
					Message:              "Internal failure to get song details!",
					ReplyParentMessageID: event.MessageId,
				})
			} else {
				a.helixBot.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             properUserID,
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
			queueCmdMutexBot.Lock()
			if time.Now().After(lastUsedQueueCmdBot.Add(time.Second * -10)) {
				lastUsedQueueCmdBot = time.Now()
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
			queueCmdMutexBot.Unlock()
			if failed {
				log.Println("Failed to get queue info from !queue", rootErr)
				a.helixBot.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             properUserID,
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

				a.helixBot.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             properUserID,
					Message:              s,
					ReplyParentMessageID: event.MessageId,
				})
			}
			return
		}
	})
}

var currentSongMutexBot = sync.Mutex{}
var lastUsedCurrentSongBot = time.Now().Add(time.Second * -10)

var queueCmdMutexBot = sync.Mutex{}
var lastUsedQueueCmdBot = time.Now().Add(time.Second * -10)
