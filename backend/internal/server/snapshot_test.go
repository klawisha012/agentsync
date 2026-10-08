package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestRecordKeepsSnapshotAfterRestart(t *testing.T) {
	db := testDB(t)
	e := mustServer(t, db)
	created := postJSON(t, e, "/accounts", map[string]string{
		"email": "keeper@example.com", "password": "secret-pass", "name": "Keeper",
	}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("create %d %s", created.Code, created.Body.String())
	}
	owner := readSessionCookie(t, created)
	machine := postJSON(t, e, "/machine", map[string]string{
		"id": testMachineID("pc-keeper"), "host": "desk", "listener": "127.0.0.1:49152",
	}, owner)
	token := decodeMachine(t, machine.Body.Bytes()).AgentToken

	early := postAuth(t, e, "/agent/record", token, pushBody("Grok", false, pushFile{"rules/ok.md", "hello"}), owner)
	if early.Code != http.StatusCreated || !strings.Contains(early.Body.String(), `"number":1`) || !strings.Contains(early.Body.String(), "rules/ok.md") {
		t.Fatalf("record before mail %d %s", early.Code, early.Body.String())
	}
	if strings.Contains(early.Body.String(), "credentials.json") {
		t.Fatalf("snapshot kept a private file %s", early.Body.String())
	}

	secret := "sk-live-SUPERSECRETVALUE"
	leaked := postAuth(t, e, "/agent/record", token, pushBody("Grok", false, pushFile{"rules/key.md", "token=" + secret}), owner)
	if leaked.Code == http.StatusCreated || !strings.Contains(leaked.Body.String(), "rules/key.md") || strings.Contains(leaked.Body.String(), secret) {
		t.Fatalf("secret %d %s", leaked.Code, leaked.Body.String())
	}
	empty := postAuth(t, e, "/agent/record", token, pushBody("Grok", false, pushFile{"credentials.json", `{"token":"abc"}`}), owner)
	if empty.Code == http.StatusCreated || !strings.Contains(empty.Body.String(), "переносим") {
		t.Fatalf("empty %d %s", empty.Code, empty.Body.String())
	}
	guest := getJSON(t, e, "/agent/store/Grok", nil)
	if guest.Code != http.StatusUnauthorized {
		t.Fatalf("guest %d %s", guest.Code, guest.Body.String())
	}

	second := postAuth(t, e, "/agent/record", token, pushBody("Grok", false,
		pushFile{"rules/ok.md", "hello"},
		pushFile{"credentials.json", `{"token":"abc"}`},
		pushFile{"rules/next.md", "again"},
	), owner)
	if second.Code != http.StatusCreated || !strings.Contains(second.Body.String(), `"number":2`) || !strings.Contains(second.Body.String(), `"current":true`) {
		t.Fatalf("second %d %s", second.Code, second.Body.String())
	}

	next := mustServer(t, db)
	list := getJSON(t, next, "/agent/store/Grok", owner)
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"number":1`) || !strings.Contains(list.Body.String(), `"number":2`) {
		t.Fatalf("list after restart %d %s", list.Code, list.Body.String())
	}
	if strings.Contains(list.Body.String(), "rules/ok.md") {
		t.Fatalf("list included file bodies %s", list.Body.String())
	}
	first := getJSON(t, next, "/agent/store/Grok/1", owner)
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), "rules/ok.md") || strings.Contains(first.Body.String(), `"current":true`) {
		t.Fatalf("first snapshot %d %s", first.Code, first.Body.String())
	}
	current := getJSON(t, next, "/agent/store/Grok/2", owner)
	if current.Code != http.StatusOK || !strings.Contains(current.Body.String(), "rules/next.md") || !strings.Contains(current.Body.String(), `"current":true`) || strings.Contains(current.Body.String(), "credentials.json") {
		t.Fatalf("current snapshot %d %s", current.Code, current.Body.String())
	}

	other := mustAccount(t, next, "other-keeper@example.com", "secret-pass", "OtherKeeper")
	foreign := getJSON(t, next, "/agent/store/Grok", other)
	if foreign.Code != http.StatusOK || strings.Contains(foreign.Body.String(), `"number":1`) {
		t.Fatalf("other account saw the store %d %s", foreign.Code, foreign.Body.String())
	}
}
