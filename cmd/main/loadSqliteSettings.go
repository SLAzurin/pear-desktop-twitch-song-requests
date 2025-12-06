package main

//lint:file-ignore ST1001 Dot imports by jet
import (
	"errors"
	"net/http"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/gen/model"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/databaseconn"
	"github.com/nicklaw5/helix/v2"

	. "github.com/azuridayo/pear-desktop-twitch-song-requests/gen/table"
	. "github.com/go-jet/jet/v2/sqlite"
)

func (a *App) loadSqliteSettings() error {
	db, err := databaseconn.NewDBConnection()
	if err != nil {
		return err
	}

	twitchDataStruct := twitchData{}
	results := []model.Settings{}
	stmt := SELECT(Settings.Value).FROM(Settings).WHERE(Settings.Key.EQ(String(data.DB_KEY_TWITCH_ACCESS_TOKEN))).LIMIT(1)
	err = stmt.QueryContext(a.ctx, db, &results)
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.Key == data.DB_KEY_TWITCH_ACCESS_TOKEN {
			twitchDataStruct.accessToken = result.Value
		}
	}

	c, err := helix.NewClient(&helix.Options{
		ClientID: data.GetTwitchClientID(),
	})
	if err != nil {
		return err
	}

	if twitchDataStruct.accessToken != "" {
		isValid, response, err := c.ValidateToken(twitchDataStruct.accessToken)
		if err != nil {
			// req error
			return err
		}
		if response.StatusCode == http.StatusOK && isValid {
			c.SetUserAccessToken(twitchDataStruct.accessToken)
			twitchDataStruct.isAuthenticated = true

			twitchDataStruct.userID = response.Data.UserID
			twitchDataStruct.login = response.Data.Login
			expiresIn := response.Data.ExpiresIn
			strDate := response.Header.Get("Date")
			t, err := time.Parse(data.TWITCH_SERVER_DATE_LAYOUT, strDate)
			if err != nil {
				return errors.New("Failed to validate server date time expiry, original error:\n" + err.Error())
			}
			t = t.Add(time.Duration(expiresIn) * time.Second)
			twitchDataStruct.expiresDate = t
		}
	}
	a.helix = c
	a.twitchDataStruct = twitchDataStruct

	return nil
}
