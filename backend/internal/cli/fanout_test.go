package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/klawisha012/agentsync/internal/agent"
)

func TestFanoutSharesOutermostSkills(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AGENTSYNC_ROOT", root)
	silenceFanoutEnv(t)
	home := filepath.Join(root, ".grok")
	writeRel(t, home, "skills/parent/SKILL.md", "parent")
	writeRel(t, home, "skills/parent/child/SKILL.md", "inner")
	writeRel(t, home, "skills/parent/note.md", "note")
	writeRel(t, home, "plugins/demo/SKILL.md", "demo")
	writeRel(t, home, "skills/wrap/only/SKILL.md", "only")
	writeRel(t, home, "SKILL.md", "home-itself")
	writeRel(t, home, "cache/hidden/SKILL.md", "hidden")
	writeRel(t, home, "mcp.json", "token=abcdefghijklmnop")

	proj := t.TempDir()
	t.Chdir(proj)
	writeRel(t, proj, ".agents/skills/local/SKILL.md", "project")

	code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor", "--into", "windsurf")
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, stderr)
	}

	cursorParent := filepath.Join(root, ".cursor", "skills", "parent")
	if body := readFile(t, filepath.Join(cursorParent, "SKILL.md")); body != "parent" {
		t.Fatalf("receiver skill %q", body)
	}
	if body := readFile(t, filepath.Join(cursorParent, "child", "SKILL.md")); body != "inner" {
		t.Fatalf("inner skill %q", body)
	}
	if _, err := os.Lstat(filepath.Join(root, ".cursor", "skills", "child")); !os.IsNotExist(err) {
		t.Fatal("inner SKILL.md got its own name")
	}
	if body := readFile(t, filepath.Join(root, ".cursor", "skills", "demo", "SKILL.md")); body != "demo" {
		t.Fatalf("skill outside skills/ %q", body)
	}
	if body := readFile(t, filepath.Join(root, ".cursor", "skills", "only", "SKILL.md")); body != "only" {
		t.Fatalf("nested skill without an outer SKILL.md %q", body)
	}
	if _, err := os.Lstat(filepath.Join(root, ".cursor", "skills", "wrap")); !os.IsNotExist(err) {
		t.Fatal("a folder without SKILL.md was shown")
	}
	if _, err := os.Lstat(filepath.Join(root, ".cursor", "skills", "hidden")); !os.IsNotExist(err) {
		t.Fatal("private skill was shown")
	}
	if _, err := os.Lstat(filepath.Join(root, ".codeium", "windsurf", "skills", "parent", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if body := readFile(t, filepath.Join(proj, ".agents", "skills", "local", "SKILL.md")); body != "project" {
		t.Fatalf("project catalog %q", body)
	}
	if _, err := os.Lstat(filepath.Join(root, ".agentsync")); !os.IsNotExist(err) {
		t.Fatal("fanout touched the snapshot chain")
	}

	note := filepath.Join(cursorParent, "note.md")
	if err := os.WriteFile(note, []byte("from-receiver"), 0o644); err != nil {
		t.Fatal(err)
	}
	if body := readFile(t, filepath.Join(home, "skills", "parent", "note.md")); body != "from-receiver" {
		t.Fatalf("canon did not see the receiver write %q", body)
	}
	canonSkill := filepath.Join(home, "skills", "parent", "SKILL.md")
	if err := os.WriteFile(canonSkill, []byte("from-canon"), 0o644); err != nil {
		t.Fatal(err)
	}
	if body := readFile(t, filepath.Join(cursorParent, "SKILL.md")); body != "from-canon" {
		t.Fatalf("receiver did not see the canon write %q", body)
	}
	windsurfSkill := filepath.Join(root, ".codeium", "windsurf", "skills", "parent", "SKILL.md")
	if body := readFile(t, windsurfSkill); body != "from-canon" {
		t.Fatalf("second receiver %q", body)
	}
}

func runFanout(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, nil, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func silenceFanoutEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"CLAUDE_CONFIG_DIR", "CODEX_HOME", "GROK_HOME", "VIBE_HOME", "HERMES_HOME",
		"AUTOHAND_HOME", "XDG_CONFIG_HOME", "AGENTSYNC_TOKEN", "AGENTSYNC_COOKIE",
		"AGENTSYNC_ACCOUNT", "AGENTSYNC_SERVER",
	} {
		t.Setenv(name, "")
	}
	t.Setenv("AGENTSYNC_STATE", t.TempDir())
}

