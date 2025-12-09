package main

import (
	"log"

	"github.com/valyala/fastjson"
)

func (a *App) handlePearDesktopMsgs() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case msg := <-a.pearDesktopIncomingMsgs:
			msgType := fastjson.GetString(msg, "type")
			if msgType == "POSITION_CHANGED" {
				continue
			}
			log.Printf("%s\n", msgType)
		}
	}
}
