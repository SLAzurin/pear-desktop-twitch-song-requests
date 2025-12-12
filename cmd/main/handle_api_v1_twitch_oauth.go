package main

//lint:file-ignore ST1001 Dot imports by jet
import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/azuridayo/pear-desktop-twitch-song-requests/gen/model"
	. "github.com/azuridayo/pear-desktop-twitch-song-requests/gen/table"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/data"
	"github.com/azuridayo/pear-desktop-twitch-song-requests/internal/databaseconn"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/nicklaw5/helix/v2"
)

func (a *App) processTwitchOAuth(c echo.Context) error {
	// auth data in url hash string params as get request
	body := c.Request().Body
	rawBodyData, err := io.ReadAll(body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "read request body",
		})
	}
	defer body.Close()

	authData := struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		State       string `json:"state,omitempty"`
		TokenType   string `json:"token_type"`
	}{}
	tokenForBot := authData.State == "bot"
	err = json.Unmarshal(rawBodyData, &authData)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "parse request body",
		})
	}

	if authData.TokenType != "bearer" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "unexpected token type",
		})
	}

	isValid, response, err := a.helix.ValidateToken(authData.AccessToken)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusServiceUnavailable, echo.Map{
			"error": "Twitch authentication validation failed, see console.",
		})
	}
	if response.StatusCode == http.StatusOK && isValid {
		expiresIn := response.Data.ExpiresIn
		strDate := response.Header.Get("Date")
		t, err := time.Parse(data.TWITCH_SERVER_DATE_LAYOUT, strDate)
		if err != nil {
			c.Logger().Error(errors.New("Failed to validate server date time expiry, original error:\n" + err.Error()))
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "incorrect expected date data from Twitch",
			})
		}
		t = t.Add(time.Duration(expiresIn) * time.Second)
		db, err := databaseconn.NewDBConnection()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "cannot save token in database",
			})
		}
		defer func() {
			db.Close()
		}()
		selectedHelix := a.helix
		selectedTwitchDataStruct := a.twitchDataStruct
		if tokenForBot {
			if a.twitchDataStruct.login == "" {
				return c.JSON(http.StatusInternalServerError, echo.Map{
					"error": "cannot save bot token before main token",
				})
			}
			if a.helixBot == nil {
				a.helixBot, err = helix.NewClient(&helix.Options{
					ClientID: data.GetTwitchClientID(),
				})
				if err != nil {
					return c.JSON(http.StatusInternalServerError, echo.Map{
						"error": "bot token validation failed preflight",
					})
				}
			}
			selectedHelix = a.helixBot
			selectedTwitchDataStruct = a.twitchDataStructBot
		}
		selectedHelix.SetUserAccessToken(authData.AccessToken)
		selectedTwitchDataStruct.accessToken = authData.AccessToken
		selectedTwitchDataStruct.expiresDate = t
		selectedTwitchDataStruct.isAuthenticated = true
		selectedTwitchDataStruct.userID = response.Data.UserID
		selectedTwitchDataStruct.login = response.Data.Login
		dbSaveKey := data.DB_KEY_TWITCH_ACCESS_TOKEN

		if tokenForBot {
			dbSaveKey = data.DB_KEY_TWITCH_ACCESS_TOKEN_BOT
		} else {
			resp, err := a.helix.GetStreams(&helix.StreamsParams{
				UserLogins: []string{a.twitchDataStruct.login},
			})
			if err == nil && len(resp.Data.Streams) > 0 && resp.Data.Streams[0].ID != "" {
				a.streamOnline = true
			}
		}

		newToken := model.Settings{
			Key:   dbSaveKey,
			Value: authData.AccessToken,
		}
		stmt := Settings.INSERT(Settings.AllColumns).MODEL(newToken).ON_CONFLICT(Settings.Key).DO_UPDATE(SET(
			Settings.Value.SET(String(authData.AccessToken)),
		))

		_, err = stmt.ExecContext(c.Request().Context(), db)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "Failed to save token in database",
			})
		}
		var b echo.Map
		if tokenForBot {
			b = echo.Map{
				"type":            "TWITCH_INFO",
				"login_bot":       a.twitchDataStructBot.login,
				"expiry_date_bot": t.Format(data.TWITCH_SERVER_DATE_LAYOUT),
			}
		} else {
			b = echo.Map{
				"type":          "TWITCH_INFO",
				"login":         a.twitchDataStruct.login,
				"expiry_date":   t.Format(data.TWITCH_SERVER_DATE_LAYOUT),
				"stream_online": a.streamOnline,
				"reward_id":     a.songRequestRewardID,
			}
		}
		bb, _ := json.Marshal(b)
		a.clientsBroadcast <- string(bb)
		return c.NoContent(http.StatusOK)
	} else {
		return c.NoContent(http.StatusUnauthorized)
	}
}
