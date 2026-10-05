package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/klawisha012/agentsync/internal/agent"
	"golang.org/x/term"
)

func fillSkillChoice(call skillCall) (skillCall, error) {
	if call.place != "" && len(call.targets) > 0 {
		return call, nil
	}
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		if call.place == "" {
			return call, errors.New("Выберите глобальный каталог или проект.")
		}
		return call, errors.New("Назовите приёмника, куда положить навыки.")
	}
	state, err := term.MakeRaw(fd)
	if err != nil {
		return call, err
	}
	defer term.Restore(fd, state)
	if call.place == "" {
		place, err := promptPlace(os.Stdin, os.Stderr)
		if err != nil {
			return call, err
		}
		call.place = place
	}
	if len(call.targets) == 0 {
		slugs, err := promptReceivers(os.Stdin, os.Stderr, call.place)
		if err != nil {
			return call, err
		}
		call.targets = slugs
	}
	return call, nil
}

type placeChoice struct {
	id    string
	label string
}

func placeChoices() []placeChoice {
	return []placeChoice{
		{id: "global", label: "Во все проекты"},
		{id: "project", label: "В этот проект"},
	}
}

func promptPlace(in *os.File, out io.Writer) (string, error) {
	choices := placeChoices()
	cursor := 0
	height := 0
	for {
		height = drawPlace(out, choices, cursor, height)
		kind, text, err := readKey(in)
		if err != nil {
			return "", err
		}
		switch kind {
		case "up":
			if cursor > 0 {
				cursor--
			}
		case "down":
			if cursor < len(choices)-1 {
				cursor++
			}
		case "enter":
			clearFrame(out, height)
			fmt.Fprintf(out, "Куда поставить: %s\r\n", choices[cursor].label)
			return choices[cursor].id, nil
		case "cancel":
			clearFrame(out, height)
			return "", errors.New("Установка отменена.")
		case "text":
			_ = text
		}
	}
}

type receiverPicker struct {
	all      []agent.ReceiverView
	query    string
	cursor   int
	selected map[string]bool
	order    []string
}

func newReceiverPicker(place string) receiverPicker {
	all := make([]agent.ReceiverView, 0)
	for _, item := range agent.ReceiverViews() {
		if place == "project" && item.Project || place != "project" && item.Global {
			all = append(all, item)
		}
	}
	return receiverPicker{all: all, selected: map[string]bool{}, order: []string{}}
}

func (p receiverPicker) visible() []agent.ReceiverView {
	needle := strings.ToLower(strings.TrimSpace(p.query))
	if needle == "" {
		return p.all
	}
	out := make([]agent.ReceiverView, 0)
	for _, item := range p.all {
		if strings.Contains(strings.ToLower(item.Display), needle) || strings.Contains(item.Slug, needle) {
			out = append(out, item)
		}
	}
	return out
}

func (p *receiverPicker) move(delta int) {
	shown := p.visible()
	if len(shown) == 0 {
		p.cursor = 0
		return
	}
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(shown) {
		p.cursor = len(shown) - 1
	}
}

func (p *receiverPicker) toggle() {
	shown := p.visible()
	if p.cursor < 0 || p.cursor >= len(shown) {
		return
	}
	slug := shown[p.cursor].Slug
	if p.selected[slug] {
		delete(p.selected, slug)
		next := p.order[:0]
		for _, item := range p.order {
			if item != slug {
				next = append(next, item)
			}
		}
		p.order = next
		return
	}
	p.selected[slug] = true
	p.order = append(p.order, slug)
}

func (p *receiverPicker) typeText(text string) {
	p.query += text
	p.cursor = 0
}

func (p *receiverPicker) backspace() {
	if p.query == "" {
		return
	}
	_, size := utf8.DecodeLastRuneInString(p.query)
	p.query = p.query[:len(p.query)-size]
	p.cursor = 0
}

