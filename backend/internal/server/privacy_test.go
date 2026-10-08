package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestPrivacyOpensProfileActions(t *testing.T) {
	e := newServer(t)
	owner := mustAccount(t, e, "owner@example.com", "secret-pass", "Owner")
	pushed := postAuth(t, e, "/agent/push", "", pushBody("Grok", false, pushFile{"rules/ok.md", "one"}), owner)
	if pushed.Code != http.StatusCreated {
		t.Fatalf("push %d %s", pushed.Code, pushed.Body.String())
	}
	hidden := getJSON(t, e, "/accounts/Owner/versions/Grok", nil)
	if hidden.Code != http.StatusNotFound {
		t.Fatalf("hidden versions %d %s", hidden.Code, hidden.Body.String())
	}
	saved := postJSON(t, e, "/account/privacy", map[string]bool{
		"copy": true, "view": true, "versions": true,
	}, owner)
	if saved.Code != http.StatusOK {
		t.Fatalf("privacy %d %s", saved.Code, saved.Body.String())
	}
	shown := getJSON(t, e, "/accounts/Owner/versions/Grok", nil)
	if shown.Code != http.StatusOK {
		t.Fatalf("shown versions %d %s", shown.Code, shown.Body.String())
	}
	page := getJSON(t, e, "/accounts/Owner", nil)
	body := page.Body.String()
	if page.Code != http.StatusOK || !strings.Contains(body, `"shareCopy":true`) || !strings.Contains(body, `"shareView":true`) || !strings.Contains(body, `"shareVersions":true`) {
		t.Fatalf("flags %s", body)
	}
}
