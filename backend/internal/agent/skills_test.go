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

	silenceReceiverEnv(t)
	err := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author:  "zwarder",
		Source:  "Grok",
		Version: 4,
		Place:   "global",
		Skills:  []string{"child", "skills/other"},
		Targets: []string{"grok", "claude-code"},
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
	if body := mustRead(t, filepath.Join(home, "skills", "child", "SKILL.md")); body != "child" {
		t.Fatal("nested skill was not copied")
	}
	if body := mustRead(t, filepath.Join(home, "skills", "child", "note.md")); body != "note" {
		t.Fatal("skill file was not copied")
	}
	if body := mustRead(t, filepath.Join(root, ".claude", "skills", "other", "SKILL.md")); body != "other" {
		t.Fatal("second receiver missed the skill")
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
		t.Fatal("skills wrote ~/Claude")
	}
	if snaps := Chain(root, "zwarder", "Grok"); len(snaps) != 0 {
		t.Fatalf("install wrote a snapshot %+v", snaps)
	}
	if err := Revert(root, "zwarder", "Grok"); err == nil {
		t.Fatal("revert undid a skill install")
	}
	if body := mustRead(t, filepath.Join(home, "skills", "child", "SKILL.md")); body != "child" {
		t.Fatal("failed revert removed the installed skill")
	}
	if body := mustRead(t, filepath.Join(home, "credentials.json")); body != "secret" {
		t.Fatal("install touched credentials")
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
	silenceReceiverEnv(t)
	missing := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author:  "zwarder",
		Source:  "Grok",
		Version: 4,
		Place:   "global",
		Skills:  []string{"absent"},
		Targets: []string{"grok"},
	})
	if missing == nil || !strings.Contains(missing.Error(), "absent") {
		t.Fatalf("missing %v", missing)
	}
	ambiguous := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author:  "zwarder",
		Source:  "Grok",
		Version: 4,
		Place:   "global",
		Skills:  []string{"note"},
		Targets: []string{"grok"},
	})
	if ambiguous == nil || !strings.Contains(ambiguous.Error(), "путь") {
		t.Fatalf("ambiguous %v", ambiguous)
	}
	exact := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author:  "zwarder",
		Source:  "Grok",
		Version: 4,
		Place:   "global",
		Skills:  []string{"skills/a/note"},
		Targets: []string{"claude-code"},
	})
	if exact != nil {
		t.Fatal(exact)
	}
	if body := mustRead(t, filepath.Join(root, ".claude", "skills", "note", "SKILL.md")); body != "a" {
		t.Fatal("path selector missed the skill")
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", "b")); !os.IsNotExist(err) {
		t.Fatal("the other skill was copied")
	}
	bare := CopySkills(t.Context(), srv.URL, root, "", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Place: "global", Skills: []string{"note"}, Targets: []string{"grok"},
	})
	if bare == nil || !strings.Contains(bare.Error(), "номер версии") {
		t.Fatalf("version %v", bare)
	}
}

func TestCopySkillsUsesReceiverCatalog(t *testing.T) {
	srv := skillServer(t, []File{{Path: "skills/parent/child/SKILL.md", Body: "child"}})
	defer srv.Close()
	root := t.TempDir()
	silenceReceiverEnv(t)

	wind := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Version: 4, Place: "global",
		Skills: []string{"child"}, Targets: []string{"windsurf"},
	})
	if wind != nil {
		t.Fatal(wind)
	}
	if body := mustRead(t, filepath.Join(root, ".codeium", "windsurf", "skills", "child", "SKILL.md")); body != "child" {
		t.Fatal(body)
	}
	if _, err := os.Stat(filepath.Join(root, ".windsurf")); !os.IsNotExist(err) {
		t.Fatal("windsurf used the short home")
	}

	unknown := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Version: 4, Place: "global",
		Skills: []string{"child"}, Targets: []string{"Claude"},
	})
	if unknown == nil || !strings.Contains(unknown.Error(), "не найден") {
		t.Fatalf("claude %v", unknown)
	}
	eve := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Version: 4, Place: "global",
		Skills: []string{"child"}, Targets: []string{"eve"},
	})
	if eve == nil || !strings.Contains(eve.Error(), "глобального") {
		t.Fatalf("eve %v", eve)
	}
	none := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Version: 4,
		Skills: []string{"child"}, Targets: []string{"cursor"},
	})
	if none == nil || !strings.Contains(none.Error(), "глобальный каталог или проект") {
		t.Fatalf("place %v", none)
	}

	proj := t.TempDir()
	t.Chdir(proj)
	shared := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Version: 4, Place: "project",
		Skills: []string{"child"}, Targets: []string{"cursor", "codex", "cursor"},
	})
	if shared != nil {
		t.Fatal(shared)
	}
	if body := mustRead(t, filepath.Join(proj, ".agents", "skills", "child", "SKILL.md")); body != "child" {
		t.Fatal(body)
	}
	surf := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Version: 4, Place: "project",
		Skills: []string{"child"}, Targets: []string{"windsurf", "eve"},
	})
	if surf != nil {
		t.Fatal(surf)
	}
	if body := mustRead(t, filepath.Join(proj, ".windsurf", "skills", "child", "SKILL.md")); body != "child" {
		t.Fatal(body)
	}
	if body := mustRead(t, filepath.Join(proj, "agent", "skills", "child", "SKILL.md")); body != "child" {
		t.Fatal(body)
	}
}

