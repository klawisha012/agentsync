package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klawisha012/agentsync/internal/agent"
)

func TestApplyChainWithdrawRenameDelete(t *testing.T) {
	e := newServer(t)
	ownerRec := postJSON(t, e, "/accounts", map[string]string{
		"email": "owner@example.com", "password": "secret-pass", "name": "Owner",
	}, nil)
	owner := readSessionCookie(t, ownerRec)
	postJSON(t, e, "/email/confirm", map[string]string{
		"token": testLetter(t, e, "owner@example.com", "confirm"),
	}, nil)
	machine := postJSON(t, e, "/machine", map[string]string{
		"id": testMachineID("pc-owner"), "host": "desk", "listener": "127.0.0.1:49152",
	}, owner)
	ownerToken := decodeMachine(t, machine.Body.Bytes()).AgentToken

	guestRec := postJSON(t, e, "/accounts", map[string]string{
		"email": "guest@example.com", "password": "secret-pass", "name": "Guest",
	}, nil)
	guest := readSessionCookie(t, guestRec)
	guestMachine := postJSON(t, e, "/machine", map[string]string{
		"id": testMachineID("pc-guest"), "host": "lap", "listener": "127.0.0.1:49152",
	}, guest)
	guestToken := decodeMachine(t, guestMachine.Body.Bytes()).AgentToken

	first := postAuth(t, e, "/agent/push", ownerToken, pushBody("Grok", false, pushFile{"rules/ok.md", "one"}), owner)
	if first.Code != http.StatusCreated {
		t.Fatalf("push 1 %d %s", first.Code, first.Body.String())
	}
	pubID := publicationID(t, first.Body.Bytes())

	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	dir := t.TempDir()
	if err := agent.Install(dir); err != nil {
		t.Fatal(err)
	}
	seedHome(t, dir)
	if err := agent.Apply(t.Context(), ts.URL, dir, "Owner", "Grok", "", ownerToken, ""); err == nil || !strings.Contains(err.Error(), "браузере") {
		t.Fatalf("own apply without a browser session: %v", err)
	}
	if err := agent.Apply(t.Context(), ts.URL, dir, "Owner", "Grok", "", guestToken, ""); err == nil || !strings.Contains(err.Error(), "скрыта") {
		t.Fatalf("hidden apply: %v", err)
	}
	opened := postJSON(t, e, "/account/privacy", map[string]bool{"view": true}, owner)
	if opened.Code != http.StatusOK {
		t.Fatalf("privacy %d %s", opened.Code, opened.Body.String())
	}
	if err := agent.Apply(t.Context(), ts.URL, dir, "Owner", "Grok", "", guestToken, ""); err != nil {
		t.Fatal(err)
	}
	if body := readHome(t, dir, "rules/ok.md"); body != "one" {
		t.Fatalf("applied body %q", body)
	}
	if _, err := os.Stat(filepath.Join(dir, "Grok", "rules", "mine.md")); !os.IsNotExist(err) {
		t.Fatal("file absent from the publication stayed in the agent")
	}
	if readHome(t, dir, "credentials.json") != "secret" || readHome(t, dir, "vendor/app") != "bin" {
		t.Fatal("private files were replaced")
	}
	if readHome(t, dir, "hooks/machine.json") == "" {
		t.Fatal("machine hook was removed")
	}
	snaps := agent.Chain(dir, "Owner", "Grok")
	if len(snaps) != 1 || !hasBody(snaps[0].Files, "rules/mine.md", "local") {
		t.Fatalf("snapshot %+v", snaps)
	}
	if len(agent.Chain(dir, "Guest", "Grok")) != 0 {
		t.Fatal("another account sees the chain")
	}

	second := postAuth(t, e, "/agent/push", ownerToken, pushBody("Grok", false,
		pushFile{"rules/ok.md", "two"}, pushFile{"rules/extra.md", "plus"}), owner)
	if second.Code != http.StatusCreated {
		t.Fatalf("push 2 %s", second.Body.String())
	}
	cookie := sessionCookie + "=" + owner[0].Value
	if err := agent.Apply(t.Context(), ts.URL, dir, "Owner", "Grok", "", ownerToken, cookie); err != nil {
		t.Fatal(err)
	}
	if len(agent.Chain(dir, "Owner", "Grok")) != 2 {
		t.Fatal("second snapshot missing")
	}
	if err := agent.Revert(dir, "Owner", "Grok"); err != nil {
		t.Fatal(err)
	}
	if readHome(t, dir, "rules/ok.md") != "one" {
		t.Fatal("revert did not restore the snapshot")
	}
	if _, err := os.Stat(filepath.Join(dir, "Grok", "rules", "extra.md")); !os.IsNotExist(err) {
		t.Fatal("reverted file stayed")
	}
	kept := getJSON(t, e, "/publications/"+pubID, owner)
	if kept.Code != http.StatusOK || !strings.Contains(kept.Body.String(), "one") {
		t.Fatalf("publication changed %s", kept.Body.String())
	}
	before := len(agent.Chain(dir, "Owner", "Grok"))
	if err := agent.ApplyBroken(t.Context(), ts.URL, dir, "Owner", "Grok", guestToken, ""); err == nil {
		t.Fatal("broken apply succeeded")
	}
	if readHome(t, dir, "rules/ok.md") != "one" || len(agent.Chain(dir, "Owner", "Grok")) != before {
		t.Fatal("broken apply left a snapshot or a new file")
	}

	sessionOnly := t.TempDir()
	seedHome(t, sessionOnly)
	if err := agent.Apply(t.Context(), ts.URL, sessionOnly, "Owner", "Grok", "", "", cookie); err != nil {
		t.Fatal(err)
	}
	if readHome(t, sessionOnly, "rules/ok.md") != "two" {
		t.Fatal("session apply did not use the publication")
	}
	if snaps := agent.Chain(sessionOnly, "Owner", "Grok"); len(snaps) != 1 || !hasBody(snaps[0].Files, "rules/mine.md", "local") {
		t.Fatalf("session apply snapshot %+v", snaps)
	}

	withdrawn := postJSON(t, e, "/publications/"+publicationID(t, second.Body.Bytes())+"/withdraw", map[string]string{}, owner)
	if withdrawn.Code != http.StatusOK {
		t.Fatalf("withdraw %d %s", withdrawn.Code, withdrawn.Body.String())
	}
	if getJSON(t, e, "/publications/"+publicationID(t, second.Body.Bytes()), nil).Code == http.StatusOK {
		t.Fatal("withdrawn preview still opens")
	}
	plain := postJSON(t, e, "/accounts", map[string]string{
		"email": "plain@example.com", "password": "secret-pass", "name": "Plain",
	}, nil)
	blocked := postJSON(t, e, "/publications/"+pubID+"/withdraw", map[string]string{}, readSessionCookie(t, plain))
	if blocked.Code == http.StatusOK || !strings.Contains(blocked.Body.String(), "не найдена") {
		t.Fatalf("foreign withdraw %d %s", blocked.Code, blocked.Body.String())
	}

	renamed := postJSON(t, e, "/account/name", map[string]string{"name": "NextName"}, owner)
	if renamed.Code != http.StatusOK {
		t.Fatalf("rename %d %s", renamed.Code, renamed.Body.String())
	}
	if getJSON(t, e, "/accounts/Owner", nil).Code == http.StatusOK {
		t.Fatal("old name still opens")
	}
	if getJSON(t, e, "/accounts/NextName", nil).Code != http.StatusOK {
		t.Fatal("new name does not open")
	}
	if getJSON(t, e, "/publications/"+pubID, owner).Code != http.StatusOK {
		t.Fatal("publication address changed with the name")
	}

	bad := postJSON(t, e, "/account/delete", map[string]string{"password": "wrong"}, owner)
	if bad.Code == http.StatusOK || getJSON(t, e, "/accounts/NextName", nil).Code != http.StatusOK {
		t.Fatalf("wrong password deleted the account %d", bad.Code)
	}
	gone := postJSON(t, e, "/account/delete", map[string]string{"password": "secret-pass"}, owner)
	if gone.Code != http.StatusOK {
		t.Fatalf("delete %d %s", gone.Code, gone.Body.String())
	}
	list := getJSON(t, e, "/accounts", nil).Body.String()
	if strings.Contains(list, "NextName") {
		t.Fatal("deleted account is still in the list")
	}
	script := getJSON(t, e, "/cli/install.ps1", nil)
	if script.Code != http.StatusOK || strings.Contains(script.Body.String(), "agentsync push") || !strings.Contains(script.Body.String(), "PATH") {
		t.Fatalf("install script %d %s", script.Code, script.Body.String())
	}
	shell := getJSON(t, e, "/cli/install.sh", nil)
	if shell.Code != http.StatusOK || !strings.HasPrefix(shell.Body.String(), "#!/bin/sh\n") || !strings.Contains(shell.Body.String(), "PATH") {
		t.Fatalf("install shell %d %s", shell.Code, shell.Body.String())
	}
}

func seedHome(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"rules/ok.md":        "old",
		"rules/mine.md":      "local",
		"credentials.json":   "secret",
		"hooks/machine.json": "bash /Users/alex/run.sh",
		"vendor/app":         "bin",
	}
	for path, body := range files {
		target := filepath.Join(dir, "Grok", filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func readHome(t *testing.T, dir, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "Grok", filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func hasBody(files []agent.File, path, body string) bool {
	for _, file := range files {
		if file.Path == path && file.Body == body {
			return true
		}
	}
	return false
}