func promptReceivers(in *os.File, out io.Writer, place string) ([]string, error) {
	picker := newReceiverPicker(place)
	height := 0
	warn := ""
	for {
		height = drawReceivers(out, picker, warn, height)
		warn = ""
		kind, text, err := readKey(in)
		if err != nil {
			return nil, err
		}
		switch kind {
		case "up":
			picker.move(-1)
		case "down":
			picker.move(1)
		case "space":
			picker.toggle()
		case "backspace":
			picker.backspace()
		case "text":
			picker.typeText(text)
		case "enter":
			if len(picker.order) == 0 {
				warn = "Выберите хотя бы один приёмник."
				continue
			}
			clearFrame(out, height)
			fmt.Fprintf(out, "Приёмники: %s\r\n", strings.Join(picker.order, ", "))
			return picker.order, nil
		case "cancel":
			clearFrame(out, height)
			return nil, errors.New("Установка отменена.")
		}
	}
}

func drawPlace(out io.Writer, choices []placeChoice, cursor, previous int) int {
	lines := []string{"Куда поставить", "↑↓ ход, Enter подтвердить", ""}
	for i, choice := range choices {
		mark := "○"
		if i == cursor {
			mark = "●"
		}
		lines = append(lines, mark+" "+choice.label)
	}
	return redraw(out, lines, previous)
}

func drawReceivers(out io.Writer, picker receiverPicker, warn string, previous int) int {
	shown := picker.visible()
	lines := []string{
		"Приёмники",
		"Поиск: " + picker.query + "█",
		"↑↓ ход, пробел выбор, Enter подтвердить",
		"",
	}
	const window = 8
	start := 0
	if picker.cursor >= window {
		start = picker.cursor - window + 1
	}
	end := start + window
	if end > len(shown) {
		end = len(shown)
	}
	if len(shown) == 0 {
		lines = append(lines, "Ничего не найдено")
	}
	for i := start; i < end; i++ {
		item := shown[i]
		mark := "○"
		if picker.selected[item.Slug] {
			mark = "●"
		}
		prefix := "  "
		if i == picker.cursor {
			prefix = "❯ "
		}
		lines = append(lines, prefix+mark+" "+item.Display)
	}
	if start > 0 || end < len(shown) {
		lines = append(lines, fmt.Sprintf("ещё %d", len(shown)-end+start))
	}
	if len(picker.order) == 0 {
		lines = append(lines, "Выбрано: нет")
	} else {
		lines = append(lines, "Выбрано: "+strings.Join(picker.order, ", "))
	}
	if warn != "" {
		lines = append(lines, warn)
	}
	return redraw(out, lines, previous)
}

func redraw(out io.Writer, lines []string, previous int) int {
	clearFrame(out, previous)
	for _, line := range lines {
		fmt.Fprintf(out, "%s\r\n", line)
	}
	return len(lines)
}

func clearFrame(out io.Writer, height int) {
	if height <= 0 {
		return
	}
	fmt.Fprintf(out, "\x1b[%dA\x1b[J", height)
}

func readKey(in *os.File) (string, string, error) {
	var buf [16]byte
	n, err := in.Read(buf[:])
	if err != nil {
		return "", "", err
	}
	if n == 0 {
		return "", "", io.EOF
	}
	switch buf[0] {
	case 3:
		return "cancel", "", nil
	case '\r', '\n':
		return "enter", "", nil
	case 8, 127:
		return "backspace", "", nil
	case ' ':
		return "space", "", nil
	case 0x1b:
		if n >= 3 && buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return "up", "", nil
			case 'B':
				return "down", "", nil
			}
		}
		return "", "", nil
	case 0xe0:
		if n >= 2 {
			switch buf[1] {
			case 72:
				return "up", "", nil
			case 80:
				return "down", "", nil
			}
		}
		return "", "", nil
	}
	text := string(buf[:n])
	if !utf8.ValidString(text) {
		return "", "", nil
	}
	for _, r := range text {
		if r < 32 {
			return "", "", nil
		}
	}
	return "text", text, nil
}
