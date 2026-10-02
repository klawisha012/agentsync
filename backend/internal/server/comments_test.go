package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestCommentAccess(t *testing.T) {
	e := newServer(t)
	owner := mustAccount(t, e, "owner@example.com", "secret", "Owner")
	guest := mustAccount(t, e, "guest@example.com", "secret", "Guest")
	other := mustAccount(t, e, "other@example.com", "secret", "Other")

	if got := getJSON(t, e, "/accounts/Owner", nil); !strings.Contains(got.Body.String(), `"commentAccess":"hidden"`) {
		t.Fatalf("default access %s", got.Body.String())
	}
	created := postJSON(t, e, "/accounts/Owner/comments", map[string]string{"body": "первая запись"}, owner)
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), "первая запись") {
		t.Fatalf("owner post %d %s", created.Code, created.Body.String())
	}
	hidden := getJSON(t, e, "/accounts/Owner/comments", nil)
	if hidden.Code != http.StatusOK || !strings.Contains(hidden.Body.String(), `"hidden":true`) || strings.Contains(hidden.Body.String(), "первая запись") {
		t.Fatalf("anonymous hidden %s", hidden.Body.String())
	}
	denied := postJSON(t, e, "/accounts/Owner/comments", map[string]string{"body": "чужая"}, guest)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("guest post while hidden %d %s", denied.Code, denied.Body.String())
	}

	opened := postJSON(t, e, "/account/privacy", map[string]any{
		"copy": false, "view": false, "versions": false, "comments": "users",
	}, owner)
	if opened.Code != http.StatusOK {
		t.Fatalf("privacy %d %s", opened.Code, opened.Body.String())
	}
	signedOut := getJSON(t, e, "/accounts/Owner/comments", nil)
	if !strings.Contains(signedOut.Body.String(), `"hidden":true`) || strings.Contains(signedOut.Body.String(), "первая запись") {
		t.Fatalf("users still hides signed-out %s", signedOut.Body.String())
	}
	seen := getJSON(t, e, "/accounts/Owner/comments", guest)
	if seen.Code != http.StatusOK || !strings.Contains(seen.Body.String(), "первая запись") || !strings.Contains(seen.Body.String(), `"canPost":true`) {
		t.Fatalf("guest sees users %s", seen.Body.String())
	}
	guestPost := postJSON(t, e, "/accounts/Owner/comments", map[string]string{"body": "ответ гостя"}, guest)
	if guestPost.Code != http.StatusCreated {
		t.Fatalf("guest post %d %s", guestPost.Code, guestPost.Body.String())
	}
	var posted struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(guestPost.Body.Bytes(), &posted); err != nil || posted.ID == "" {
		t.Fatalf("guest id %v %s", err, guestPost.Body.String())
	}
	forbidden := deleteCookie(t, e, "/accounts/Owner/comments/"+posted.ID, other)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("other delete %d %s", forbidden.Code, forbidden.Body.String())
	}
	removed := deleteCookie(t, e, "/accounts/Owner/comments/"+posted.ID, owner)
	if removed.Code != http.StatusNoContent {
		t.Fatalf("owner delete %d %s", removed.Code, removed.Body.String())
	}

	world := postJSON(t, e, "/account/privacy", map[string]any{"comments": "open"}, owner)
	if world.Code != http.StatusOK {
		t.Fatalf("open %d %s", world.Code, world.Body.String())
	}
	public := getJSON(t, e, "/accounts/Owner/comments", nil)
	if !strings.Contains(public.Body.String(), "первая запись") || strings.Contains(public.Body.String(), `"hidden":true`) {
		t.Fatalf("open list %s", public.Body.String())
	}
	anon := postJSON(t, e, "/accounts/Owner/comments", map[string]string{"body": "без входа"}, nil)
	if anon.Code != http.StatusUnauthorized {
		t.Fatalf("anon post %d %s", anon.Code, anon.Body.String())
	}
}

