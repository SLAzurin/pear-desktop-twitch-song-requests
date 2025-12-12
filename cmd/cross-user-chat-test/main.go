package main

//lint:file-ignore ST1001 Dot imports by jet
import (
	"log"

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
	secret2 := ""
	for _, v := range ss {
		if v.Key == data.DB_KEY_TWITCH_ACCESS_TOKEN {
			secret = v.Value
		}
		if v.Key == data.DB_KEY_TWITCH_ACCESS_TOKEN_BOT {
			secret2 = v.Value
		}
	}
	if secret == "" {
		panic("no auth1")
	}
	if secret2 == "" {
		panic("no auth2")
	}

	c, _ := helix.NewClient(&helix.Options{
		// ClientID: data.GetTwitchClientID(),
		ClientID: "771pv6m10b4nwaytvk48b7m192y2iu",
	})
	c2, _ := helix.NewClient(&helix.Options{
		// ClientID: data.GetTwitchClientID(),
		ClientID: "771pv6m10b4nwaytvk48b7m192y2iu",
	})

	b, _, _ := c.ValidateToken(secret)
	if !b {
		panic("api: unauthorized 1")
	}
	c.SetUserAccessToken(secret)

	b2, _, _ := c2.ValidateToken(secret2)
	if !b2 {
		panic("api: unauthorized 2")
	}
	c2.SetUserAccessToken(secret2)

	users, err := c.GetUsers(&helix.UsersParams{})
	if len(users.Data.Users) < 1 {
		log.Println(users)
		panic(err)
	}

	id1 := users.Data.Users[0].ID

	users2, err := c2.GetUsers(&helix.UsersParams{})
	if len(users2.Data.Users) < 1 {
		log.Println(users2)
		panic(err)
	}

	id2 := users2.Data.Users[0].ID

	c2.SendChatMessage(&helix.SendChatMessageParams{
		BroadcasterID: id1,
		SenderID:      id2,
		Message:       "testering",
	})
}
