package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopySkillsPlacesChosenFolders(t *testing.T) {
	var gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apply/zwarder/Grok" {
			t.Errorf("path %s", r.URL.Path)
		}
		gotVersion = r.URL.Query().Get("version")
		raw, err := json.Marshal(publicationPayload{
			Author:  "zwarder",
			Agent:   "Grok",
			Version: 4,
			Files: []File{
				{Path: "skills/parent/SKILL.md", Body: "parent"},
				{Path: "skills/parent/child/SKILL.md", Body: "child"},
				{Path: "skills/parent/child/note.md", Body: "note"},
				{Path: "skills/other/SKILL.md", Body: "other"},
				{Path: "skills/ok/../../secret.txt", Body: "pwn"},
				{Path: "rules/nope.md", Body: "nope"},
			},
		})
		if err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(raw); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	root := t.TempDir()
	home := filepath.Join(root, ".grok")
	write := func(rel, body string) {
		t.Helper()
		target := filepath.Join(home, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("AGENTS.md", "old")
	write("credentials.json", "secret")
	write("skills/keep/SKILL.md", "stay")

	err := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author:  "zwarder",
		Source:  "Grok",
		Version: 4,
		Skills:  []string{"child", "skills/other"},
		Targets: []string{"Grok", "Claude"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotVersion != "4" {
		t.Fatalf("version %q", gotVersion)
	}
	if body := mustRead(t, filepath.Join(home, "AGENTS.md")); body != "old" {
		t.Fatal("agents file changed")
	}
	if body := mustRead(t, filepath.Join(home, "credentials.json")); body != "secret" {
		t.Fatal("credentials changed")
	}
	if body := mustRead(t, filepath.Join(home, "skills", "keep", "SKILL.md")); body != "stay" {
		t.Fatal("unselected local skill changed")
	}
	if body := mustRead(t, filepath.Join(home, "skills", "parent", "child", "SKILL.md")); body != "child" {
		t.Fatal("nested skill was not copied")
	}
	if body := mustRead(t, filepath.Join(home, "skills", "parent", "child", "note.md")); body != "note" {
		t.Fatal("skill file was not copied")
	}
	if body := mustRead(t, filepath.Join(root, ".claude", "skills", "other", "SKILL.md")); body != "other" {
		t.Fatal("second home missed the skill")
	}
	if _, err := os.Stat(filepath.Join(home, "skills", "parent", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("parent skill was copied with the child")
	}
	if _, err := os.Stat(filepath.Join(home, "rules", "nope.md")); !os.IsNotExist(err) {
		t.Fatal("non-skill file was copied")
	}
	if _, err := os.Stat(filepath.Join(home, "secret.txt")); !os.IsNotExist(err) {
		t.Fatal("path outside the skill was written")
	}
	if _, err := os.Stat(filepath.Join(root, "Claude")); !os.IsNotExist(err) {
		t.Fatal("skills wrote ~/Claude instead of ~/.claude")
	}
	snaps := Chain(root, "zwarder", "Grok")
	keptAgents := snapHas(snaps, "AGENTS.md", "old")
	keptSkill := snapHas(snaps, "skills/keep/SKILL.md", "stay")
	if len(snaps) != 1 || !keptAgents || !keptSkill {
		t.Fatalf("snapshot %+v", snaps)
	}
	if err := Revert(root, "zwarder", "Grok"); err != nil {
		t.Fatal(err)
	}
	if body := mustRead(t, filepath.Join(home, "AGENTS.md")); body != "old" {
		t.Fatal("revert lost the previous file")
	}
	if _, err := os.Stat(filepath.Join(home, "skills", "parent", "child", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("revert left the copied skill")
	}
	if body := mustRead(t, filepath.Join(home, "credentials.json")); body != "secret" {
		t.Fatal("revert touched credentials")
	}
}

func TestCopySkillsRejectsMissingAndAmbiguous(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := json.Marshal(publicationPayload{
			Author: "zwarder",
			Agent:  "Grok",
			Files: []File{
				{Path: "skills/a/note/SKILL.md", Body: "a"},
				{Path: "skills/b/note/SKILL.md", Body: "b"},
			},
		})
		if err != nil {
			t.Error(err)
		}
		if _, err := w.Write(raw); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	root := t.TempDir()
	missing := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author:  "zwarder",
		Source:  "Grok",
		Version: 4,
		Skills:  []string{"absent"},
		Targets: []string{"Grok"},
	})
	if missing == nil || !strings.Contains(missing.Error(), "absent") {
		t.Fatalf("missing %v", missing)
	}
	ambiguous := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author:  "zwarder",
		Source:  "Grok",
		Version: 4,
		Skills:  []string{"note"},
		Targets: []string{"Grok"},
	})
	if ambiguous == nil || !strings.Contains(ambiguous.Error(), "путь") {
		t.Fatalf("ambiguous %v", ambiguous)
	}
	exact := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author:  "zwarder",
		Source:  "Grok",
		Version: 4,
		Skills:  []string{"skills/a/note"},
		Targets: []string{"Claude"},
	})
	if exact != nil {
		t.Fatal(exact)
	}
	if body := mustRead(t, filepath.Join(root, ".claude", "skills", "a", "note", "SKILL.md")); body != "a" {
		t.Fatal("path selector missed the skill")
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", "b")); !os.IsNotExist(err) {
		t.Fatal("the other skill was copied")
	}
	bare := CopySkills(t.Context(), srv.URL, root, "", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Skills: []string{"note"}, Targets: []string{"Grok"},
	})
	if bare == nil || !strings.Contains(bare.Error(), "номер версии") {
		t.Fatalf("version %v", bare)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func snapHas(snaps []Snapshot, path, body string) bool {
	if len(snaps) == 0 {
		return false
	}
	for _, file := range snaps[0].Files {
		if file.Path == path && file.Body == body {
			return true
		}
	}
	return false
}