func TestAddCommentRejects(t *testing.T) {
	e := newServer(t)
	owner := mustAccount(t, e, "owner@example.com", "secret", "Owner")
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "empty", body: "   ", want: http.StatusBadRequest},
		{name: "exact", body: strings.Repeat("я", commentMaxRunes), want: http.StatusCreated},
		{name: "emoji exact", body: strings.Repeat("🔥", commentMaxRunes), want: http.StatusCreated},
		{name: "too long", body: strings.Repeat("я", commentMaxRunes+1), want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postJSON(t, e, "/accounts/Owner/comments", map[string]string{"body": tt.body}, owner)
			if got.Code != tt.want {
				t.Fatalf("status %d %s", got.Code, got.Body.String())
			}
			if tt.want == http.StatusBadRequest && tt.name == "too long" && !strings.Contains(got.Body.String(), "1\u202f000 символов") {
				t.Fatalf("message %s", got.Body.String())
			}
		})
	}
}

func TestCommentPages(t *testing.T) {
	e := newServer(t)
	owner := mustAccount(t, e, "owner@example.com", "secret", "Owner")
	for i := 1; i <= commentPageSize+1; i++ {
		posted := postJSON(t, e, "/accounts/Owner/comments", map[string]string{"body": fmt.Sprintf("запись %d", i)}, owner)
		if posted.Code != http.StatusCreated {
			t.Fatalf("post %d %s", posted.Code, posted.Body.String())
		}
	}
	newest := fmt.Sprintf("запись %d", commentPageSize+1)
	first := getJSON(t, e, "/accounts/Owner/comments?page=1", owner)
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), `"page":1`) || !strings.Contains(first.Body.String(), `"pages":2`) || !strings.Contains(first.Body.String(), newest) || strings.Contains(first.Body.String(), "запись 1\"") {
		t.Fatalf("page 1 %s", first.Body.String())
	}
	second := getJSON(t, e, "/accounts/Owner/comments?page=2", owner)
	if !strings.Contains(second.Body.String(), `"page":2`) || !strings.Contains(second.Body.String(), "запись 1") || strings.Contains(second.Body.String(), newest) {
		t.Fatalf("page 2 %s", second.Body.String())
	}
	clamped := getJSON(t, e, "/accounts/Owner/comments?page=9", owner)
	if !strings.Contains(clamped.Body.String(), `"page":2`) {
		t.Fatalf("clamped %s", clamped.Body.String())
	}
}

func TestCommentUTF8(t *testing.T) {
	e := newServer(t)
	owner := mustAccount(t, e, "owner@example.com", "secret", "Owner")
	const body = "ёжик ♥ 🔥"
	created := postJSON(t, e, "/accounts/Owner/comments", map[string]string{"body": body}, owner)
	if created.Code != http.StatusCreated || !strings.Contains(strings.ToLower(created.Header().Get("Content-Type")), "utf-8") || !strings.Contains(created.Body.String(), body) {
		t.Fatalf("create %d %s %s", created.Code, created.Header().Get("Content-Type"), created.Body.String())
	}
	listed := getJSON(t, e, "/accounts/Owner/comments", owner)
	if !strings.Contains(strings.ToLower(listed.Header().Get("Content-Type")), "utf-8") || !strings.Contains(listed.Body.String(), body) {
		t.Fatalf("list %s %s", listed.Header().Get("Content-Type"), listed.Body.String())
	}
}

func TestPrivacyRejectsUnknownComments(t *testing.T) {
	e := newServer(t)
	owner := mustAccount(t, e, "owner@example.com", "secret", "Owner")
	got := postJSON(t, e, "/account/privacy", map[string]any{"comments": "friends"}, owner)
	if got.Code != http.StatusBadRequest || !strings.Contains(got.Body.String(), "Неизвестный доступ к комментариям.") {
		t.Fatalf("unknown %d %s", got.Code, got.Body.String())
	}
}
