package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/songrequests"
	"github.com/joeyak/go-twitch-eventsub/v3"
	"github.com/nicklaw5/helix/v2"
)

func (a *App) songRequestSubmit(useProperHelix *helix.Client, properUserID string, event twitch.EventChannelChatMessage) {
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
		useProperHelix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             properUserID,
			Message:              emsg,
			ReplyParentMessageID: event.MessageId,
		})
		return
	}
	qb, err := io.ReadAll(preResponse.Body)
	if err != nil {
		emsg := "Internal error processing data to check if song is already in queue"
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
	preResponse.Body.Close()
	if err != nil {
		emsg := "Failed to check if song exists in queue."
		log.Println(emsg, err)
		useProperHelix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             properUserID,
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
		useProperHelix.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID:        event.BroadcasterUserId,
			SenderID:             properUserID,
			Message:              msg,
			ReplyParentMessageID: event.MessageId,
		})
		return
	}

	// Committing to adding song to q
	useProperHelix.SendChatMessage(&helix.SendChatMessageParams{
		BroadcasterID:        event.BroadcasterUserId,
		SenderID:             properUserID,
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
}
