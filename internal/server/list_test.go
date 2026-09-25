package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestTopWeekFollowsTheLikesLeader(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-24 * time.Hour)
	old := now.Add(-8 * 24 * time.Hour)
	cards := []accountCard{
		{Name: "leader", Likes: 10, PublishedAt: &old},
		{Name: "recent", Likes: 3, PublishedAt: &recent},
		{Name: "plain", Likes: 1},
	}
	markTopWeek(cards, now)
	for _, card := range cards {
		if card.TopWeek {
			t.Fatalf("old leader should leave no badge, got %s", card.Name)
		}
	}
	cards[0].PublishedAt = &recent
	markTopWeek(cards, now)
	if !cards[0].TopWeek || cards[1].TopWeek || cards[2].TopWeek {
		t.Fatalf("badge %+v", cards)
	}
}

func TestAccountListSearchAndSort(t *testing.T) {
	e := New("http://localhost:3000")
	for _, name := range []string{"Nova", "harbor"} {
		rec := postJSON(t, e, "/accounts", map[string]string{
			"email": name + "@example.com", "password": "secret", "name": name,
		}, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("seed %s %d %s", name, rec.Code, rec.Body.String())
		}
	}

	all := getJSON(t, e, "/accounts", nil)
	if all.Code != http.StatusOK {
		t.Fatalf("list %d %s", all.Code, all.Body.String())
	}
	got := decodeNames(t, all.Body.Bytes())
	if !sameNames(got, []string{"harbor", "Nova"}) {
		t.Fatalf("default order %v", got)
	}

	blank := getJSON(t, e, "/accounts?q=", nil)
	if !sameNames(decodeNames(t, blank.Body.Bytes()), []string{"harbor", "Nova"}) {
		t.Fatalf("blank query %s", blank.Body.String())
	}

	hit := getJSON(t, e, "/accounts?q=NOV", nil)
	if !sameNames(decodeNames(t, hit.Body.Bytes()), []string{"Nova"}) {
		t.Fatalf("search %s", hit.Body.String())
	}
	miss := getJSON(t, e, "/accounts?q=secret", nil)
	if len(decodeNames(t, miss.Body.Bytes())) != 0 {
		t.Fatalf("query outside the name matched: %s", miss.Body.String())
	}

	byName := getJSON(t, e, "/accounts?sort=name", nil)
	if !sameNames(decodeNames(t, byName.Body.Bytes()), []string{"harbor", "Nova"}) {
		t.Fatalf("name sort %s", byName.Body.String())
	}
	byTime := getJSON(t, e, "/accounts?sort=time", nil)
	timed := decodeCards(t, byTime.Body.Bytes())
	if len(timed) != 2 || timed[0].PublishedAt != nil || timed[1].PublishedAt != nil {
		t.Fatalf("unpublished should stay in the time sort: %+v", timed)
	}
	if !sameNames([]string{timed[0].Name, timed[1].Name}, []string{"harbor", "Nova"}) {
		t.Fatalf("time sort %s", byTime.Body.String())
	}
}

type listCard struct {
	Name        string  `json:"name"`
	PublishedAt *string `json:"publishedAt"`
	Agents      []struct {
		Name    string `json:"name"`
		Version *int   `json:"version"`
	} `json:"agents"`
}

func decodeNames(t *testing.T, raw []byte) []string {
	t.Helper()
	cards := decodeCards(t, raw)
	names := make([]string, 0, len(cards))
	for _, card := range cards {
		names = append(names, card.Name)
		if card.VersionPublished(t) {
			t.Fatalf("empty account published a slot: %+v", card)
		}
	}
	return names
}

func (c listCard) VersionPublished(t *testing.T) bool {
	t.Helper()
	for _, agent := range c.Agents {
		if agent.Version != nil {
			return true
		}
	}
	return false
}

func decodeCards(t *testing.T, raw []byte) []listCard {
	t.Helper()
	var body struct {
		Accounts []listCard `json:"accounts"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	return body.Accounts
}

func sameNames(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
