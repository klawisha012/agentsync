package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func connectRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("AGENTSYNC_ROOT", root)
	silenceAgentEnv(t)
	t.Setenv("AGENTSYNC_ACCOUNT", "zwarder")
	return root
}

func TestOnShowsOutermostCanonSkills(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "skills/parent/SKILL.md", "parent")
	writeRel(t, canon, "skills/parent/child/SKILL.md", "inner")
	writeRel(t, canon, "skills/wrap/only/SKILL.md", "only")
	writeRel(t, canon, "plugins/demo/SKILL.md", "demo")

	code, _, stderr := runCLI(t, "cursor", "on")
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, stderr)
	}
	skills := filepath.Join(root, ".cursor", "skills")
	if body := readFile(t, filepath.Join(skills, "parent", "SKILL.md")); body != "parent" {
		t.Fatalf("parent %q", body)
	}
	if body := readFile(t, filepath.Join(skills, "only", "SKILL.md")); body != "only" {
		t.Fatalf("only %q", body)
	}
	for _, name := range []string{"child", "wrap", "demo"} {
		if _, err := os.Lstat(filepath.Join(skills, name)); !os.IsNotExist(err) {
			t.Fatalf("%s shown", name)
		}
	}
	writeRel(t, canon, "skills/parent/SKILL.md", "edited")
	if body := readFile(t, filepath.Join(skills, "parent", "SKILL.md")); body != "edited" {
		t.Fatalf("edit in canon not visible: %q", body)
	}
	if _, err := os.Stat(filepath.Join(root, ".agentsync", "chains")); !os.IsNotExist(err) {
		t.Fatal("snapshot without an occupied name")
	}
}

func TestOnAcceptsACatalogSynonym(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "skills/note/SKILL.md", "note")
	if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if body := readFile(t, filepath.Join(root, ".claude", "skills", "note", "SKILL.md")); body != "note" {
		t.Fatalf("note %q", body)
	}
}

func TestOnSnapshotsAnOccupiedNameThenLinks(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "skills/grill-me/SKILL.md", "canon")
	agentSkills := filepath.Join(root, ".cursor", "skills")
	writeRel(t, agentSkills, "grill-me/SKILL.md", "mine")
	writeRel(t, agentSkills, "cursor-only/SKILL.md", "own")

	if code, _, stderr := runCLI(t, "cursor", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if body := readFile(t, filepath.Join(agentSkills, "grill-me", "SKILL.md")); body != "canon" {
		t.Fatalf("grill-me %q", body)
	}
	if body := readFile(t, filepath.Join(agentSkills, "cursor-only", "SKILL.md")); body != "own" {
		t.Fatalf("own skill %q", body)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "cursor-only")); !os.IsNotExist(err) {
		t.Fatal("own skill moved into the canon")
	}
	saved := filepath.Join(root, ".agentsync", "chains", "zwarder", "cursor", "0001", "skills", "grill-me", "SKILL.md")
	if body := readFile(t, saved); body != "mine" {
		t.Fatalf("snapshot %q", body)
	}
}

