package server

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

var exampleAgents = []string{"Grok", "Agents", "Claude"}

type agentSlot struct {
	Name    string `json:"name"`
	Version *int   `json:"version"`
}

type accountCard struct {
	Name        string      `json:"name"`
	Views       int         `json:"views"`
	Likes       int         `json:"likes"`
	Verified    bool        `json:"verified"`
	PublishedAt *time.Time  `json:"publishedAt"`
	Fresh       bool        `json:"fresh"`
	TopWeek     bool        `json:"topWeek"`
	Agents      []agentSlot `json:"agents"`
}

func (a account) card() accountCard {
	slots := make([]agentSlot, 0, len(exampleAgents))
	for _, name := range exampleAgents {
		slots = append(slots, agentSlot{Name: name})
	}
	return accountCard{
		Name:     a.name,
		Views:    a.views,
		Likes:    a.likes,
		Verified: a.verified,
		Fresh:    a.publishedAt == nil,
		Agents:   slots,
	}
}

func (s *store) list(query, sortKey string) []accountCard {
	foldedQuery := foldKey.String(strings.TrimSpace(query))
	s.mu.Lock()
	all := make([]accountCard, 0, len(s.byName))
	for _, item := range s.byName {
		card := item.card()
		card.PublishedAt = item.publishedAt
		all = append(all, card)
	}
	s.mu.Unlock()
	markTopWeek(all, time.Now())
	cards := make([]accountCard, 0, len(all))
	for _, card := range all {
		if foldedQuery != "" && !strings.Contains(foldKey.String(card.Name), foldedQuery) {
			continue
		}
		cards = append(cards, card)
	}
	sortCards(cards, sortKey)
	return cards
}

func sortCards(cards []accountCard, sortKey string) {
	slices.SortFunc(cards, func(a, b accountCard) int {
		if cmp := compareCards(a, b, sortKey); cmp != 0 {
			return cmp
		}
		return strings.Compare(foldKey.String(a.Name), foldKey.String(b.Name))
	})
}

func compareCards(a, b accountCard, sortKey string) int {
	switch sortKey {
	case "views":
		return cmpDesc(a.Views, b.Views)
	case "name":
		return 0
	case "time":
		return compareTime(a.PublishedAt, b.PublishedAt)
	default:
		if cmp := cmpDesc(a.Likes, b.Likes); cmp != 0 {
			return cmp
		}
		return cmpDesc(a.Views, b.Views)
	}
}

func compareTime(a, b *time.Time) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	return b.Compare(*a)
}

func cmpDesc(a, b int) int {
	return b - a
}

func markTopWeek(cards []accountCard, now time.Time) {
	best := -1
	for i := range cards {
		if best == -1 || likesAhead(cards[i], cards[best]) {
			best = i
		}
	}
	if best < 0 {
		return
	}
	at := cards[best].PublishedAt
	if at == nil || now.Sub(*at) > 7*24*time.Hour || now.Before(*at) {
		return
	}
	cards[best].TopWeek = true
}

func likesAhead(a, b accountCard) bool {
	if a.Likes != b.Likes {
		return a.Likes > b.Likes
	}
	if a.Views != b.Views {
		return a.Views > b.Views
	}
	return foldKey.String(a.Name) < foldKey.String(b.Name)
}

func (a *app) listAccounts(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"accounts": a.accounts.list(c.QueryParam("q"), c.QueryParam("sort")),
	})
}
