package main

//lint:file-ignore ST1001 Dot imports by jet
import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/gen/model"
	. "github.com/azuridayo/pear-desktop-twitch-song-requests/gen/table"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/databaseconn"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/labstack/echo/v4"
)

func (a *App) processTwitchSettings(c echo.Context) error {
	// auth data in url hash string params as get request
	body := c.Request().Body
	rawBodyData, err := io.ReadAll(body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "read request body",
		})
	}
	defer body.Close()

	settings := map[string]string{}
	err = json.Unmarshal(rawBodyData, &settings)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "parse request body",
		})
	}
	db, err := databaseconn.NewDBConnection()
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "save data failed",
		})
	}
	defer db.Close()
	for k, v := range settings {
		if k == data.DB_KEY_TWITCH_SONG_REQUEST_REWARD_ID {
			newSetting := model.Settings{
				Key:   data.DB_KEY_TWITCH_SONG_REQUEST_REWARD_ID,
				Value: v,
			}
			stmt := Settings.INSERT(Settings.AllColumns).MODEL(newSetting).ON_CONFLICT(Settings.Key).DO_UPDATE(SET(
				Settings.Value.SET(String(v)),
			))
			stmt.ExecContext(c.Request().Context(), db)
			a.songRequestRewardID = v
		}
	}

	b := echo.Map{
		"type":            "TWITCH_INFO",
		"login":           a.twitchDataStruct.login,
		"expiry_date":     a.twitchDataStruct.expiresDate.Local().Format(data.TWITCH_SERVER_DATE_LAYOUT),
		"stream_online":   a.streamOnline,
		"reward_id":       a.songRequestRewardID,
		"login_bot":       a.twitchDataStructBot.login,
		"expiry_date_bot": a.twitchDataStructBot.expiresDate.Local().Format(data.TWITCH_SERVER_DATE_LAYOUT),
	}
	bb, _ := json.Marshal(b)
	a.clientsBroadcast <- string(bb)
	return c.NoContent(http.StatusOK)

}