func TestOnAgainFollowsTheCanon(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "skills/keep/SKILL.md", "keep")
	writeRel(t, canon, "skills/old/SKILL.md", "old")
	if code, _, stderr := runCLI(t, "cursor", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if err := os.RemoveAll(filepath.Join(canon, "skills", "old")); err != nil {
		t.Fatal(err)
	}
	writeRel(t, canon, "skills/new/SKILL.md", "new")
	if code, _, stderr := runCLI(t, "cursor", "on"); code != 0 {
		t.Fatal(stderr)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".cursor", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(namesOf(entries), ","); got != "keep,new" {
		t.Fatalf("names %s", got)
	}
}

func TestOffRemovesOnlyCanonLinks(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "skills/shared/SKILL.md", "shared")
	writeRel(t, canon, "skills/gone/SKILL.md", "gone")
	agentSkills := filepath.Join(root, ".cursor", "skills")
	writeRel(t, agentSkills, "installed/SKILL.md", "installed")
	if code, _, stderr := runCLI(t, "cursor", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if err := os.RemoveAll(filepath.Join(canon, "skills", "gone")); err != nil {
		t.Fatal(err)
	}

	if code, _, stderr := runCLI(t, "cursor", "off"); code != 0 {
		t.Fatal(stderr)
	}
	entries, err := os.ReadDir(agentSkills)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(namesOf(entries), ","); got != "installed" {
		t.Fatalf("names %s", got)
	}
	if body := readFile(t, filepath.Join(canon, "skills", "shared", "SKILL.md")); body != "shared" {
		t.Fatalf("canon changed %q", body)
	}
	if code, _, stderr := runCLI(t, "cursor", "off"); code != 0 {
		t.Fatalf("second off %s", stderr)
	}
	if code, _, stderr := runCLI(t, "windsurf", "off"); code != 0 {
		t.Fatalf("off without links %s", stderr)
	}
}

func TestOnRefusalsLeaveTheAgentUntouched(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	agentSkills := filepath.Join(root, ".cursor", "skills")
	writeRel(t, agentSkills, "mine/SKILL.md", "mine")
	untouched := func(name string) {
		t.Helper()
		entries, err := os.ReadDir(agentSkills)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(namesOf(entries), ","); got != "mine" {
			t.Fatalf("%s changed the agent: %s", name, got)
		}
	}

	code, _, stderr := runCLI(t, "cursor", "on")
	if code != 1 || !strings.Contains(stderr, "каноне нечего показать.") {
		t.Fatalf("empty %d %s", code, stderr)
	}
	untouched("empty")

	writeRel(t, canon, "skills/ok/SKILL.md", "ok")
	code, _, stderr = runCLI(t, "nope", "on")
	if code != 1 || !strings.Contains(stderr, "ИИ-агента «nope» нет в") {
		t.Fatalf("unknown %d %s", code, stderr)
	}

	writeRel(t, canon, "skills/a/note/SKILL.md", "a")
	writeRel(t, canon, "skills/b/note/SKILL.md", "b")
	code, _, stderr = runCLI(t, "cursor", "on")
	if code != 1 || !strings.Contains(stderr, "note") {
		t.Fatalf("twins %d %s", code, stderr)
	}
	untouched("twins")
	if err := os.RemoveAll(filepath.Join(canon, "skills", "b")); err != nil {
		t.Fatal(err)
	}

	secret := "EXAMPLEKEY99ABCDEF"
	writeRel(t, canon, "skills/ok/config.json", `{"token":"`+secret+`"}`)
	code, _, stderr = runCLI(t, "cursor", "on")
	if code != 1 || !strings.Contains(stderr, "skills/ok/config.json") || strings.Contains(stderr, secret) {
		t.Fatalf("secret %d %s", code, stderr)
	}
	untouched("secret")
}

func TestOnLeavesTheAgentAsItWasWhenTheAttemptBreaks(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "skills/ok/SKILL.md", "ok")
	blocked := filepath.Join(root, ".cursor", "skills")
	if err := os.MkdirAll(filepath.Dir(blocked), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocked, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runCLI(t, "cursor", "on")
	if code == 0 || stderr == "" {
		t.Fatalf("blocked %d %q", code, stderr)
	}
	if info, err := os.Lstat(blocked); err != nil || info.IsDir() {
		t.Fatal("blocked path changed")
	}
}

func TestOnHelpSaysAccountIsUnused(t *testing.T) {
	connectRoot(t)
	for _, name := range []string{"on", "off"} {
		code, stdout, _ := runCLI(t, "help", name)
		if code != 0 || !strings.Contains(stdout, "agentsync <ии-агент> "+name) || !strings.Contains(stdout, "Аккаунт") {
			t.Fatalf("%s help %d %s", name, code, stdout)
		}
	}
	if code, _, stderr := runCLI(t, "fanout", "Grok", "--into", "cursor"); code != 2 || !strings.Contains(stderr, "Неизвестная команда") {
		t.Fatalf("fanout still runs %d %s", code, stderr)
	}
}

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, nil, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func silenceAgentEnv(t *testing.T) {
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

func TestPushOfAConnectedAgentCarriesCanonCopies(t *testing.T) {
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		sent = string(raw)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "skills/note/SKILL.md", "from-canon")
	writeRel(t, filepath.Join(root, ".agents"), "AGENTS.md", "rules")
	writeRel(t, filepath.Join(root, ".cursor"), "rules/own.md", "own")
	outside := t.TempDir()
	writeRel(t, outside, "SKILL.md", "elsewhere")
	if err := os.Symlink(outside, filepath.Join(root, ".cursor", "skills-outside")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, ".agents", "AGENTS.md"), filepath.Join(root, ".cursor", "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := runCLI(t, "cursor", "on"); code != 0 {
		t.Fatal(stderr)
	}
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")

	if code, _, stderr := runCLI(t, "push", "cursor"); code != 0 {
		t.Fatalf("push %s", stderr)
	}
	for _, want := range []string{`"skills/note/SKILL.md"`, "from-canon", `"AGENTS.md"`, `"rules/own.md"`} {
		if !strings.Contains(sent, want) {
			t.Fatalf("snapshot lacks %s: %s", want, sent)
		}
	}
	if strings.Contains(sent, "elsewhere") {
		t.Fatalf("link outside the canon followed: %s", sent)
	}
}

func TestOnLeavesTheCanonAloneForAnAgentThatReadsIt(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "skills/keep/SKILL.md", "keep")
	if code, _, stderr := runCLI(t, "cline", "on"); code != 0 {
		t.Fatalf("on %s", stderr)
	}
	if code, _, _ := runCLI(t, "cline", "off"); code != 1 {
		t.Fatal("off inside the canon succeeded")
	}
	info, err := os.Lstat(filepath.Join(canon, "skills", "keep"))
	if err != nil || info.Mode()&os.ModeSymlink != 0 || readFile(t, filepath.Join(canon, "skills", "keep", "SKILL.md")) != "keep" {
		t.Fatalf("canon skill changed: %v", err)
	}
}

