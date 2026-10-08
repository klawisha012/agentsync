package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFoundReadsAgentHomes(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		target := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".grok/skills/ru-text/SKILL.md", "skill")
	write("Claude/AGENTS.md", "rules")
	write("Notes/readme.txt", "not an agent")
	write("proj/src/skills/hidden.md", "nested")
	write(".agentsync/skills/local.md", "state")
	write(".codex/skills/one.md", "dot")
	write("Codex/skills/two.md", "plain")
	if err := os.MkdirAll(filepath.Join(root, "empty-skills", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "blank"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Found(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Claude", "Codex", "grok"}
	if len(got) != len(want) {
		t.Fatalf("found %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("found %v", got)
		}
	}
}
