package main

//lint:file-ignore ST1001 Dot imports by jet
import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/gen/model"
	. "github.com/azuridayo/pear-desktop-twitch-song-requests/gen/table"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/databaseconn"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/songrequests"
	"github.com/joeyak/go-twitch-eventsub/v3"
	"github.com/labstack/echo/v4"
	"github.com/nicklaw5/helix/v2"
)

var srChan = make(chan struct {
	song  *songrequests.SongResult
	event twitch.EventChannelChatMessage
})

func (a *App) songRequestLogic(song *songrequests.SongResult, event twitch.EventChannelChatMessage) {
	// Check if song ends <4s to prevent player state changes timing fkup
	a.safeLockMutexWaitForSongEnds(4)
	defer songQueueMutex.Unlock()

	var useProperHelix *helix.Client
	properUserID := ""
	if a.twitchDataStructBot.isAuthenticated {
		useProperHelix = a.helixBot
		properUserID = a.twitchDataStructBot.userID
	} else {
		useProperHelix = a.helix
		properUserID = a.twitchDataStruct.userID
	}

	for _, v := range songQueue {
		if song.VideoID == v.song.VideoID {
			// Song was added too fast, between internal api calls
			return
		}
	}

	b := echo.Map{
		"videoId":        song.VideoID,
		"insertPosition": "INSERT_AFTER_CURRENT_VIDEO",
	}
	bb, _ := json.Marshal(b)
	resp, err := http.Post("http://"+songrequests.GetPearDesktopHost()+"/api/v1/queue", "application/json", bytes.NewBuffer(bb))
	if err != nil || resp.StatusCode != http.StatusNoContent {
		emsg := "Internal error when adding song to queue. Disregard previous message."
		log.Println(emsg, err)
		useProperHelix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             properUserID,
			Message:              emsg,
			ReplyParentMessageID: event.MessageId,
		})
		return
	}
	if strings.EqualFold(event.BroadcasterUserLogin, a.twitchDataStructBot.login) {
		log.Println("hehe chatter " + event.ChatterUserLogin + ": Queued song " + song.Title + " - " + song.Artist)
	} else {
		log.Println(event.ChatterUserLogin + ": Queued song " + song.Title + " - " + song.Artist)
	}

	nowIndex := -1
	addedSongIndex := -1
	afterVideoIndex := -1
	afterVideoId := playerInfo.Song.VideoId
	if len(songQueue) > 0 {
		afterVideoId = songQueue[len(songQueue)-1].song.VideoID
	}
	songQueue = append(songQueue, struct {
		requestedBy string
		song        songrequests.SongResult
	}{
		requestedBy: event.ChatterUserLogin,
		song:        *song,
	})

	// save to history
	go func() {
		db, err := databaseconn.NewDBConnection()
		if err != nil {
			log.Println("Somehow failed to add !sr history to database")
			return
		}
		srData := model.SongRequests{
			VideoID:    song.VideoID,
			SongTitle:  song.Title,
			ArtistName: song.Artist,
			ImageURL:   song.ImageUrl,
		}
		stmt := SongRequests.INSERT(SongRequests.AllColumns).MODEL(srData).ON_CONFLICT(SongRequests.VideoID).DO_NOTHING()
		_, err = stmt.Exec(db)
		if err != nil {
			log.Println("Somehow failed to save !sr song to database")
			return
		}
		srrData := model.SongRequestRequesters{
			VideoID:        song.VideoID,
			TwitchUsername: event.ChatterUserLogin,
			RequestedAt:    time.Now().Local().Format(data.TWITCH_SERVER_DATE_LAYOUT),
		}
		stmt = SongRequestRequesters.INSERT(SongRequestRequesters.AllColumns).MODEL(srrData)
		_, err = stmt.Exec(db)
		if err != nil {
			log.Println("Somehow failed to save !sr requester name to database")
			return
		}
	}()

	// Fetch new q details
	// Get q info
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

	timeout := time.After(time.Second * 10)
