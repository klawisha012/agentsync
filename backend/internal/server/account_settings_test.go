package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestChangeMailAndPassword(t *testing.T) {
	e := newServer(t)
	created := postJSON(t, e, "/accounts", map[string]string{
		"email": "old@example.com", "password": "secret", "name": "Setter",
	}, nil)
	cookie := readSessionCookie(t, created)

	changed := postJSON(t, e, "/account/email", map[string]string{
		"email": "next@example.com", "password": "secret",
	}, cookie)
	if changed.Code != http.StatusOK || !strings.Contains(changed.Body.String(), "letterPath") || !strings.Contains(changed.Body.String(), `"verified":false`) {
		t.Fatalf("email %d %s", changed.Code, changed.Body.String())
	}
	page := getJSON(t, e, "/accounts/Setter", cookie)
	if !strings.Contains(page.Body.String(), "next@***.com") {
		t.Fatalf("masked %s", page.Body.String())
	}
	if strings.Contains(page.Body.String(), `"verified":true`) {
		t.Fatal("email change left the account verified")
	}

	bad := postJSON(t, e, "/account/password", map[string]string{
		"current": "nope", "next": "newer-secret",
	}, cookie)
	if bad.Code == http.StatusOK {
		t.Fatal("wrong password changed the account")
	}
	ok := postJSON(t, e, "/account/password", map[string]string{
		"current": "secret", "next": "newer-secret",
	}, cookie)
	if ok.Code != http.StatusOK {
		t.Fatalf("password %d %s", ok.Code, ok.Body.String())
	}
	login := postJSON(t, e, "/session", map[string]string{
		"email": "next@example.com", "password": "newer-secret",
	}, nil)
	if login.Code != http.StatusOK {
		t.Fatalf("login %d %s", login.Code, login.Body.String())
	}
}
