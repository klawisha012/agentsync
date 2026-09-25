package server

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var pgDSN string

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("agentsync"),
		postgres.WithUsername("agentsync"),
		postgres.WithPassword("agentsync"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		panic(err)
	}
	pgDSN, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}
	code := m.Run()
	_ = container.Terminate(context.Background())
	os.Exit(code)
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	name, err := newID()
	if err != nil {
		t.Fatal(err)
	}
	name = "t" + name[:12]
	admin, err := sql.Open("pgx", pgDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close() })
	if _, err := admin.ExecContext(context.Background(), "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		drop, err := sql.Open("pgx", pgDSN)
		if err != nil {
			return
		}
		defer drop.Close()
		_, _ = drop.ExecContext(context.Background(), "DROP DATABASE "+name+" WITH (FORCE)")
	})
	dsn, err := withDatabase(pgDSN, name)
	if err != nil {
		t.Fatal(err)
	}
	db, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func withDatabase(dsn, name string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	parsed.Path = "/" + name
	return parsed.String(), nil
}

func mustServer(t *testing.T, db *sql.DB) *echo.Echo {
	t.Helper()
	e, err := New("http://localhost:3000", db)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func newServer(t *testing.T) *echo.Echo {
	t.Helper()
	return mustServer(t, testDB(t))
}
