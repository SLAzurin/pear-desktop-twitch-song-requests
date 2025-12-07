package songrequests

import (
	"github.com/joeyak/go-twitch-eventsub/v3"
)

var queueNextSong chan struct {
	VideoID string
}

var RcvMsgChan chan twitch.EventChannelChatMessage

type SongQueueAction = string

const (
	SongQueueActionAddSong    SongQueueAction = "ADD_SONG"
	SongQueueActionRemoveSong SongQueueAction = "REMOVE_SONG"
	SongQueueActionMoveSong   SongQueueAction = "MOVE_SONG"
)

type SongQueueInteraction struct {
	Action SongQueueAction
}

var SongQueueInteractionChan chan SongQueueInteraction

type SongRequestQueueItem struct {
	SongTitle string
	Artist    string
	VideoID   string
}

var SongRequestQueue []SongRequestQueueItem

func init() {
	queueNextSong = make(chan struct {
		VideoID string
	})

	RcvMsgChan = make(chan twitch.EventChannelChatMessage)

	SongQueueInteractionChan = make(chan SongQueueInteraction)

	SongRequestQueue = []SongRequestQueueItem{}
}
