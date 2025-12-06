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
		a.helix.SetUserAccessToken(authData.AccessToken)
		a.twitchDataStruct.expiresDate = t
		a.twitchDataStruct.isAuthenticated = true
		a.twitchDataStruct.userID = response.Data.UserID
		a.twitchDataStruct.login = response.Data.Login

		db, err := databaseconn.NewDBConnection()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "cannot save token in database",
			})
		}
		defer func() {
			db.Close()
		}()
		newToken := model.Settings{
			Key:   data.DB_KEY_TWITCH_ACCESS_TOKEN,
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

		return c.NoContent(http.StatusOK)
	} else {
		return c.NoContent(http.StatusUnauthorized)
	}
}