func TestRevertAfterOnBringsTheOwnFolderBack(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "skills/grill-me/SKILL.md", "canon")
	agentSkills := filepath.Join(root, ".cursor", "skills")
	writeRel(t, agentSkills, "grill-me/SKILL.md", "mine")
	writeRel(t, agentSkills, "grill-me/node_modules/dep.js", "dep")
	if code, _, stderr := runCLI(t, "cursor", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if code, _, stderr := runCLI(t, "revert", "cursor"); code != 0 {
		t.Fatalf("revert %s", stderr)
	}
	info, err := os.Lstat(filepath.Join(agentSkills, "grill-me"))
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("grill-me is still a link: %v", err)
	}
	if body := readFile(t, filepath.Join(agentSkills, "grill-me", "SKILL.md")); body != "mine" {
		t.Fatalf("own skill %q", body)
	}
	if body := readFile(t, filepath.Join(agentSkills, "grill-me", "node_modules", "dep.js")); body != "dep" {
		t.Fatalf("private file lost %q", body)
	}
	if body := readFile(t, filepath.Join(root, ".agents", "skills", "grill-me", "SKILL.md")); body != "canon" {
		t.Fatalf("canon changed %q", body)
	}
}

func TestPushReadsALinkedSettingsFile(t *testing.T) {
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		sent = string(raw)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)
	root := connectRoot(t)
	dotfiles := t.TempDir()
	writeRel(t, dotfiles, "settings.json", `{"theme":"dark"}`)
	writeRel(t, filepath.Join(root, ".cursor"), "rules/own.md", "own")
	if err := os.Symlink(filepath.Join(dotfiles, "settings.json"), filepath.Join(root, ".cursor", "settings.json")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")
	if code, _, stderr := runCLI(t, "push", "cursor"); code != 0 {
		t.Fatalf("push %s", stderr)
	}
	if !strings.Contains(sent, `"settings.json"`) || !strings.Contains(sent, "dark") {
		t.Fatalf("linked settings lost: %s", sent)
	}
}
