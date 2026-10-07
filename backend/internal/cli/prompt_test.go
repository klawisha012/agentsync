package cli

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/term"
)

func TestFillSkillChoiceNeedsATerminal(t *testing.T) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is a terminal")
	}
	_, err := fillSkillChoice(skillCall{author: "Author", source: "Grok", version: 4, skills: []string{"ru-text"}})
	if err == nil || !strings.Contains(err.Error(), "глобальный каталог или проект") {
		t.Fatalf("place %v", err)
	}
	_, err = fillSkillChoice(skillCall{author: "Author", source: "Grok", version: 4, place: "global", skills: []string{"ru-text"}})
	if err == nil || !strings.Contains(err.Error(), "приёмника") {
		t.Fatalf("targets %v", err)
	}
}

func TestReceiverPickerSearchAndToggle(t *testing.T) {
	picker := newReceiverPicker("global")
	picker.typeText("cursor")
	shown := picker.visible()
	if len(shown) != 1 || shown[0].Slug != "cursor" {
		t.Fatalf("search %+v", shown)
	}
	picker.toggle()
	if !picker.selected["cursor"] || len(picker.order) != 1 {
		t.Fatalf("select %+v", picker.order)
	}
	picker.toggle()
	if picker.selected["cursor"] || len(picker.order) != 0 {
		t.Fatal("toggle off failed")
	}
	picker.backspace()
	if picker.query != "curso" {
		t.Fatal(picker.query)
	}

	project := newReceiverPicker("project")
	project.typeText("eve")
	found := project.visible()
	if len(found) != 1 || found[0].Slug != "eve" {
		t.Fatalf("eve %+v", found)
	}
	global := newReceiverPicker("global")
	global.typeText("eve")
	if len(global.visible()) != 0 {
		t.Fatalf("eve global %+v", global.visible())
	}
	anyPlace := newReceiverPicker("")
	anyPlace.typeText("eve")
	if len(anyPlace.visible()) != 1 || anyPlace.visible()[0].Slug != "eve" {
		t.Fatalf("eve any %+v", anyPlace.visible())
	}
}

func TestPlaceFollowsSelectedReceivers(t *testing.T) {
	both := placeChoicesFor([]string{"cursor"})
	if len(both) != 2 || both[0].id != "project" || both[1].id != "global" {
		t.Fatalf("cursor %+v", both)
	}
	onlyProject := placeChoicesFor([]string{"cursor", "eve"})
	if len(onlyProject) != 1 || onlyProject[0].id != "project" {
		t.Fatalf("with eve %+v", onlyProject)
	}
}
