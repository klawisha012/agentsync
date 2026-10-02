package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginStoresSessionInStateDir(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"zwarder"}`))
	}))
	defer srv.Close()

	t.Setenv("AGENTSYNC_STATE", t.TempDir())
	name, err := Login(context.Background(), srv.URL, "a@b.c", "secret")
	if err != nil || name != "zwarder" {
		t.Fatalf("login %v %s", err, name)
	}
	if HomeFile("session") != "abc" || SessionCookie() != "session=abc" {
		t.Fatalf("session %q cookie %q", HomeFile("session"), SessionCookie())
	}
}
