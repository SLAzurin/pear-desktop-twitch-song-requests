package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/songrequests"
	"github.com/joeyak/go-twitch-eventsub/v3"
	"github.com/labstack/echo/v4"
	"github.com/nicklaw5/helix/v2"
)

func (a *App) songRequestLogic(q string, event twitch.EventChannelChatMessage) error {
	s := songrequests.ParseSearchQuery(q)
	song, err := songrequests.SearchSong(s, 60, 600)
	if err != nil {
		return err
	}

	// Loop through queue state to check if song is queued already
	songQueueMutex.RLock()
	songExistsInQueue := false
	for _, v := range songQueue {
		if v.song.VideoID == song.VideoID {
			songExistsInQueue = true
			break
		}
	}
	songQueueMutex.RUnlock()
	if songExistsInQueue {
		msg := "Song is already in queue!"
		a.helix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             a.twitchDataStruct.userID,
			Message:              msg,
			ReplyParentMessageID: event.MessageId,
		})
		return nil
	}

	// Committing to adding song to q
	a.helix.SendChatMessage(&helix.SendChatMessageParams{
		BroadcasterID:        event.BroadcasterUserId,
		SenderID:             a.twitchDataStruct.userID,
		Message:              "Added song: " + song.Title + " - " + song.Artist + " " + "https://youtu.be/" + song.VideoID,
		ReplyParentMessageID: event.MessageId,
	})

	go a.commitAddSongToQueue(song, event)

	return nil

}

func (a *App) commitAddSongToQueue(song *songrequests.SongResult, event twitch.EventChannelChatMessage) {
	// Check if song ends <4s to prevent player state changes timing fkup
	playerInfoMutex.RLock()
	if playerInfo.IsPlaying && playerInfo.Song.SongDuration-playerInfo.Position <= 4 {
		currentVideoId := playerInfo.Song.VideoId
		playerInfoMutex.RUnlock()
		// This unlock relock allows for <1s remaining time check
		playerInfoMutex.RLock()
		for playerInfo.IsPlaying && currentVideoId == playerInfo.Song.VideoId && !(playerInfo.Position <= 5) {
			playerInfoMutex.RUnlock()
			time.Sleep(200 * time.Millisecond)
			playerInfoMutex.RLock()
		}
		if !playerInfo.IsPlaying {
			playerInfoMutex.RUnlock()
			return
		}
	}
	playerInfoMutex.RUnlock()
	// Finally done lock unlock timing checks

	// Actually put song in queue
	songQueueMutex.Lock()
	b := echo.Map{
		"videoId":        song.VideoID,
		"insertPosition": "INSERT_AFTER_CURRENT_VIDEO",
	}
	bb, _ := json.Marshal(b)
	resp, err := http.Post("http://"+songrequests.GetPearDesktopHost()+"/api/v1/queue", "application/json", bytes.NewBuffer(bb))
	if err != nil || resp.StatusCode != http.StatusNoContent {
		emsg := "Internal error when adding song to queue. Disregard previous message."
		log.Println(emsg, err)
		a.helix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             a.twitchDataStruct.userID,
			Message:              emsg,
			ReplyParentMessageID: event.MessageId,
		})
		songQueueMutex.Unlock()
		return
	}
	songQueue = append(songQueue, struct {
		requestedBy string
		song        songrequests.SongResult
	}{
		requestedBy: event.ChatterUserLogin,
		song:        *song,
	})
	defer songQueueMutex.Unlock()

	// save to history
	// TODO

	// Fetch new q details
	// Get q info
	resp, err = http.Get("http://" + songrequests.GetPearDesktopHost() + "/api/v1/queue")
	if err != nil || resp.StatusCode != http.StatusOK {
		emsg := "Internal error when checking if song is already in queue. Disregard previous message."
		log.Println(emsg, err)
		a.helix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             a.twitchDataStruct.userID,
			Message:              emsg,
			ReplyParentMessageID: event.MessageId,
		})
		return
	}
	qb, err := io.ReadAll(resp.Body)
	if err != nil {
		emsg := "Internal error processing data to check if song is already in queue. Disregard previous message."
		log.Println(emsg, err)
		a.helix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             a.twitchDataStruct.userID,
			Message:              emsg,
			ReplyParentMessageID: event.MessageId,
		})
		return
	}
	defer resp.Body.Close()
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

	err = json.Unmarshal(qb, &queue)
	if err != nil {
		emsg := "Failed to check queue order. Must fix the song order manually!"
		log.Println(emsg, err)
		a.helix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             a.twitchDataStruct.userID,
			Message:              emsg,
			ReplyParentMessageID: event.MessageId,
		})
		return
	}

	// get song index & drag song down to wherever is needed
	nowIndex := -1
	for _, v := range queue.Items {
		if v.PlaylistPanelVideoRenderer.Selected {
			nowIndex = v.PlaylistPanelVideoRenderer.NavigationEndpoint.WatchEndpoint.Index
			break
		}
		// if song.VideoID == v.PlaylistPanelVideoRenderer.VideoId {
		// 	addedSongIndex = v.PlaylistPanelVideoRenderer.NavigationEndpoint.WatchEndpoint.Index
		// 	break
		// }
	}
	// log.Println(nowIndex, addedSongIndex)
	// path1 := filepath.Join("response_queue.json")
	// os.WriteFile(path1, qb, 0644)

	if nowIndex == -1 {
		a.helix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             a.twitchDataStruct.userID,
			Message:              "Failed to queue song in the right order. Must fix the song order manually!",
			ReplyParentMessageID: event.MessageId,
		})
		return
	}

	// Drag song into the right order
	nowIndex += len(songQueue)

	b2, _ := json.Marshal(echo.Map{
		"toIndex": nowIndex + 1 + len(songQueue),
	})
	req, _ := http.NewRequest(http.MethodPatch, "http://"+songrequests.GetPearDesktopHost()+"/api/v1/queue/"+strconv.Itoa(nowIndex+1), bytes.NewBuffer(b2))
	req.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil || resp2.StatusCode != http.StatusNoContent {
		a.helix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             a.twitchDataStruct.userID,
			Message:              "Failed to move song in the right order. Must fix the song order manually!",
			ReplyParentMessageID: event.MessageId,
		})
		return
	}
	// Already replied to chatter song is alr added to q
}
