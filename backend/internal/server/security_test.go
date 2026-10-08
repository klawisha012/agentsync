package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/klawisha012/agentsync/internal/agent"
)

func TestRecoveryHidesTheLetter(t *testing.T) {
	e := newServer(t)
	owner := mustAccount(t, e, "owner@example.com", "secret-pass", "Owner")
	known := postJSON(t, e, "/recovery", map[string]string{"email": "owner@example.com"}, nil)
	unknown := postJSON(t, e, "/recovery", map[string]string{"email": "missing@example.com"}, nil)
	if known.Code != http.StatusOK || unknown.Code != http.StatusOK {
		t.Fatalf("known %d unknown %d", known.Code, unknown.Code)
	}
	if known.Body.String() != unknown.Body.String() || strings.Contains(known.Body.String(), "letterPath") {
		t.Fatalf("recovery bodies %s | %s", known.Body.String(), unknown.Body.String())
	}
	page := getJSON(t, e, "/accounts/Owner", owner)
	if strings.Contains(page.Body.String(), "letterPath") || strings.Contains(page.Body.String(), "/confirm/") {
		t.Fatalf("owner page leaked a letter: %s", page.Body.String())
	}
}

func TestPasswordRejectsShortValue(t *testing.T) {
	e := newServer(t)
	created := postJSON(t, e, "/accounts", map[string]string{
		"email": "short@example.com", "password": "secret", "name": "Shorty",
	}, nil)
	if created.Code != http.StatusBadRequest || !strings.Contains(created.Body.String(), "8") {
		t.Fatalf("short password %d %s", created.Code, created.Body.String())
	}
}

func TestResetDropsEverySession(t *testing.T) {
	e := newServer(t)
	const email = "ada@example.com"
	created := postJSON(t, e, "/accounts", map[string]string{
		"email": email, "password": "secret-pass", "name": "Ada",
	}, nil)
	first := readSessionCookie(t, created)
	secondRec := postJSON(t, e, "/session", map[string]string{"email": email, "password": "secret-pass"}, nil)
	second := readSessionCookie(t, secondRec)
	postJSON(t, e, "/recovery", map[string]string{"email": email}, nil)
	reset := postJSON(t, e, "/recovery/password", map[string]string{
		"token": testLetter(t, e, email, "reset"), "password": "newer-pass",
	}, nil)
	if reset.Code != http.StatusOK {
		t.Fatalf("reset %d %s", reset.Code, reset.Body.String())
	}
	for _, cookie := range [][]*http.Cookie{first, second} {
		view := getJSON(t, e, "/session", cookie)
		if view.Code != http.StatusUnauthorized {
			t.Fatalf("session after reset %d %s", view.Code, view.Body.String())
		}
	}
}

func TestTokenHashMatchesPostgres(t *testing.T) {
	db := testDB(t)
	mustServer(t, db)
	var got string
	if err := db.QueryRow(`SELECT encode(digest('abc', 'sha256'), 'hex')`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got != want || got != hashToken("abc") {
		t.Fatalf("digest %s", got)
	}
}

func TestUnsafePathsAndSecrets(t *testing.T) {
	if portablePath("rules/../../.bashrc") || portablePath("skills/$(calc)/SKILL.md") {
		t.Fatal("unsafe path stayed portable")
	}
	if !privatePath(".env.local") || !privatePath("rules/.env.production") {
		t.Fatal("env file was not secret")
	}
	if _, ok := agent.CleanRel("C:/Users/Public/skills/pwn.txt"); ok {
		t.Fatal("windows path was accepted")
	}
}

func TestShortMachineIDRejected(t *testing.T) {
	e := newServer(t)
	owner := mustAccount(t, e, "box@example.com", "secret-pass", "Box")
	rec := postJSON(t, e, "/machine", map[string]string{"id": "pc-1", "host": "desk"}, owner)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("short id %d %s", rec.Code, rec.Body.String())
	}
}

func TestCrossSiteViewDoesNotCount(t *testing.T) {
	e := newServer(t)
	mustAccount(t, e, "page@example.com", "secret-pass", "Page")
	req := httptest.NewRequest(http.MethodGet, "/accounts/Page", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || viewsOf(t, rec) != 0 {
		t.Fatalf("cross-site view %d %s", rec.Code, rec.Body.String())
	}
	if viewsOf(t, getJSON(t, e, "/accounts/Page", nil)) != 1 {
		t.Fatal("direct view was not counted")
	}
}

func TestPasswordChangeKeepsOnlyTheCurrentSession(t *testing.T) {
	e := newServer(t)
	const email = "keep@example.com"
	created := postJSON(t, e, "/accounts", map[string]string{
		"email": email, "password": "secret-pass", "name": "Keep",
	}, nil)
	current := readSessionCookie(t, created)
	other := readSessionCookie(t, postJSON(t, e, "/session", map[string]string{
		"email": email, "password": "secret-pass",
	}, nil))
	changed := postJSON(t, e, "/account/password", map[string]string{
		"current": "secret-pass", "next": "newer-pass",
	}, current)
	if changed.Code != http.StatusOK {
		t.Fatalf("password %d %s", changed.Code, changed.Body.String())
	}
	if getJSON(t, e, "/session", current).Code != http.StatusOK {
		t.Fatal("current session was dropped")
	}
	if getJSON(t, e, "/session", other).Code == http.StatusOK {
		t.Fatal("other session stayed open")
	}
}
