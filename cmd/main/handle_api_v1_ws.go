package main

//lint:file-ignore ST1001 Dot imports by jet
import (
	"encoding/json"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

func (a *App) handleAppWs(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		// Add client to the map
		a.clientsMu.Lock()
		a.clients[ws] = struct{}{}
		a.clientsMu.Unlock()

		defer func() {
			a.clientsMu.Lock()
			delete(a.clients, ws)
			a.clientsMu.Unlock()
		}()

		// Send initial info
		// only login and expiry date
		expiryDate := ""
		if a.twitchDataStruct.login != "" {
			expiryDate = a.twitchDataStruct.expiresDate.Local().Format(data.TWITCH_SERVER_DATE_LAYOUT)
		}

		expiryDateBot := ""
		if a.twitchDataStructBot.login != "" {
			expiryDate = a.twitchDataStructBot.expiresDate.Local().Format(data.TWITCH_SERVER_DATE_LAYOUT)
		}

		infoOnConnect, _ := json.Marshal(echo.Map{
			"type":            "TWITCH_INFO",
			"login":           a.twitchDataStruct.login,
			"expiry_date":     expiryDate,
			"stream_online":   a.streamOnline,
			"reward_id":       a.songRequestRewardID,
			"login_bot":       a.twitchDataStructBot.login,
			"expiry_date_bot": expiryDateBot,
		})
		err := websocket.Message.Send(ws, string(infoOnConnect))
		if err != nil {
			// conn already closed
			return
		}

		// Keep connection alive and handle any incoming messages
		for {
			msg := ""
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				// This break marks the ws closure
				break
			}
			// We don't handle incoming messages from frontend ever
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}