func TestFanoutUsesTheReceiverCatalog(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AGENTSYNC_ROOT", root)
	silenceFanoutEnv(t)
	home := filepath.Join(root, ".grok")
	writeRel(t, home, "skills/parent/SKILL.md", "parent")
	shared := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", shared)
	t.Setenv("CODEX_HOME", shared)

	code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "Cursor", "--into", "claude-code", "--into", "codex")
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, stderr)
	}
	entries, err := os.ReadDir(filepath.Join(shared, "skills"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "parent" {
		t.Fatalf("shared catalog %+v", namesOf(entries))
	}
	if _, err := os.Lstat(filepath.Join(root, ".claude", "skills", "parent")); !os.IsNotExist(err) {
		t.Fatal("environment override was ignored")
	}
	if _, err := os.Lstat(filepath.Join(root, ".cursor", "skills", "parent", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestFanoutHelpSaysAccountIsUnused(t *testing.T) {
	code, stdout, stderr := runFanout(t, "help", "fanout")
	if code != 0 || stderr != "" {
		t.Fatalf("help %d %s", code, stderr)
	}
	if !strings.Contains(stdout, "Аккаунт") || !strings.Contains(stdout, "не нуж") {
		t.Fatalf("help %s", stdout)
	}
	code, stdout, stderr = runFanout(t, "help", "unfanout")
	if code != 0 || !strings.Contains(stdout, "остаётся") {
		t.Fatalf("unfanout help %d %s %s", code, stdout, stderr)
	}
	code, stdout, stderr = runFanout(t)
	if code != 0 || !strings.Contains(stdout, "fanout") || !strings.Contains(stdout, "аккаунт не нужен") {
		t.Fatalf("command list %d %s", code, stdout)
	}
}

func TestFanoutMissingAgent(t *testing.T) {
	for _, args := range [][]string{
		{"fanout"},
		{"fanout", "--into", "cursor"},
		{"unfanout"},
		{"unfanout", "--into", "cursor"},
	} {
		code, _, stderr := runFanout(t, args...)
		if code != 2 || !strings.Contains(stderr, "Назовите ИИ-агента.") {
			t.Fatalf("%v code %d stderr %s", args, code, stderr)
		}
	}
}

func TestFanoutRefusalsLeaveReceiversUntouched(t *testing.T) {
	t.Run("no skills before a missing receiver", func(t *testing.T) {
		root := fanoutRoot(t)
		writeRel(t, filepath.Join(root, ".grok"), "AGENTS.md", "rules")
		code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "nope")
		if code != 1 || !strings.Contains(stderr, "Нет навыков.") {
			t.Fatalf("code %d stderr %s", code, stderr)
		}
		if _, err := os.Lstat(filepath.Join(root, ".cursor")); !os.IsNotExist(err) {
			t.Fatal("refusal created a catalog")
		}
	})

	t.Run("missing receiver before a duplicate name", func(t *testing.T) {
		root := fanoutRoot(t)
		writeTwinSkills(t, filepath.Join(root, ".grok"))
		code, _, stderr := runFanout(t, "fanout", "Grok")
		if code != 1 || !strings.Contains(stderr, "Назовите приёмника.") {
			t.Fatalf("code %d stderr %s", code, stderr)
		}
	})

	t.Run("unknown receiver and a receiver without a global catalog", func(t *testing.T) {
		root := fanoutRoot(t)
		writeRel(t, filepath.Join(root, ".grok"), "skills/parent/SKILL.md", "parent")
		code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "nope")
		if code != 1 || !strings.Contains(stderr, "Приёмник «nope» не найден.") {
			t.Fatalf("code %d stderr %s", code, stderr)
		}
		code, _, stderr = runFanout(t, "fanout", "Grok", "--into", "eve")
		if code != 1 || !strings.Contains(stderr, "У приёмника «Eve» нет глобального каталога.") {
			t.Fatalf("code %d stderr %s", code, stderr)
		}
		if _, err := os.Lstat(filepath.Join(root, ".cursor")); !os.IsNotExist(err) {
			t.Fatal("refusal created a catalog")
		}
	})

	t.Run("duplicate names before a secret", func(t *testing.T) {
		root := fanoutRoot(t)
		home := filepath.Join(root, ".grok")
		writeTwinSkills(t, home)
		writeRel(t, home, "skills/a/note/SKILL.md", "token=abcdefghijklmnop")
		code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor")
		if code != 1 || !strings.Contains(stderr, "Навыки «plugins/b/note» и «skills/a/note» называются одинаково. Оставьте один.") {
			t.Fatalf("code %d stderr %s", code, stderr)
		}
		if strings.Contains(stderr, "abcdefghijklmnop") {
			t.Fatal(stderr)
		}
		if _, err := os.Lstat(filepath.Join(root, ".cursor")); !os.IsNotExist(err) {
			t.Fatal("refusal created a catalog")
		}
	})

	t.Run("a machine file inside a skill does not stop fanout", func(t *testing.T) {
		root := fanoutRoot(t)
		home := filepath.Join(root, ".grok")
		writeRel(t, home, "skills/parent/SKILL.md", "parent")
		writeRel(t, home, "skills/parent/mcp.json", "token=abcdefghijklmnop")
		code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor")
		if code != 0 {
			t.Fatalf("code %d stderr %s", code, stderr)
		}
		if strings.Contains(stderr, "abcdefghijklmnop") {
			t.Fatal(stderr)
		}
		if body := readFile(t, filepath.Join(root, ".cursor", "skills", "parent", "SKILL.md")); body != "parent" {
			t.Fatalf("skill %q", body)
		}
	})

	t.Run("secret before an occupied name", func(t *testing.T) {
		root := fanoutRoot(t)
		home := filepath.Join(root, ".grok")
		writeRel(t, home, "skills/parent/SKILL.md", "token=abcdefghijklmnop")
		catalog := filepath.Join(root, ".cursor", "skills", "parent")
		writeRel(t, catalog, "keep.txt", "stay")
		code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor")
		if code != 1 || !strings.Contains(stderr, "Секрет в\u00a0файле skills/parent/SKILL.md.") {
			t.Fatalf("code %d stderr %s", code, stderr)
		}
		if strings.Contains(stderr, "abcdefghijklmnop") {
			t.Fatal(stderr)
		}
		if body := readFile(t, filepath.Join(catalog, "keep.txt")); body != "stay" {
			t.Fatalf("occupied folder changed %q", body)
		}
	})
}

func TestFanoutOccupiedNameAndCase(t *testing.T) {
	root := fanoutRoot(t)
	home := filepath.Join(root, ".grok")
	writeRel(t, home, "skills/parent/SKILL.md", "parent")
	writeRel(t, home, "skills/other/SKILL.md", "other")
	catalog := filepath.Join(root, ".cursor", "skills")
	writeRel(t, catalog, "parent/keep.txt", "stay")
	windsurf := filepath.Join(root, ".codeium", "windsurf", "skills", "other")
	writeRel(t, windsurf, "keep.txt", "wind")

	code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor", "--into", "windsurf")
	if code != 1 || !strings.Contains(stderr, "У приёмника «Cursor» навык «parent» уже занят.") {
		t.Fatalf("code %d stderr %s", code, stderr)
	}
	if body := readFile(t, filepath.Join(catalog, "parent", "keep.txt")); body != "stay" {
		t.Fatalf("cursor %q", body)
	}
	if body := readFile(t, filepath.Join(windsurf, "keep.txt")); body != "wind" {
		t.Fatalf("windsurf %q", body)
	}
	if _, err := os.Lstat(filepath.Join(catalog, "other")); !os.IsNotExist(err) {
		t.Fatal("a later skill was written after an occupied name")
	}
}

func TestFanoutFolderCaseFollowsTheMachine(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("this check is the Windows folder case")
	}
	root := fanoutRoot(t)
	writeRel(t, filepath.Join(root, ".grok"), "skills/foo/SKILL.md", "skill")
	catalog := filepath.Join(root, ".cursor", "skills")
	writeRel(t, catalog, "Foo/keep.txt", "stay")
	code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor")
	if code != 1 || !strings.Contains(stderr, "уже занят") {
		t.Fatalf("code %d stderr %s", code, stderr)
	}
	info, err := os.Lstat(filepath.Join(catalog, "Foo"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("Foo was replaced")
	}
	if body := readFile(t, filepath.Join(catalog, "Foo", "keep.txt")); body != "stay" {
		t.Fatalf("Foo %q", body)
	}
}

func TestFanoutAgainMatchesTheCurrentCanon(t *testing.T) {
	root := fanoutRoot(t)
	home := filepath.Join(root, ".grok")
	writeRel(t, home, "skills/keep/SKILL.md", "keep")
	writeRel(t, home, "skills/old/SKILL.md", "old")
	if code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor"); code != 0 {
		t.Fatal(stderr)
	}
	if err := os.RemoveAll(filepath.Join(home, "skills", "old")); err != nil {
		t.Fatal(err)
	}
	writeRel(t, home, "skills/new/SKILL.md", "new")
	if code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor"); code != 0 {
		t.Fatal(stderr)
	}
	cursor := filepath.Join(root, ".cursor", "skills")
	if body := readFile(t, filepath.Join(cursor, "new", "SKILL.md")); body != "new" {
		t.Fatalf("new %q", body)
	}
	if body := readFile(t, filepath.Join(cursor, "keep", "SKILL.md")); body != "keep" {
		t.Fatalf("keep %q", body)
	}
	if _, err := os.Lstat(filepath.Join(cursor, "old")); !os.IsNotExist(err) {
		t.Fatal("removed canon skill stayed visible")
	}
}

