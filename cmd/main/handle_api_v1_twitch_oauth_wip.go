package main

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

func (a *App) handleTwitchOAuth(c echo.Context) error {
	// auth data in url hash string params as get request
	body := c.Request().Body
	rawBodyData, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	defer body.Close()

	authData := struct {
		Something string
	}{}
	err = json.Unmarshal(rawBodyData, &authData)
	return echo.NewHTTPError(http.StatusTeapot, "under construction", err)
}
