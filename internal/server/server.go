package server

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type app struct {
	accounts *store
}

func New(webOrigin string, db *sql.DB) (*echo.Echo, error) {
	accounts, err := newStore(context.Background(), db)
	if err != nil {
		return nil, err
	}
	a := &app{accounts: accounts}
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	if webOrigin != "" {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{webOrigin},
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions},
			AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAuthorization},
			AllowCredentials: true,
		}))
	}
	e.GET("/health", health)
	e.POST("/accounts", a.createAccount)
	e.GET("/accounts", a.listAccounts)
	e.GET("/accounts/:name", a.publicAccount)
	e.POST("/accounts/:name/like", a.toggleLike)
	e.POST("/email/confirm", a.confirmEmail)
	e.POST("/recovery", a.requestRecovery)
	e.POST("/recovery/password", a.resetPassword)
	e.POST("/publications", a.publicationGate)
	e.GET("/publications/:id", a.showPublication)
	e.POST("/publications/:id/withdraw", a.withdrawPublication)
	e.GET("/api/apply/:name/:agent", a.applyPublication)
	e.POST("/account/name", a.renameAccount)
	e.POST("/account/delete", a.deleteAccount)
	e.GET("/cli/install.ps1", installPS)
	e.GET("/cli/install.sh", installSH)
	e.DELETE("/publications/:id", a.publicationGate)
	e.POST("/agent/push", a.pushAgent)
	e.POST("/machine", a.confirmMachine)
	e.DELETE("/machine/:id", a.releaseMachine)
	e.GET("/agent/session", a.agentSession)
	e.POST("/session", a.createSession)
	e.GET("/session", a.currentSession)
	e.DELETE("/session", a.deleteSession)
	return e, nil
}

func health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
