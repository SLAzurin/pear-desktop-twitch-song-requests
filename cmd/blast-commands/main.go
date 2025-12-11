package main

//lint:file-ignore ST1001 Dot imports by jet
import (
	"log"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/databaseconn"
	"github.com/nicklaw5/helix/v2"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/gen/model"
	. "github.com/azuridayo/pear-desktop-twitch-song-requests/gen/table"
	. "github.com/go-jet/jet/v2/sqlite"
)

func main() {
	db, err := databaseconn.NewDBConnection()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt := SELECT(Settings.Key, Settings.Value).FROM(Settings)

	ss := []model.Settings{}

	err = stmt.Query(db, &ss)
	if err != nil {
		panic(err)
	}

	secret := ""
	for _, v := range ss {
		if v.Key == data.DB_KEY_TWITCH_ACCESS_TOKEN {
			secret = v.Value
			break
		}
	}
	if secret == "" {
		panic("no auth")
	}

	c, _ := helix.NewClient(&helix.Options{
		// ClientID: data.GetTwitchClientID(),
		ClientID: "771pv6m10b4nwaytvk48b7m192y2iu",
	})

	b, _, _ := c.ValidateToken(secret)
	if !b {
		panic("api: unauthorized")
	}

	c.SetUserAccessToken(secret)

	users, err := c.GetUsers(&helix.UsersParams{})
	if len(users.Data.Users) < 1 {
		log.Println(users)
		panic(err)
	}

	id := users.Data.Users[0].ID

	sr := []string{
		"!sr yena nemonemo",
		"!sr yena good morning",
		"!sr yena being a good girl hurts",
		"!sr yena smartphone",
		"!skip",
	}

	// App must survive this blast and start playing nemonemo
	for i, m := range sr {
		if i > 0 {
			time.Sleep(time.Millisecond * 200)
		}
		c.SendChatMessage(&helix.SendChatMessageParams{
			BroadcasterID: id,
			SenderID:      id,
			Message:       m,
		})
	}
}