func TestFanoutRestoresReceiversWhenTheAttemptBreaks(t *testing.T) {
	root := fanoutRoot(t)
	home := filepath.Join(root, ".grok")
	writeRel(t, home, "skills/keep/SKILL.md", "keep")
	writeRel(t, home, "skills/old/SKILL.md", "old")
	if code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor"); code != 0 {
		t.Fatal(stderr)
	}
	if err := os.Remove(filepath.Join(home, "skills", "old", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	writeRel(t, home, "skills/extra/SKILL.md", "extra")
	blocked := filepath.Join(root, ".codeium", "windsurf", "skills")
	if err := os.MkdirAll(filepath.Dir(blocked), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocked, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor", "--into", "windsurf")
	if code == 0 {
		t.Fatal("blocked catalog succeeded")
	}
	if stderr == "" {
		t.Fatal("missing explanation")
	}
	cursor := filepath.Join(root, ".cursor", "skills")
	if _, err := os.Lstat(filepath.Join(cursor, "old")); err != nil {
		t.Fatal("a skill the attempt would have removed is gone")
	}
	if _, err := os.Lstat(filepath.Join(cursor, "extra")); !os.IsNotExist(err) {
		t.Fatal("the broken attempt left a new skill")
	}
	if body := readFile(t, filepath.Join(cursor, "keep", "SKILL.md")); body != "keep" {
		t.Fatalf("keep %q", body)
	}
	info, err := os.Lstat(blocked)
	if err != nil || info.IsDir() {
		t.Fatal("blocked path changed")
	}
	if _, err := os.Lstat(filepath.Join(blocked, "keep")); !os.IsNotExist(err) {
		t.Fatal("skill bytes were copied into the blocked receiver")
	}
}

func TestFanoutRemovesACatalogItCreated(t *testing.T) {
	root := fanoutRoot(t)
	writeRel(t, filepath.Join(root, ".grok"), "skills/keep/SKILL.md", "keep")
	blocked := filepath.Join(root, ".codeium", "windsurf", "skills")
	if err := os.MkdirAll(filepath.Dir(blocked), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocked, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, _ := runFanout(t, "fanout", "Grok", "--into", "cursor", "--into", "windsurf")
	if code == 0 {
		t.Fatal("blocked catalog succeeded")
	}
	if _, err := os.Lstat(filepath.Join(root, ".cursor")); !os.IsNotExist(err) {
		t.Fatal("a catalog created by the attempt stayed behind")
	}
	info, err := os.Lstat(blocked)
	if err != nil || info.IsDir() {
		t.Fatal("blocked path changed")
	}
}

func TestFanoutOccupiedNewNameRestoresThePreviousSet(t *testing.T) {
	root := fanoutRoot(t)
	home := filepath.Join(root, ".grok")
	writeRel(t, home, "skills/keep/SKILL.md", "keep")
	writeRel(t, home, "skills/old/SKILL.md", "old")
	if code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor"); code != 0 {
		t.Fatal(stderr)
	}
	if err := os.Remove(filepath.Join(home, "skills", "old", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	writeRel(t, home, "skills/extra/SKILL.md", "extra")
	writeRel(t, filepath.Join(root, ".codeium", "windsurf", "skills"), "extra/keep.txt", "stay")

	code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor", "--into", "windsurf")
	if code != 1 || !strings.Contains(stderr, "У приёмника «Windsurf» навык «extra» уже занят.") {
		t.Fatalf("code %d stderr %s", code, stderr)
	}
	cursor := filepath.Join(root, ".cursor", "skills")
	if _, err := os.Lstat(filepath.Join(cursor, "old")); err != nil {
		t.Fatal("a skill the attempt would have removed is gone")
	}
	if _, err := os.Lstat(filepath.Join(cursor, "extra")); !os.IsNotExist(err) {
		t.Fatal("occupied name still wrote the new skill")
	}
	if body := readFile(t, filepath.Join(root, ".codeium", "windsurf", "skills", "extra", "keep.txt")); body != "stay" {
		t.Fatalf("occupied %q", body)
	}
}

func TestUnfanoutRemovesOnlyThisHomesRecords(t *testing.T) {
	root := fanoutRoot(t)
	home := filepath.Join(root, ".grok")
	writeRel(t, home, "skills/parent/SKILL.md", "parent")
	if code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor"); code != 0 {
		t.Fatal(stderr)
	}
	cursor := filepath.Join(root, ".cursor", "skills")
	writeRel(t, cursor, "copied/SKILL.md", "copied")
	if err := os.WriteFile(filepath.Join(home, "skills", "parent", "SKILL.md"), []byte("token=abcdefghijklmnop"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runFanout(t, "unfanout", "Grok", "--into", "cursor")
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, stderr)
	}
	if _, err := os.Lstat(filepath.Join(cursor, "parent")); !os.IsNotExist(err) {
		t.Fatal("receiver still sees the skill")
	}
	if body := readFile(t, filepath.Join(home, "skills", "parent", "SKILL.md")); !strings.Contains(body, "token=") {
		t.Fatal("canon was removed")
	}
	if body := readFile(t, filepath.Join(cursor, "copied", "SKILL.md")); body != "copied" {
		t.Fatalf("copied neighbor %q", body)
	}

	if err := os.RemoveAll(filepath.Join(home, "skills", "parent")); err != nil {
		t.Fatal(err)
	}
	writeRel(t, home, "skills/next/SKILL.md", "next")
	if code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor"); code != 0 {
		t.Fatal(stderr)
	}
	if err := os.RemoveAll(filepath.Join(home, "skills", "next")); err != nil {
		t.Fatal(err)
	}
	code, _, stderr = runFanout(t, "unfanout", "Grok", "--into", "cursor")
	if code != 0 {
		t.Fatalf("dangling %d %s", code, stderr)
	}
	if _, err := os.Lstat(filepath.Join(cursor, "next")); !os.IsNotExist(err) {
		t.Fatal("a record whose skill folder is gone stayed")
	}
	if body := readFile(t, filepath.Join(cursor, "copied", "SKILL.md")); body != "copied" {
		t.Fatal("neighbor changed")
	}
	code, _, stderr = runFanout(t, "unfanout", "Grok", "--into", "cursor")
	if code != 0 {
		t.Fatalf("repeat %d %s", code, stderr)
	}
	if body := readFile(t, filepath.Join(cursor, "copied", "SKILL.md")); body != "copied" {
		t.Fatal("repeat changed a neighbor")
	}
	if _, err := os.Lstat(home); err != nil {
		t.Fatal(err)
	}
}

func TestUnfanoutRefusalsMatchFanout(t *testing.T) {
	root := fanoutRoot(t)
	writeRel(t, filepath.Join(root, ".grok"), "skills/parent/SKILL.md", "parent")
	catalog := filepath.Join(root, ".cursor", "skills", "parent")
	writeRel(t, catalog, "keep.txt", "stay")
	code, _, stderr := runFanout(t, "unfanout", "Grok")
	if code != 1 || !strings.Contains(stderr, "Назовите приёмника.") {
		t.Fatalf("code %d stderr %s", code, stderr)
	}
	code, _, stderr = runFanout(t, "unfanout", "Grok", "--into", "nope")
	if code != 1 || !strings.Contains(stderr, "Приёмник «nope» не найден.") {
		t.Fatalf("code %d stderr %s", code, stderr)
	}
	if body := readFile(t, filepath.Join(catalog, "keep.txt")); body != "stay" {
		t.Fatalf("refusal changed the receiver %q", body)
	}
}

func TestUnfanoutSkipsAReceiverWithoutAGlobalCatalog(t *testing.T) {
	root := fanoutRoot(t)
	home := filepath.Join(root, ".grok")
	writeRel(t, home, "skills/parent/SKILL.md", "parent")
	if code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor"); code != 0 {
		t.Fatal(stderr)
	}
	code, _, stderr := runFanout(t, "unfanout", "Grok", "--into", "eve")
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, stderr)
	}
	if body := readFile(t, filepath.Join(root, ".cursor", "skills", "parent", "SKILL.md")); body != "parent" {
		t.Fatalf("cursor changed %q", body)
	}
}

