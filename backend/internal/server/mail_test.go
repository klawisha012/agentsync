package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestEmailConfirmBlocksUploadAndResetPassword(t *testing.T) {
	e := newServer(t)
	const (
		email    = "alex@studio.io"
		password = "secret-pass"
		name     = "alex"
	)
	created := postJSON(t, e, "/accounts", map[string]string{
		"email": email, "password": password, "name": name,
	}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("create %d %s", created.Code, created.Body.String())
	}
	if strings.Contains(created.Body.String(), email) || strings.Contains(created.Body.String(), password) {
		t.Fatalf("create leaked a secret: %s", created.Body.String())
	}
	cookie := readSessionCookie(t, created)
	confirmToken := testLetter(t, e, email, "confirm")
	if strings.Contains(created.Body.String(), "letterPath") {
		t.Fatalf("create returned a letter: %s", created.Body.String())
	}

	blocked := postJSON(t, e, "/publications", map[string]string{}, cookie)
	if blocked.Code != http.StatusNotImplemented || strings.Contains(blocked.Body.String(), "почт") {
		t.Fatalf("upload before the letter %d %s", blocked.Code, blocked.Body.String())
	}
	removed := deleteCookie(t, e, "/publications/any", cookie)
	if removed.Code != http.StatusNotImplemented || strings.Contains(removed.Body.String(), "почт") {
		t.Fatalf("withdraw before the letter %d %s", removed.Code, removed.Body.String())
	}

	bad := postJSON(t, e, "/email/confirm", map[string]string{"token": "missing"}, cookie)
	if bad.Code == http.StatusOK || !strings.Contains(bad.Body.String(), "explanation") {
		t.Fatalf("bad confirm %d %s", bad.Code, bad.Body.String())
	}
	if ownerVerified(t, getJSON(t, e, "/accounts/"+name, cookie)) {
		t.Fatal("bad link confirmed the mail")
	}

	ok := postJSON(t, e, "/email/confirm", map[string]string{"token": confirmToken}, nil)
	if ok.Code != http.StatusOK {
		t.Fatalf("confirm %d %s", ok.Code, ok.Body.String())
	}
	if !ownerVerified(t, getJSON(t, e, "/accounts/"+name, cookie)) {
		t.Fatal("letter did not confirm the mail")
	}
	again := postJSON(t, e, "/email/confirm", map[string]string{"token": confirmToken}, nil)
	if again.Code != http.StatusOK || !ownerVerified(t, getJSON(t, e, "/accounts/"+name, cookie)) {
		t.Fatalf("repeat confirm %d %s", again.Code, again.Body.String())
	}

	open := postJSON(t, e, "/publications", map[string]string{}, cookie)
	if open.Code != http.StatusNotImplemented || strings.Contains(open.Body.String(), "почт") {
		t.Fatalf("confirmed mail still blocks upload: %d %s", open.Code, open.Body.String())
	}
	if strings.Contains(open.Body.String(), "переносимой") {
		t.Fatalf("upload pretended to inspect a portable setup: %s", open.Body.String())
	}

	guest := getJSON(t, e, "/accounts/"+name, nil)
	if strings.Contains(guest.Body.String(), "alex@") || strings.Contains(guest.Body.String(), "maskedMail") {
		t.Fatalf("guest saw the mail: %s", guest.Body.String())
	}
	owner := getJSON(t, e, "/accounts/"+name, cookie)
	if !strings.Contains(owner.Body.String(), "alex@***.io") || strings.Contains(owner.Body.String(), email) {
		t.Fatalf("owner mail %s", owner.Body.String())
	}

	reset := postJSON(t, e, "/recovery", map[string]string{"email": email}, nil)
	if reset.Code != http.StatusOK {
		t.Fatalf("recovery %d %s", reset.Code, reset.Body.String())
	}
	if strings.Contains(reset.Body.String(), "letterPath") {
		t.Fatalf("recovery returned a letter: %s", reset.Body.String())
	}
	resetToken := testLetter(t, e, email, "reset")
	wrong := postJSON(t, e, "/recovery/password", map[string]string{"token": "missing", "password": "newer"}, nil)
	if wrong.Code == http.StatusOK {
		t.Fatalf("bad reset changed the password: %s", wrong.Body.String())
	}
	still := postJSON(t, e, "/session", map[string]string{"email": email, "password": password}, nil)
	if still.Code != http.StatusOK {
		t.Fatalf("old password rejected after a bad link: %d %s", still.Code, still.Body.String())
	}

	changed := postJSON(t, e, "/recovery/password", map[string]string{"token": resetToken, "password": "newer-pass"}, nil)
	if changed.Code != http.StatusOK {
		t.Fatalf("reset %d %s", changed.Code, changed.Body.String())
	}
	old := postJSON(t, e, "/session", map[string]string{"email": email, "password": password}, nil)
	if old.Code == http.StatusOK {
		t.Fatal("old password still opens the account")
	}
	next := postJSON(t, e, "/session", map[string]string{"email": email, "password": "newer-pass"}, nil)
	if next.Code != http.StatusOK {
		t.Fatalf("new password %d %s", next.Code, next.Body.String())
	}
	seen := decodeAccount(t, next.Body)
	if seen.Name != name {
		t.Fatalf("reset replaced the account: %#v", seen)
	}
}

func testLetter(t *testing.T, e *echo.Echo, email, kind string) string {
	t.Helper()
	value, ok := testStores.Load(e)
	if !ok {
		t.Fatal("хранилище писем не зарегистрировано")
	}
	s := value.(*store)
	s.mu.Lock()
	raw := s.letters[foldKey.String(email)+"\n"+kind]
	s.mu.Unlock()
	if raw == "" || strings.Contains(raw, "@") {
		t.Fatalf("letter %s for %s is %q", kind, email, raw)
	}
	return raw
}

func ownerVerified(t *testing.T, rec *httptest.ResponseRecorder) bool {
	t.Helper()
	var body struct {
		Verified bool `json:"verified"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Verified
}
