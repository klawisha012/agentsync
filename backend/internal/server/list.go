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
	Name          string     `json:"name"`
	Version       *int       `json:"version"`
	Files         int        `json:"files,omitempty"`
	PublishedAt   *time.Time `json:"publishedAt,omitempty"`
	PublicationID string     `json:"publicationId,omitempty"`
	Likes         int        `json:"likes"`
	Liked         bool       `json:"liked"`
}

type accountCard struct {
	Name          string      `json:"name"`
	Views         int         `json:"views"`
	Likes         int         `json:"likes"`
	Verified      bool        `json:"verified"`
	Liked         bool        `json:"liked"`
	PublishedAt   *time.Time  `json:"publishedAt"`
	Fresh         bool        `json:"fresh"`
	TopWeek       bool        `json:"topWeek"`
	HasAvatar     bool        `json:"hasAvatar"`
	AvatarUpdated *time.Time  `json:"avatarUpdated,omitempty"`
	Agents        []agentSlot `json:"agents"`
}

func sameAgent(a, b string) bool {
	return foldKey.String(a) == foldKey.String(b)
}

func (a account) agentSlots() []agentSlot {
	latest := map[string]*publication{}
	for _, pub := range a.publications {
		if pub.withdrawn {
			continue
		}
		key := foldKey.String(pub.agent)
		prev := latest[key]
		if prev == nil || pub.version > prev.version {
			latest[key] = pub
		}
	}
	slots := make([]agentSlot, 0, len(exampleAgents)+len(latest))
	seen := map[string]bool{}
	for _, name := range exampleAgents {
		key := foldKey.String(name)
		seen[key] = true
		slots = append(slots, slotFor(name, latest[key]))
	}
	extra := make([]*publication, 0)
	for key, pub := range latest {
		if !seen[key] {
			extra = append(extra, pub)
		}
	}
	slices.SortFunc(extra, func(a, b *publication) int {
		return strings.Compare(foldKey.String(a.agent), foldKey.String(b.agent))
	})
	for _, pub := range extra {
		slots = append(slots, slotFor(pub.agent, pub))
	}
	return slots
}

func (a account) withLikes(slots []agentSlot, viewer *account) []agentSlot {
	for i := range slots {
		slots[i].Likes = a.agentLikeCount(slots[i].Name)
		slots[i].Liked = a.agentLiked(slots[i].Name, viewer)
	}
	return slots
}

func slotFor(name string, pub *publication) agentSlot {
	slot := agentSlot{Name: name}
	if pub == nil {
		return slot
	}
	version := pub.version
	created := pub.created
	slot.Version = &version
	slot.Files = len(pub.files)
	slot.PublishedAt = &created
	slot.PublicationID = pub.id
	return slot
}

func (a account) card() accountCard {
	return accountCard{
		Name:          a.name,
		Views:         a.views,
		Likes:         a.publicationLikes(),
		Verified:      a.verified,
		Fresh:         a.publishedAt == nil,
		HasAvatar:     a.hasAvatar,
		AvatarUpdated: avatarStamp(a.avatarUpdated),
		Agents:        a.agentSlots(),
	}
}

func (s *store) list(query, sortKey, sessionID string) []accountCard {
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
		"accounts": a.accounts.list(c.QueryParam("q"), c.QueryParam("sort"), sessionID(c)),
	})
}