func TestCopySkillsHonorsEnvironmentAndRestores(t *testing.T) {
	srv := skillServer(t, []File{
		{Path: "skills/child/SKILL.md", Body: "new"},
		{Path: "skills/child/note.md", Body: "note"},
	})
	defer srv.Close()
	root := t.TempDir()
	silenceReceiverEnv(t)
	custom := filepath.Join(root, "moved-claude")
	t.Setenv("CLAUDE_CONFIG_DIR", custom)
	moved := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Version: 4, Place: "global",
		Skills: []string{"child"}, Targets: []string{"claude-code"},
	})
	if moved != nil {
		t.Fatal(moved)
	}
	if body := mustRead(t, filepath.Join(custom, "skills", "child", "SKILL.md")); body != "new" {
		t.Fatal(body)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Fatal("env did not move claude")
	}

	t.Setenv("CLAUDE_CONFIG_DIR", "")
	if err := os.MkdirAll(filepath.Join(root, ".clawdbot"), 0o755); err != nil {
		t.Fatal(err)
	}
	claw := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Version: 4, Place: "global",
		Skills: []string{"child"}, Targets: []string{"openclaw"},
	})
	if claw != nil {
		t.Fatal(claw)
	}
	if body := mustRead(t, filepath.Join(root, ".clawdbot", "skills", "child", "SKILL.md")); body != "new" {
		t.Fatal(body)
	}

	cursorSkill := filepath.Join(root, ".cursor", "skills", "child", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(cursorSkill), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cursorSkill, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(root, ".cursor", "skills", "child", "mine.md")
	if err := os.WriteFile(extra, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".codeium"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".codeium", "windsurf"), []byte("block"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Version: 4, Place: "global",
		Skills: []string{"child"}, Targets: []string{"cursor", "windsurf"},
	})
	if broken == nil {
		t.Fatal("expected the second receiver to fail")
	}
	if body := mustRead(t, cursorSkill); body != "old" {
		t.Fatalf("rollback %q", body)
	}
	if _, err := os.Stat(filepath.Join(root, ".cursor", "skills", "child", "note.md")); !os.IsNotExist(err) {
		t.Fatal("rollback left a new file")
	}
	if body := mustRead(t, extra); body != "mine" {
		t.Fatal("rollback touched an extra file")
	}
}

func TestCopySkillsRefusesDuplicateFolderNames(t *testing.T) {
	srv := skillServer(t, []File{
		{Path: "skills/a/note/SKILL.md", Body: "a"},
		{Path: "skills/b/note/SKILL.md", Body: "b"},
	})
	defer srv.Close()
	root := t.TempDir()
	silenceReceiverEnv(t)
	err := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
		Author: "zwarder", Source: "Grok", Version: 4, Place: "global",
		Skills: []string{"skills/a/note", "skills/b/note"}, Targets: []string{"cursor"},
	})
	if err == nil || !strings.Contains(err.Error(), "skills/a/note") || !strings.Contains(err.Error(), "skills/b/note") {
		t.Fatalf("duplicate %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".cursor")); !os.IsNotExist(statErr) {
		t.Fatal("duplicate names wrote a file")
	}
}

func TestEveryGlobalReceiverAcceptsInstall(t *testing.T) {
	srv := skillServer(t, []File{{Path: "skills/child/SKILL.md", Body: "ok"}})
	defer srv.Close()
	root := t.TempDir()
	silenceReceiverEnv(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	if len(receivers) != 78 {
		t.Fatalf("catalog %d", len(receivers))
	}
	if _, ok := FindReceiver("universal"); ok {
		t.Fatal("universal is a receiver")
	}
	for _, item := range receivers {
		if !item.HasGlobal() {
			err := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
				Author: "zwarder", Source: "Grok", Version: 4, Place: "global",
				Skills: []string{"child"}, Targets: []string{item.Slug},
			})
			if err == nil {
				t.Fatalf("%s global succeeded", item.Slug)
			}
			continue
		}
		err := CopySkills(t.Context(), srv.URL, root, "zwarder", "", "", SkillCopy{
			Author: "zwarder", Source: "Grok", Version: 4, Place: "global",
			Skills: []string{"child"}, Targets: []string{item.Slug},
		})
		if err != nil {
			t.Fatalf("%s %v", item.Slug, err)
		}
		dir, _ := item.GlobalDir(root)
		if _, statErr := os.Stat(filepath.Join(dir, "child", "SKILL.md")); statErr != nil {
			t.Fatalf("%s %v", item.Slug, statErr)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".cursor", "skills", "child", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".config", "agents", "skills", "child", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".codeium", "windsurf", "skills", "child", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func skillServer(t *testing.T, files []File) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := json.Marshal(publicationPayload{Author: "zwarder", Agent: "Grok", Version: 4, Files: files})
		if err != nil {
			t.Error(err)
		}
		if _, err := w.Write(raw); err != nil {
			t.Error(err)
		}
	}))
}

func silenceReceiverEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "GROK_HOME", "VIBE_HOME", "HERMES_HOME", "AUTOHAND_HOME", "XDG_CONFIG_HOME"} {
		t.Setenv(name, "")
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