func TestApplyDoesNotToggleFanout(t *testing.T) {
	root := fanoutRoot(t)
	t.Setenv("AGENTSYNC_ACCOUNT", "zwarder")
	home := filepath.Join(root, ".grok")
	writeRel(t, home, "skills/shared/SKILL.md", "before")
	if code, stderr := pipePublication(t, "from-publication"); code != 0 {
		t.Fatal(stderr)
	}
	if _, err := os.Lstat(filepath.Join(root, ".cursor")); !os.IsNotExist(err) {
		t.Fatal("apply made the skill visible")
	}
	if code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor"); code != 0 {
		t.Fatal(stderr)
	}
	if code, stderr := pipePublication(t, "replaced"); code != 0 {
		t.Fatal(stderr)
	}
	shared := filepath.Join(root, ".cursor", "skills", "shared", "SKILL.md")
	if body := readFile(t, shared); body != "replaced" {
		t.Fatalf("receiver after apply %q", body)
	}
	code, _, stderr := runFanout(t, "revert", "Grok")
	if code != 0 {
		t.Fatal(stderr)
	}
	if body := readFile(t, shared); body != "from-publication" {
		t.Fatalf("receiver after revert %q", body)
	}
	if body := readFile(t, filepath.Join(home, "skills", "shared", "SKILL.md")); body != "from-publication" {
		t.Fatalf("canon after revert %q", body)
	}
}

