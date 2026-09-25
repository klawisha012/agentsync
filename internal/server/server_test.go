package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	e := New("http://localhost:3000")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body %v", body)
	}
}

func TestRegisterThenSessionShowsName(t *testing.T) {
	e := New("http://localhost:3000")
	rec := postJSON(t, e, "/accounts", map[string]string{
		"email":    "a@example.com",
		"password": "secret",
		"name":     "Alice",
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d body %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "a@example.com") || strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("response leaked a secret: %s", rec.Body.String())
	}
	cookie := readSessionCookie(t, rec)
	view := getJSON(t, e, "/session", cookie)
	if view.Code != http.StatusOK {
		t.Fatalf("session status %d body %s", view.Code, view.Body.String())
	}
	got := decodeAccount(t, view.Body)
	if got.Name != "Alice" || got.Views != 0 || got.Likes != 0 {
		t.Fatalf("session %#v", got)
	}
	public := getJSON(t, e, "/accounts/Alice", nil)
	if public.Code != http.StatusOK {
		t.Fatalf("public status %d", public.Code)
	}
	page := decodeAccount(t, public.Body)
	if page.Name != "Alice" || page.Views != 0 || page.Likes != 0 {
		t.Fatalf("page %#v", page)
	}
	if strings.Contains(public.Body.String(), "email") || strings.Contains(public.Body.String(), "a@example.com") {
		t.Fatalf("public page leaked email: %s", public.Body.String())
	}
	again := postJSON(t, e, "/accounts", map[string]string{
		"email": "a@example.com", "password": "secret", "name": "harbor",
	}, nil)
	if again.Code != http.StatusConflict {
		t.Fatalf("duplicate email %d %s", again.Code, again.Body.String())
	}

	out := postJSON(t, e, "/session", map[string]string{
		"email":    "a@example.com",
		"password": "secret",
	}, nil)
	if out.Code != http.StatusOK {
		t.Fatalf("login status %d body %s", out.Code, out.Body.String())
	}
	end := deleteCookie(t, e, "/session", readSessionCookie(t, out))
	if end.Code != http.StatusNoContent {
		t.Fatalf("logout status %d", end.Code)
	}
	gone := getJSON(t, e, "/session", readSessionCookie(t, out))
	if gone.Code != http.StatusUnauthorized {
		t.Fatalf("session after logout %d", gone.Code)
	}
}

func TestRegisterRejectsName(t *testing.T) {
	tests := []struct {
		name     string
		account  string
		wantCode int
	}{
		{name: "too short", account: "Al", wantCode: http.StatusBadRequest},
		{name: "hyphen on the edge", account: "-alice", wantCode: http.StatusBadRequest},
		{name: "space inside", account: "al ice", wantCode: http.StatusBadRequest},
		{name: "site page", account: "login", wantCode: http.StatusConflict},
		{name: "same letters different case", account: "alice", wantCode: http.StatusConflict},
		{name: "other alphabet stays distinct", account: "Аlice", wantCode: http.StatusCreated},
		{name: "not a site page", account: "session", wantCode: http.StatusCreated},
	}
	e := New("http://localhost:3000")
	first := postJSON(t, e, "/accounts", map[string]string{
		"email": "first@example.com", "password": "secret", "name": "Alice",
	}, nil)
	if first.Code != http.StatusCreated {
		t.Fatalf("seed %d %s", first.Code, first.Body.String())
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postJSON(t, e, "/accounts", map[string]string{
				"email": tt.account + "@example.com", "password": "secret", "name": tt.account,
			}, nil)
			if rec.Code != tt.wantCode {
				t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
			}
			if tt.wantCode != http.StatusCreated && !strings.Contains(rec.Body.String(), "explanation") {
				t.Fatalf("missing explanation: %s", rec.Body.String())
			}
		})
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	e := New("http://localhost:3000")
	created := postJSON(t, e, "/accounts", map[string]string{
		"email": "a@example.com", "password": "secret", "name": "harbor",
	}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("seed %d %s", created.Code, created.Body.String())
	}
	tests := []struct {
		name string
		body map[string]string
	}{
		{name: "empty fields", body: map[string]string{"email": "", "password": ""}},
		{name: "wrong password", body: map[string]string{"email": "a@example.com", "password": "nope"}},
		{name: "unknown email", body: map[string]string{"email": "missing@example.com", "password": "secret"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postJSON(t, e, "/session", tt.body, nil)
			if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
			}
			if tt.name == "empty fields" && rec.Code != http.StatusBadRequest {
				t.Fatalf("empty status %d", rec.Code)
			}
			if tt.name != "empty fields" && rec.Code != http.StatusUnauthorized {
				t.Fatalf("bad login status %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "explanation") {
				t.Fatalf("missing explanation: %s", rec.Body.String())
			}
		})
	}
}

type seenAccount struct {
	Name  string `json:"name"`
	Views int    `json:"views"`
	Likes int    `json:"likes"`
}

func postJSON(t *testing.T, h http.Handler, path string, body any, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func getJSON(t *testing.T, h http.Handler, path string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func deleteCookie(t *testing.T, h http.Handler, path string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func readSessionCookie(t *testing.T, rec *httptest.ResponseRecorder) []*http.Cookie {
	t.Helper()
	var kept []*http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" && c.Value != "" && c.MaxAge >= 0 {
			kept = append(kept, c)
		}
	}
	if len(kept) != 1 {
		t.Fatalf("session cookie %#v", rec.Result().Cookies())
	}
	if !kept[0].HttpOnly {
		t.Fatal("session cookie is readable from script")
	}
	return kept
}

func decodeAccount(t *testing.T, body *bytes.Buffer) seenAccount {
	t.Helper()
	raw, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	var view seenAccount
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatal(err)
	}
	return view
}
