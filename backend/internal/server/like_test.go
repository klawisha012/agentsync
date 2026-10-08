package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOneViewAndLike(t *testing.T) {
	e := newServer(t)
	alice := mustAccount(t, e, "alice@example.com", "secret-pass", "Alice")
	bob := mustAccount(t, e, "bob@example.com", "secret-pass", "Bob")

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
		"email": "bob@example.com", "password": "secret-pass",
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

	pushed := postAuth(t, e, "/agent/push", "", pushBody("Grok", false, pushFile{"rules/ok.md", "one"}), alice)
	if pushed.Code != http.StatusCreated {
		t.Fatalf("push %d %s", pushed.Code, pushed.Body.String())
	}
	guest := postJSON(t, e, "/accounts/Alice/agents/Grok/like", map[string]string{}, nil)
	if guest.Code != http.StatusUnauthorized || likesOf(t, getJSON(t, e, "/accounts/Alice", alice)) != 0 {
		t.Fatalf("guest like %d %s", guest.Code, guest.Body.String())
	}
	self := postJSON(t, e, "/accounts/Alice/agents/Grok/like", map[string]string{}, alice)
	if self.Code == http.StatusOK || likesOf(t, getJSON(t, e, "/accounts/Alice", alice)) != 0 {
		t.Fatalf("self like %d %s", self.Code, self.Body.String())
	}
	on := postJSON(t, e, "/accounts/Alice/agents/Grok/like", map[string]string{}, bob)
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
	off := postJSON(t, e, "/accounts/Alice/agents/Grok/like", map[string]string{}, bob)
	if off.Code != http.StatusOK || likesOf(t, getJSON(t, e, "/accounts/Alice", alice)) != 0 {
		t.Fatalf("unlike %d %s", off.Code, off.Body.String())
	}
}

func TestPublicationLikesSum(t *testing.T) {
	e := newServer(t)
	alice := mustAccount(t, e, "alice@example.com", "secret-pass", "Alice")
	bob := mustAccount(t, e, "bob@example.com", "secret-pass", "Bob")
	for _, agent := range []string{"Grok", "Claude"} {
		pushed := postAuth(t, e, "/agent/push", "", pushBody(agent, false, pushFile{"rules/ok.md", "one"}), alice)
		if pushed.Code != http.StatusCreated {
			t.Fatalf("push %s %d %s", agent, pushed.Code, pushed.Body.String())
		}
	}
	hidden := postJSON(t, e, "/accounts/Alice/agents/Agents/like", map[string]string{}, bob)
	if hidden.Code != http.StatusNotFound {
		t.Fatalf("unpublished %d %s", hidden.Code, hidden.Body.String())
	}
	for _, agent := range []string{"Grok", "Claude"} {
		liked := postJSON(t, e, "/accounts/Alice/agents/"+agent+"/like", map[string]string{}, bob)
		if liked.Code != http.StatusOK || !strings.Contains(liked.Body.String(), `"likes":1`) || !strings.Contains(liked.Body.String(), `"total":`) {
			t.Fatalf("like %s %d %s", agent, liked.Code, liked.Body.String())
		}
	}
	page := getJSON(t, e, "/accounts/Alice", bob)
	if likesOf(t, page) != 2 || agentLikesOf(t, page.Body.Bytes(), "Grok") != 1 || agentLikesOf(t, page.Body.Bytes(), "Claude") != 1 {
		t.Fatalf("sum %s", page.Body.String())
	}
	list := getJSON(t, e, "/accounts", nil)
	if !strings.Contains(list.Body.String(), `"name":"Alice"`) || !strings.Contains(list.Body.String(), `"likes":2`) {
		t.Fatalf("list sum %s", list.Body.String())
	}
}

func TestOwnerSelfViewIsDropped(t *testing.T) {
	e := newServer(t)
	mustAccount(t, e, "alice@example.com", "secret-pass", "Alice")
	mustAccount(t, e, "bob@example.com", "secret-pass", "Bob")

	first := getJSON(t, e, "/accounts/Alice", nil)
	if viewsOf(t, first) != 1 {
		t.Fatalf("anonymous self view %s", first.Body.String())
	}
	visitor := readNamedCookie(t, first, "visitor")
	login := postJSON(t, e, "/session", map[string]string{
		"email": "alice@example.com", "password": "secret-pass",
	}, visitor)
	if login.Code != http.StatusOK {
		t.Fatalf("login %d %s", login.Code, login.Body.String())
	}
	owner := append(readSessionCookie(t, login), visitor...)
	opened := getJSON(t, e, "/accounts/Alice", owner)
	if viewsOf(t, opened) != 0 {
		t.Fatalf("own view still counted %s", opened.Body.String())
	}
	fresh := getJSON(t, e, "/accounts/Alice", readSessionCookie(t, login))
	if viewsOf(t, fresh) != 0 {
		t.Fatalf("new browser of the owner counted %s", fresh.Body.String())
	}
	guest := getJSON(t, e, "/accounts/Alice", nil)
	if viewsOf(t, guest) != 1 {
		t.Fatalf("guest view %s", guest.Body.String())
	}
	again := getJSON(t, e, "/accounts/Alice", readNamedCookie(t, guest, "visitor"))
	if viewsOf(t, again) != 1 {
		t.Fatalf("guest repeat %s", again.Body.String())
	}
}

func viewsOf(t *testing.T, rec *httptest.ResponseRecorder) int {
	t.Helper()
	return metric(t, rec.Body.Bytes(), "views")
}

func agentLikesOf(t *testing.T, raw []byte, name string) int {
	t.Helper()
	var body struct {
		Agents []struct {
			Name  string `json:"name"`
			Likes int    `json:"likes"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	for _, agent := range body.Agents {
		if agent.Name == name {
			return agent.Likes
		}
	}
	t.Fatalf("no agent %s in %s", name, raw)
	return 0
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
