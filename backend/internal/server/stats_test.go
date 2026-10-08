package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPeriodWindowSkipsUndatedViews(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	start, bounded, ok := periodCutoff("7", now)
	if !ok || !bounded {
		t.Fatal("7 days should be a bounded window")
	}
	page := &account{
		views:    2,
		viewers:  map[string]struct{}{"old": {}, "new": {}},
		viewedAt: map[string]time.Time{"new": now.Add(-time.Hour)},
	}
	if got := page.viewsIn(start, true); got != 1 {
		t.Fatalf("dated views %d", got)
	}
	if got := page.viewsIn(time.Time{}, false); got != 2 {
		t.Fatalf("all views %d", got)
	}
	if _, _, ok := periodCutoff("year", now); ok {
		t.Fatal("unknown period was accepted")
	}
}

func TestStatsPeriods(t *testing.T) {
	e := newServer(t)
	unknown := getJSON(t, e, "/stats?period=year", nil)
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown period %d %s", unknown.Code, unknown.Body.String())
	}
	blocked := postJSON(t, e, "/accounts", map[string]string{
		"email": "docs@example.com", "password": "secret", "name": "docs",
	}, nil)
	if blocked.Code != http.StatusConflict {
		t.Fatalf("docs name %d %s", blocked.Code, blocked.Body.String())
	}

	alice := mustAccount(t, e, "alice@example.com", "secret", "Alice")
	bob := mustAccount(t, e, "bob@example.com", "secret", "Bob")
	pushed := postAuth(t, e, "/agent/push", "", pushBody("Grok", false, pushFile{"rules/ok.md", "one"}), alice)
	if pushed.Code != http.StatusCreated {
		t.Fatalf("push %d %s", pushed.Code, pushed.Body.String())
	}
	if got := getJSON(t, e, "/accounts/Alice", bob); viewsOf(t, got) != 1 {
		t.Fatalf("view %s", got.Body.String())
	}
	liked := postJSON(t, e, "/accounts/Alice/agents/Grok/like", map[string]string{}, bob)
	if liked.Code != http.StatusOK {
		t.Fatalf("like %d %s", liked.Code, liked.Body.String())
	}

	for _, period := range []string{"7", "all"} {
		rec := getJSON(t, e, "/stats?period="+period, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %d %s", period, rec.Code, rec.Body.String())
		}
		var body statsReport
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Period != period || body.Accounts != 2 || body.ViewsTotal != 1 || body.LikesTotal != 1 {
			t.Fatalf("%s totals %+v", period, body)
		}
		if body.Publications != 1 || body.SharesTotal != 1 || len(body.Shares) != 1 || body.Shares[0].Name != "Grok" || body.Shares[0].Pct != 100 {
			t.Fatalf("%s publications %+v", period, body)
		}
		if body.ViewsLeader == nil || body.ViewsLeader.Name != "Alice" || body.ViewsLeader.Views != 1 {
			t.Fatalf("%s leader %+v", period, body.ViewsLeader)
		}
		if body.TopAgent == nil || body.TopAgent.Name != "Grok" || body.LikeRate != 100 {
			t.Fatalf("%s agent %+v rate %v", period, body.TopAgent, body.LikeRate)
		}
		if body.Fresh != 1 || body.TopWeek != 1 || body.LastActivity == nil {
			t.Fatalf("%s catalog %+v", period, body)
		}
	}
}

func TestStatsWindowSplitsDatedActivity(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -20)
	recent := now.Add(-time.Hour)
	page := &account{
		name:    "Alice",
		views:   2,
		viewers: map[string]struct{}{"old": {}, "new": {}},
		viewedAt: map[string]time.Time{
			"old": old,
			"new": recent,
		},
		publications: []*publication{
			{agent: "Grok", version: 1, created: old},
			{agent: "Grok", version: 2, created: recent},
			{agent: "Claude", version: 1, created: old, withdrawn: true},
		},
	}
	quiet := &account{name: "Bob", viewers: map[string]struct{}{}, viewedAt: map[string]time.Time{}}
	store := &store{byName: map[string]*account{"alice": page, "bob": quiet}}

	week, code, msg := store.stats("7", now)
	if code != http.StatusOK || msg != "" {
		t.Fatalf("week %d %s", code, msg)
	}
	if week.ViewsTotal != 1 || week.Publications != 1 || week.SharesTotal != 1 {
		t.Fatalf("week totals %+v", week)
	}
	if week.ViewsLeader == nil || week.ViewsLeader.Name != "Alice" || week.ViewsLeader.Views != 1 || len(week.Leaders) != 1 {
		t.Fatalf("week leader %+v list %+v", week.ViewsLeader, week.Leaders)
	}
	if week.TopAgent == nil || week.TopAgent.Name != "Grok" || week.TopAgent.Count != 1 {
		t.Fatalf("week agent %+v", week.TopAgent)
	}

	all, code, msg := store.stats("all", now)
	if code != http.StatusOK || msg != "" {
		t.Fatalf("all %d %s", code, msg)
	}
	if all.ViewsTotal != 2 || all.Publications != 2 || all.SharesTotal != 2 {
		t.Fatalf("all totals %+v", all)
	}
	if len(all.Leaders) != 2 || all.Leaders[0].Name != "Alice" {
		t.Fatalf("all leaders %+v", all.Leaders)
	}
}

func TestStatsLeaderCarriesAvatar(t *testing.T) {
	e := newServer(t)
	owner := mustAccount(t, e, "owner@example.com", "secret", "Owner")
	saved := postFile(t, e, "/account/avatar", "a.png", "image/png", onePixelPNG(t), owner)
	if saved.Code != http.StatusOK {
		t.Fatalf("avatar %d %s", saved.Code, saved.Body.String())
	}
	rec := getJSON(t, e, "/stats?period=all", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"Owner"`) || !strings.Contains(rec.Body.String(), `"hasAvatar":true`) || !strings.Contains(rec.Body.String(), `"avatarUpdated"`) {
		t.Fatalf("stats %d %s", rec.Code, rec.Body.String())
	}
}
