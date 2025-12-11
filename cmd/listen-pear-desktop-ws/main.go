package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/songrequests"
	"github.com/recws-org/recws"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ws := recws.RecConn{
		RecIntvlFactor: 1,
		RecIntvlMin:    3 * time.Second,
	}
	ws.Dial("ws://"+songrequests.GetPearDesktopHost()+"/api/v1/ws", nil)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			select {
			case <-ctx.Done():
				go ws.Close()
				log.Printf("Websocket closed %s", ws.GetURL())
				return
			default:
				if !ws.IsConnected() {
					time.Sleep(3 * time.Second)
					continue
				}

				_, message, err := ws.ReadMessage()
				if err != nil {
					time.Sleep(3 * time.Second)
					continue
				}

				log.Printf("%s\n", message)
			}
		}
	}()
	<-sigs
	cancel()
	time.Sleep(5 * time.Second)
}
