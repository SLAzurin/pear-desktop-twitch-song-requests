package main

//lint:file-ignore ST1001 Dot imports by jet
import (
	"encoding/json"
	"time"

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
		expiryDate := a.twitchDataStruct.expiresDate.Local().Format(time.DateTime)

		infoOnConnect, _ := json.Marshal(echo.Map{
			"type":        "TWITCH_INFO",
			"login":       a.twitchDataStruct.login,
			"expiry_date": expiryDate,
			// "is_connected":
		})
		// ws.Write\
		c.Logger().Info(infoOnConnect)

		// Keep connection alive and handle any incoming messages
		for {
			buffer := make([]byte, 1000)
			_, err := ws.Read(buffer)
			if err != nil {
				// This break marks the ws closure
				break
			}
			// We don't handle incoming messages from frontend ever
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}