func TestSkillInstallSitsBesideASharedSkill(t *testing.T) {
	root := fanoutRoot(t)
	home := filepath.Join(root, ".grok")
	writeRel(t, home, "skills/shared/SKILL.md", "shared")
	if code, _, stderr := runFanout(t, "fanout", "Grok", "--into", "cursor"); code != 0 {
		t.Fatal(stderr)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := struct {
			Author  string       `json:"author"`
			Agent   string       `json:"agent"`
			Version int          `json:"version"`
			Files   []agent.File `json:"files"`
		}{
			Author: "Author", Agent: "Grok", Version: 4,
			Files: []agent.File{{Path: "skills/extra/SKILL.md", Body: "copied"}},
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Error(err)
		}
		if _, err := w.Write(encoded); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)

	code, _, stderr := runFanout(t, "skills", "Author", "Grok", "--version", "4", "--global", "--into", "cursor", "extra")
	if code != 0 {
		t.Fatalf("install %d %s", code, stderr)
	}
	cursor := filepath.Join(root, ".cursor", "skills")
	if body := readFile(t, filepath.Join(cursor, "extra", "SKILL.md")); body != "copied" {
		t.Fatalf("copy %q", body)
	}
	if err := os.WriteFile(filepath.Join(cursor, "extra", "local.md"), []byte("only-copy"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(home, "skills", "extra")); !os.IsNotExist(err) {
		t.Fatal("the installed skill became the canon record")
	}
	if body := readFile(t, filepath.Join(cursor, "shared", "SKILL.md")); body != "shared" {
		t.Fatalf("shared %q", body)
	}
	if err := os.WriteFile(filepath.Join(cursor, "shared", "SKILL.md"), []byte("from-receiver"), 0o644); err != nil {
		t.Fatal(err)
	}
	if body := readFile(t, filepath.Join(home, "skills", "shared", "SKILL.md")); body != "from-receiver" {
		t.Fatalf("shared record broke %q", body)
	}
}

func fanoutRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("AGENTSYNC_ROOT", root)
	silenceFanoutEnv(t)
	return root
}

func writeTwinSkills(t *testing.T, home string) {
	t.Helper()
	writeRel(t, home, "skills/a/note/SKILL.md", "a")
	writeRel(t, home, "plugins/b/note/SKILL.md", "b")
}

func pipePublication(t *testing.T, body string) (int, string) {
	t.Helper()
	raw := `{"author":"zwarder","agent":"Grok","files":[{"path":"skills/shared/SKILL.md","body":"` + body + `"}]}`
	var stdout, stderr bytes.Buffer
	code := Run(nil, strings.NewReader(raw), &stdout, &stderr)
	return code, stderr.String()
}

func namesOf(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func writeRel(t *testing.T, root, rel, body string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
