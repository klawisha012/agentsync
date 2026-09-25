package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type app struct {
	accounts *store
}

func New(webOrigin string) *echo.Echo {
	a := &app{accounts: newStore()}
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	if webOrigin != "" {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{webOrigin},
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions},
			AllowHeaders:     []string{echo.HeaderContentType},
			AllowCredentials: true,
		}))
	}
	e.GET("/health", health)
	e.POST("/accounts", a.createAccount)
	e.GET("/accounts/:name", a.publicAccount)
	e.POST("/session", a.createSession)
	e.GET("/session", a.currentSession)
	e.DELETE("/session", a.deleteSession)
	return e
}

func health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