OuterLoop:
	for {
		time.Sleep(time.Millisecond * 500)
		select {
		case <-timeout:
			break OuterLoop
		default:
			resp, err := http.Get("http://" + songrequests.GetPearDesktopHost() + "/api/v1/queue")
			if err != nil || resp.StatusCode != http.StatusOK {
				emsg := "Internal error when checking if song is already in queue. Disregard previous message."
				log.Println(emsg, err)
				useProperHelix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             properUserID,
					Message:              emsg,
					ReplyParentMessageID: event.MessageId,
				})
				return
			}
			qb, err := io.ReadAll(resp.Body)
			if err != nil {
				emsg := "Internal error processing data to check if song is already in queue. Disregard previous message."
				log.Println(emsg, err)
				useProperHelix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             properUserID,
					Message:              emsg,
					ReplyParentMessageID: event.MessageId,
				})
				return
			}
			err = json.Unmarshal(qb, &queue)
			resp.Body.Close()
			if err != nil {
				emsg := event.Broadcaster.BroadcasterUserLogin + " Failed to check queue order. Must fix the song order manually!"
				log.Println(emsg, err)
				useProperHelix.SendChatMessage(&helix.SendChatMessageParams{
					BroadcasterID:        event.BroadcasterUserId,
					SenderID:             properUserID,
					Message:              emsg,
					ReplyParentMessageID: event.MessageId,
				})
				return
			}
			nowIndex = -1
			addedSongIndex = -1
			afterVideoIndex = -1
			for i, v := range queue.Items {
				if v.PlaylistPanelVideoRenderer.Selected {
					nowIndex = i
				}
				if nowIndex == -1 {
					continue
				}
				if nowIndex != -1 && afterVideoId == v.PlaylistPanelVideoRenderer.VideoId {
					afterVideoIndex = i
				}
				if nowIndex != -1 && song.VideoID == v.PlaylistPanelVideoRenderer.VideoId {
					addedSongIndex = i
				}
				if afterVideoIndex != -1 && addedSongIndex != -1 {
					break
				}
			}
			if nowIndex != -1 && addedSongIndex != -1 && afterVideoIndex != -1 {
				break OuterLoop
			}
		}
	}

	// get song index & drag song down to wherever is needed
	if nowIndex == -1 || addedSongIndex == -1 || afterVideoIndex == -1 {
		useProperHelix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             properUserID,
			Message:              event.Broadcaster.BroadcasterUserLogin + " Failed to queue song in the right order. Must fix the song order manually!",
			ReplyParentMessageID: event.MessageId,
		})
		return
	}

	// Drag song into the right order
	if afterVideoIndex+1 == addedSongIndex {
		// do not move anything
		return
	}
	b2, _ := json.Marshal(echo.Map{
		"toIndex": afterVideoIndex,
	})
	req, _ := http.NewRequest(http.MethodPatch, "http://"+songrequests.GetPearDesktopHost()+"/api/v1/queue/"+strconv.Itoa(addedSongIndex), bytes.NewBuffer(b2))
	req.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil || resp2.StatusCode != http.StatusNoContent {
		useProperHelix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             properUserID,
			Message:              event.Broadcaster.BroadcasterUserLogin + " Failed to move song in the right order. Must fix the song order manually!",
			ReplyParentMessageID: event.MessageId,
		})
		return
	}
	// Already replied to chatter song is alr added to q
}

func (a *App) safeLockMutexWaitForSongEnds(underTimeInSeconds int) {
	songQueueMutex.Lock()
	if playerInfo.IsPlaying && playerInfo.Song.SongDuration-playerInfo.Position <= underTimeInSeconds {
		currentVideoId := playerInfo.Song.VideoId
		// This unlock relock allows for <1s remaining time check

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
		timeout := time.After(time.Duration(underTimeInSeconds+10) * time.Second) // give extra 10 seconds buffer in case of api delay
		for {
			time.Sleep(200 * time.Millisecond)
			select {
			case <-timeout:
				return
			default:
				shouldBreak := false
				resp, err := http.Get("http://" + songrequests.GetPearDesktopHost() + "/api/v1/queue")
				if err != nil || resp.StatusCode != http.StatusOK {
					continue
				}
				qb, err := io.ReadAll(resp.Body)
				if err != nil {
					continue
				}
				err = json.Unmarshal(qb, &queue)
				resp.Body.Close()
				if err != nil {
					continue
				}

				for _, v := range queue.Items {
					if v.PlaylistPanelVideoRenderer.Selected && v.PlaylistPanelVideoRenderer.VideoId != currentVideoId {
						shouldBreak = true
						break
					}
					if v.PlaylistPanelVideoRenderer.Selected {
						break
					}
				}

				if shouldBreak {
					return
				}
			}
		}
	}
}
