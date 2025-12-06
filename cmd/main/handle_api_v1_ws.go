package main

import (
	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

func (a *App) handleAppWs(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		// Add client to the map
		a.clientsMu.Lock()
		a.clients[ws] = true
		a.clientsMu.Unlock()

		defer func() {
			a.clientsMu.Lock()
			delete(a.clients, ws)
			a.clientsMu.Unlock()
		}()

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
