package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAccountSessionAndLetterSurviveRestart(t *testing.T) {
	db := testDB(t)
	first := mustServer(t, db)
	created := postJSON(t, first, "/accounts", map[string]string{
		"email": "ada@example.com", "password": "secret", "name": "Ada",
	}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("create %d %s", created.Code, created.Body.String())
	}
	cookie := readSessionCookie(t, created)
	confirmToken := pathToken(letterPath(t, created.Body.Bytes()))

	second := mustServer(t, db)
	view := getJSON(t, second, "/session", cookie)
	if view.Code != http.StatusOK || decodeAccount(t, view.Body).Name != "Ada" {
		t.Fatalf("session after restart %d %s", view.Code, view.Body.String())
	}
	confirmed := postJSON(t, second, "/email/confirm", map[string]string{"token": confirmToken}, nil)
	if confirmed.Code != http.StatusOK {
		t.Fatalf("letter after restart %d %s", confirmed.Code, confirmed.Body.String())
	}
	reset := postJSON(t, second, "/recovery", map[string]string{"email": "ada@example.com"}, nil)
	if reset.Code != http.StatusOK {
		t.Fatalf("recovery %d %s", reset.Code, reset.Body.String())
	}
	resetToken := pathToken(letterPath(t, reset.Body.Bytes()))

	third := mustServer(t, db)
	changed := postJSON(t, third, "/recovery/password", map[string]string{
		"token": resetToken, "password": "newer-pass",
	}, nil)
	if changed.Code != http.StatusOK {
		t.Fatalf("reset after restart %d %s", changed.Code, changed.Body.String())
	}
	old := postJSON(t, third, "/session", map[string]string{
		"email": "ada@example.com", "password": "secret",
	}, nil)
	if old.Code == http.StatusOK {
		t.Fatal("old password still opens the account after restart")
	}
	next := postJSON(t, third, "/session", map[string]string{
		"email": "ada@example.com", "password": "newer-pass",
	}, nil)
	if next.Code != http.StatusOK || decodeAccount(t, next.Body).Name != "Ada" {
		t.Fatalf("new password after restart %d %s", next.Code, next.Body.String())
	}
}

func TestPageAndPublicationSurviveRestart(t *testing.T) {
	db := testDB(t)
	first := mustServer(t, db)
	owner := postJSON(t, first, "/accounts", map[string]string{
		"email": "owner@example.com", "password": "secret", "name": "Owner",
	}, nil)
	ownerCookie := readSessionCookie(t, owner)
	confirm := postJSON(t, first, "/email/confirm", map[string]string{
		"token": pathToken(letterPath(t, owner.Body.Bytes())),
	}, nil)
	if confirm.Code != http.StatusOK {
		t.Fatalf("confirm %d %s", confirm.Code, confirm.Body.String())
	}
	machine := postJSON(t, first, "/machine", map[string]string{
		"id": "pc-owner", "host": "desk", "listener": "127.0.0.1:49152",
	}, ownerCookie)
	token := decodeMachine(t, machine.Body.Bytes()).AgentToken
	pushed := postAuth(t, first, "/agent/push", token, pushBody("Grok", false, pushFile{"rules/ok.md", "hello"}), ownerCookie)
	if pushed.Code != http.StatusCreated {
		t.Fatalf("push %d %s", pushed.Code, pushed.Body.String())
	}
	pubID := publicationID(t, pushed.Body.Bytes())

	guest := getJSON(t, first, "/accounts/Owner", nil)
	visitor := readNamedCookie(t, guest, "visitor")
	if viewsOf(t, guest) != 1 {
		t.Fatalf("view %s", guest.Body.String())
	}
	liker := postJSON(t, first, "/accounts", map[string]string{
		"email": "liker@example.com", "password": "secret", "name": "Liker",
	}, nil)
	likerCookie := readSessionCookie(t, liker)
	liked := postJSON(t, first, "/accounts/Owner/like", map[string]string{}, likerCookie)
	if liked.Code != http.StatusOK {
		t.Fatalf("like %d %s", liked.Code, liked.Body.String())
	}

	second := mustServer(t, db)
	again := getJSON(t, second, "/accounts/Owner", visitor)
	if viewsOf(t, again) != 1 || likesOf(t, again) != 1 {
		t.Fatalf("page after restart %s", again.Body.String())
	}
	kept := getJSON(t, second, "/publications/"+pubID, nil)
	if kept.Code != http.StatusOK || !strings.Contains(kept.Body.String(), "rules/ok.md") {
		t.Fatalf("publication after restart %d %s", kept.Code, kept.Body.String())
	}
	agent := getAuth(t, second, "/agent/session", token)
	if agent.Code != http.StatusOK || decodeMachine(t, agent.Body.Bytes()).Account != "Owner" {
		t.Fatalf("machine after restart %d %s", agent.Code, agent.Body.String())
	}
}
