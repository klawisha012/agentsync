package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOneViewAndLike(t *testing.T) {
	e := New("http://localhost:3000")
	alice := mustAccount(t, e, "alice@example.com", "secret", "Alice")
	bob := mustAccount(t, e, "bob@example.com", "secret", "Bob")

	first := getJSON(t, e, "/accounts/Alice", nil)
	if viewsOf(t, first) != 1 {
		t.Fatalf("first view %s", first.Body.String())
	}
	visitor := readNamedCookie(t, first, "visitor")
	again := getJSON(t, e, "/accounts/Alice", visitor)
	if viewsOf(t, again) != 1 {
		t.Fatalf("repeat view %s", again.Body.String())
	}

	login := postJSON(t, e, "/session", map[string]string{
		"email": "bob@example.com", "password": "secret",
	}, visitor)
	if login.Code != http.StatusOK {
		t.Fatalf("login %d %s", login.Code, login.Body.String())
	}
	bobAgain := readSessionCookie(t, login)
	bound := getJSON(t, e, "/accounts/Alice", bobAgain)
	if viewsOf(t, bound) != 1 {
		t.Fatalf("login added a view %s", bound.Body.String())
	}
	otherComputer := getJSON(t, e, "/accounts/Alice", bob)
	if viewsOf(t, otherComputer) != 1 {
		t.Fatalf("second computer %s", otherComputer.Body.String())
	}
	owner := getJSON(t, e, "/accounts/Alice", alice)
	if viewsOf(t, owner) != 1 {
		t.Fatalf("owner view %s", owner.Body.String())
	}
	beforePreview := viewsOf(t, getJSON(t, e, "/accounts/Alice", alice))
	preview := getJSON(t, e, "/publications/not-a-page", nil)
	if preview.Code == http.StatusOK {
		t.Fatalf("preview looked like a page %s", preview.Body.String())
	}
	if viewsOf(t, getJSON(t, e, "/accounts/Alice", alice)) != beforePreview {
		t.Fatal("preview increased the page view")
	}

	guest := postJSON(t, e, "/accounts/Alice/like", map[string]string{}, nil)
	if guest.Code != http.StatusUnauthorized || likesOf(t, getJSON(t, e, "/accounts/Alice", alice)) != 0 {
		t.Fatalf("guest like %d %s", guest.Code, guest.Body.String())
	}
	self := postJSON(t, e, "/accounts/Alice/like", map[string]string{}, alice)
	if self.Code == http.StatusOK || likesOf(t, getJSON(t, e, "/accounts/Alice", alice)) != 0 {
		t.Fatalf("self like %d %s", self.Code, self.Body.String())
	}
	on := postJSON(t, e, "/accounts/Alice/like", map[string]string{}, bob)
	if on.Code != http.StatusOK || !strings.Contains(on.Body.String(), `"likes":1`) {
		t.Fatalf("like %d %s", on.Code, on.Body.String())
	}
	if likesOf(t, getJSON(t, e, "/accounts/Alice", bob)) != 1 {
		t.Fatal("page lost the like")
	}
	list := getJSON(t, e, "/accounts", bob)
	if !strings.Contains(list.Body.String(), `"name":"Alice"`) || !strings.Contains(list.Body.String(), `"likes":1`) {
		t.Fatalf("list %s", list.Body.String())
	}
	off := postJSON(t, e, "/accounts/Alice/like", map[string]string{}, bob)
	if off.Code != http.StatusOK || likesOf(t, getJSON(t, e, "/accounts/Alice", alice)) != 0 {
		t.Fatalf("unlike %d %s", off.Code, off.Body.String())
	}
}

func viewsOf(t *testing.T, rec *httptest.ResponseRecorder) int {
	t.Helper()
	return metric(t, rec.Body.Bytes(), "views")
}

func likesOf(t *testing.T, rec *httptest.ResponseRecorder) int {
	t.Helper()
	return metric(t, rec.Body.Bytes(), "likes")
}

func metric(t *testing.T, raw []byte, key string) int {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	value, ok := body[key].(float64)
	if !ok {
		t.Fatalf("no %s in %s", key, raw)
	}
	return int(value)
}

func readNamedCookie(t *testing.T, rec *httptest.ResponseRecorder, name string) []*http.Cookie {
	t.Helper()
	var kept []*http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == name && c.Value != "" {
			kept = append(kept, c)
		}
	}
	if len(kept) != 1 {
		t.Fatalf("%s cookie %#v", name, rec.Result().Cookies())
	}
	return kept
}
