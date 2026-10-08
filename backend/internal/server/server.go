package server

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type app struct {
	accounts  *store
	webOrigin string
	authHits  hitWindow
}

func New(webOrigin string, db *sql.DB) (*echo.Echo, error) {
	accounts, err := newStore(context.Background(), db)
	if err != nil {
		return nil, err
	}
	scripts, err := readInstallScripts()
	if err != nil {
		return nil, err
	}
	a := &app{accounts: accounts, webOrigin: webOrigin}
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("1G"))
	e.Use(a.guardOrigin)
	if webOrigin != "" {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{webOrigin},
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions},
			AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAuthorization},
			AllowCredentials: true,
		}))
	}
	tight := middleware.BodyLimit("32K")
	e.GET("/health", health)
	e.GET("/receivers", listReceivers)
	e.POST("/accounts", a.createAccount, a.limitAuth, tight)
	e.GET("/accounts", a.listAccounts)
	e.GET("/stats", a.showStats)
	e.GET("/accounts/:name", a.publicAccount)
	e.GET("/accounts/:name/versions/:agent", a.listVersions)
	e.GET("/accounts/:name/versions/:agent/:version", a.showVersion)
	e.POST("/accounts/:name/agents/:agent/like", a.toggleLike)
	e.GET("/accounts/:name/avatar", a.showAvatar)
	e.GET("/accounts/:name/comments", a.listComments)
	e.POST("/accounts/:name/comments", a.postComment, tight)
	e.DELETE("/accounts/:name/comments/:id", a.removeComment)
	e.POST("/email/confirm", a.confirmEmail, a.limitAuth, tight)
	e.POST("/recovery", a.requestRecovery, a.limitAuth, tight)
	e.POST("/recovery/password", a.resetPassword, a.limitAuth, tight)
	e.POST("/publications", a.publicationGate)
	e.GET("/publications/:id", a.showPublication)
	e.POST("/publications/:id/withdraw", a.withdrawPublication)
	e.GET("/apply/:name/:agent", a.applyPublication)
	e.GET("/api/apply/:name/:agent", a.applyPublication)
	e.POST("/account/name", a.renameAccount)
	e.POST("/account/privacy", a.setPrivacy)
	e.POST("/account/email", a.changeEmail, a.limitAuth, tight)
	e.POST("/account/password", a.changePassword, a.limitAuth, tight)
	e.POST("/account/avatar", a.putAvatar)
	e.DELETE("/account/avatar", a.deleteAvatar)
	e.POST("/account/delete", a.deleteAccount, a.limitAuth, tight)
	bins := openCLIBinaries()
	e.GET("/cli/install.ps1", scripts.ps)
	e.GET("/cli/install.sh", scripts.sh)
	e.GET("/cli/version", bins.cliVersion)
	e.GET("/cli/:file", bins.get)
	e.DELETE("/publications/:id", a.publicationGate)
	e.POST("/agent/push", a.pushAgent)
	e.POST("/agent/record", a.recordSnapshot)
	e.GET("/agent/store/:agent", a.listSnapshots)
	e.GET("/agent/store/:agent/:number", a.showSnapshot)
	e.POST("/machine", a.confirmMachine)
	e.DELETE("/machine/:id", a.releaseMachine)
	e.GET("/agent/session", a.agentSession)
	e.POST("/session", a.createSession, a.limitAuth, tight)
	e.GET("/session", a.currentSession)
	e.DELETE("/session", a.deleteSession)
	if testing.Testing() {
		testStores.Store(e, accounts)
	}
	return e, nil
}

func health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
