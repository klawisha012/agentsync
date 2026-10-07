package server

import (
	"math"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type statsLeader struct {
	Name     string `json:"name"`
	Views    int    `json:"views"`
	Verified bool   `json:"verified"`
}

type statsAgent struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type statsShare struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	Pct   int    `json:"pct"`
}

type statsReport struct {
	Period       string        `json:"period"`
	ViewsTotal   int           `json:"viewsTotal"`
	ViewsAverage int           `json:"viewsAverage"`
	ViewsLeader  *statsLeader  `json:"viewsLeader"`
	LikesTotal   int           `json:"likesTotal"`
	LikesAverage int           `json:"likesAverage"`
	LikeRate     float64       `json:"likeRate"`
	Publications int           `json:"publications"`
	TopAgent     *statsAgent   `json:"topAgent"`
	Accounts     int           `json:"accounts"`
	Verified     int           `json:"verified"`
	Fresh        int           `json:"fresh"`
	TopWeek      int           `json:"topWeek"`
	LastActivity *time.Time    `json:"lastActivity"`
	Leaders      []statsLeader `json:"leaders"`
	SharesTotal  int           `json:"sharesTotal"`
	Shares       []statsShare  `json:"shares"`
}

func periodCutoff(period string, now time.Time) (time.Time, bool, bool) {
	switch period {
	case "", "all":
		return time.Time{}, false, true
	case "7":
		return now.AddDate(0, 0, -7), true, true
	case "30":
		return now.AddDate(0, 0, -30), true, true
	case "90":
		return now.AddDate(0, 0, -90), true, true
	default:
		return time.Time{}, false, false
	}
}

func inWindow(at time.Time, start time.Time, bounded bool) bool {
	if !bounded {
		return true
	}
	return !at.IsZero() && !at.Before(start)
}

func (a *account) viewsIn(start time.Time, bounded bool) int {
	if !bounded {
		return a.views
	}
	total := 0
	for key := range a.viewers {
		if inWindow(a.viewedAt[key], start, true) {
			total++
		}
	}
	return total
}

func (a *account) likesIn(start time.Time, bounded bool) int {
	if !bounded {
		return a.publicationLikes()
	}
	seen := map[string]struct{}{}
	total := 0
	for _, pub := range a.publications {
		if pub.withdrawn {
			continue
		}
		key := foldKey.String(pub.agent)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		for _, at := range a.likedAt[key] {
			if inWindow(at, start, true) {
				total++
			}
		}
	}
	return total
}

func (a *account) publicationsIn(start time.Time, bounded bool) []agentSlot {
	slots := a.agentSlots()
	if !bounded {
		return slots
	}
	kept := make([]agentSlot, 0, len(slots))
	for _, slot := range slots {
		if slot.PublishedAt != nil && inWindow(*slot.PublishedAt, start, true) {
			kept = append(kept, slot)
		}
	}
	return kept
}

func roundedAverage(total, count int) int {
	if count <= 0 || total <= 0 {
		return 0
	}
	return (total + count/2) / count
}

func (s *store) stats(period string, now time.Time) (statsReport, int, string) {
	start, bounded, ok := periodCutoff(period, now)
	if !ok {
		return statsReport{}, http.StatusBadRequest, "Неизвестный промежуток."
	}
	if period == "" {
		period = "all"
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	cards := make([]accountCard, 0, len(s.byName))
	leaders := make([]statsLeader, 0, len(s.byName))
	report := statsReport{
		Period:  period,
		Leaders: []statsLeader{},
		Shares:  []statsShare{},
	}
	var last *time.Time
	counts := map[string]int{}
	shareCounts := map[string]int{}
	for _, item := range s.byName {
		card := item.card()
		card.PublishedAt = item.publishedAt
		cards = append(cards, card)
		views := item.viewsIn(start, bounded)
		likes := item.likesIn(start, bounded)
		report.ViewsTotal += views
		report.LikesTotal += likes
		report.Accounts++
		if item.verified {
			report.Verified++
		}
		if item.publishedAt == nil {
			report.Fresh++
		}
		if item.publishedAt != nil && (last == nil || item.publishedAt.After(*last)) {
			stamp := item.publishedAt.UTC()
			last = &stamp
		}
		leaders = append(leaders, statsLeader{Name: item.name, Views: views, Verified: item.verified})
		for _, slot := range item.publicationsIn(start, bounded) {
			report.Publications++
			counts[slot.Name]++
		}
		for _, slot := range item.agentSlots() {
			report.SharesTotal++
			shareCounts[slot.Name]++
		}
	}
	markTopWeek(cards, now)
	for _, card := range cards {
		if card.TopWeek {
			report.TopWeek++
		}
	}
	report.ViewsAverage = roundedAverage(report.ViewsTotal, report.Accounts)
	report.LikesAverage = roundedAverage(report.LikesTotal, report.Accounts)
	if report.ViewsTotal > 0 {
		report.LikeRate = math.Round(float64(report.LikesTotal)/float64(report.ViewsTotal)*1000) / 10
	}
	report.LastActivity = last

	slices.SortFunc(leaders, func(a, b statsLeader) int {
		if a.Views != b.Views {
			return b.Views - a.Views
		}
		return foldedCompare(a.Name, b.Name)
	})
	if len(leaders) > 0 {
		best := leaders[0]
		report.ViewsLeader = &best
	}
	if len(leaders) > 5 {
		leaders = leaders[:5]
	}
	report.Leaders = leaders

	bestName, bestCount := "", 0
	for name, count := range counts {
		if count > bestCount || (count == bestCount && (bestName == "" || foldedCompare(name, bestName) < 0)) {
			bestName, bestCount = name, count
		}
	}
	if bestCount > 0 {
		report.TopAgent = &statsAgent{Name: bestName, Count: bestCount}
	}
	report.Shares = shareRows(shareCounts, report.SharesTotal)
	return report, http.StatusOK, ""
}

func shareRows(counts map[string]int, total int) []statsShare {
	rows := make([]statsShare, 0, len(counts))
	for name, count := range counts {
		pct := 0
		if total > 0 {
			pct = int(math.Round(float64(count) / float64(total) * 100))
		}
		rows = append(rows, statsShare{Name: name, Count: count, Pct: pct})
	}
	slices.SortFunc(rows, func(a, b statsShare) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return foldedCompare(a.Name, b.Name)
	})
	return rows
}

func foldedCompare(a, b string) int {
	return strings.Compare(foldKey.String(a), foldKey.String(b))
}

func (a *app) showStats(c echo.Context) error {
	report, code, msg := a.accounts.stats(c.QueryParam("period"), time.Now().UTC())
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, report)
}
