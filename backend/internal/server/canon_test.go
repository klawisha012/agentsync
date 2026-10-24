package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/klawisha012/agentsync/internal/agent"
)

func TestCanonCardPushApplyBreakRevert(t *testing.T) {
	e := newServer(t)
	ownerRec := postJSON(t, e, "/accounts", map[string]string{
		"email": "canon@example.com", "password": "secret-pass", "name": "Canon",
	}, nil)
	owner := readSessionCookie(t, ownerRec)
	cookie := sessionCookie + "=" + owner[0].Value
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	t.Setenv("AGENTSYNC_STATE", t.TempDir())

	write := func(root, rel, body string) {
		t.Helper()
		target := filepath.Join(root, ".agents", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	read := func(root, rel string) string {
		raw, err := os.ReadFile(filepath.Join(root, ".agents", filepath.FromSlash(rel)))
		if err != nil {
			return ""
		}
		return string(raw)
	}

	src := t.TempDir()
	write(src, "skills/note/SKILL.md", "note")
	write(src, "AGENTS.md", "rules")
	write(src, ".env", "SECRET=from-author")
	if _, err := agent.Push(t.Context(), ts.URL, src, "agents", "", cookie); err != nil {
		t.Fatal(err)
	}
	page := getJSON(t, e, "/accounts/Canon", nil)
	if names := agentNames(t, page.Body.Bytes()); !slices.Contains(names, "agents") {
		t.Fatalf("canon card missing %v", names)
	}

	dst := t.TempDir()
	write(dst, "skills/old/SKILL.md", "old")
	write(dst, ".env", "MINE=1")
	if err := agent.ApplyBroken(t.Context(), ts.URL, dst, "Canon", "agents", "", cookie); err == nil {
		t.Fatal("broken apply succeeded")
	}
	if read(dst, "skills/old/SKILL.md") != "old" || read(dst, "skills/note/SKILL.md") != "" || len(agent.Chain(dst, "Canon", "agents")) != 0 {
		t.Fatal("broken apply changed the canon or left a snapshot")
	}

	if err := agent.Apply(t.Context(), ts.URL, dst, "Canon", "agents", "", "", cookie); err != nil {
		t.Fatal(err)
	}
	if read(dst, "skills/note/SKILL.md") != "note" || read(dst, "AGENTS.md") != "rules" || read(dst, "skills/old/SKILL.md") != "" {
		t.Fatal("canon is not an exact copy of the card")
	}
	if read(dst, ".env") != "MINE=1" || strings.Contains(read(dst, ".env"), "from-author") {
		t.Fatal(".env travelled or was replaced")
	}
	if err := agent.Revert(dst, "Canon", "agents"); err != nil {
		t.Fatal(err)
	}
	if read(dst, "skills/old/SKILL.md") != "old" || read(dst, "skills/note/SKILL.md") != "" {
		t.Fatal("revert did not bring the canon back")
	}
}
