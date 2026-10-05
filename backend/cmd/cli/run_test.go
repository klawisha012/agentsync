package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klawisha012/agentsync/internal/agent"
)

func TestPipeApplySnapshotsGrokHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AGENTSYNC_ROOT", dir)
	t.Setenv("AGENTSYNC_ACCOUNT", "zwarder")
	t.Setenv("AGENTSYNC_STATE", t.TempDir())
	home := filepath.Join(dir, ".grok")
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
	payload := bytes.NewBufferString(`{"author":"zwarder","agent":"Grok","files":[{"path":"AGENTS.md","body":"new"}]}`)
	var out, err bytes.Buffer
	if code := run(nil, payload, &out, &err); code != 0 {
		t.Fatalf("apply %d %s", code, err.String())
	}
	if body := readFile(t, filepath.Join(home, "AGENTS.md")); body != "new" {
		t.Fatalf("home %q", body)
	}
	if body := readFile(t, filepath.Join(home, "credentials.json")); body != "secret" {
		t.Fatal("credentials changed")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "Grok")); !os.IsNotExist(statErr) {
		t.Fatal("apply wrote ~/Grok instead of ~/.grok")
	}
	snaps := agent.Chain(dir, "zwarder", "Grok")
	if len(snaps) != 1 || !hasBody(snaps[0].Files, "AGENTS.md", "old") {
		t.Fatalf("snapshot %+v", snaps)
	}
	if revertErr := agent.Revert(dir, "zwarder", "Grok"); revertErr != nil {
		t.Fatal(revertErr)
	}
	if body := readFile(t, filepath.Join(home, "AGENTS.md")); body != "old" {
		t.Fatalf("revert %q", body)
	}
	if body := readFile(t, filepath.Join(home, "credentials.json")); body != "secret" {
		t.Fatal("revert touched credentials")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func hasBody(files []agent.File, path, body string) bool {
	for _, file := range files {
		if file.Path == path && file.Body == body {
			return true
		}
	}
	return false
}

func TestRunHelp(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		code   int
		stdout string
		stderr string
	}{
		{name: "no args", args: nil, code: 0, stdout: "Команды:", stderr: ""},
		{name: "help flag", args: []string{"--help"}, code: 0, stdout: "agentsync help apply", stderr: ""},
		{name: "short help flag", args: []string{"-h"}, code: 0, stdout: "AGENTSYNC_SERVER", stderr: ""},
		{name: "help command", args: []string{"help"}, code: 0, stdout: "agentsync push", stderr: ""},
		{name: "command help", args: []string{"help", "apply"}, code: 0, stdout: "agentsync apply <автор> <ии-агент>", stderr: ""},
		{name: "skills help", args: []string{"help", "skills"}, code: 0, stdout: "agentsync skills <автор> <ии-агент>", stderr: ""},
		{name: "flag after command", args: []string{"push", "--help"}, code: 0, stdout: "agentsync push <ии-агент>", stderr: ""},
		{name: "unknown command", args: []string{"nope"}, code: 2, stdout: "", stderr: "Неизвестная команда «nope»."},
		{name: "unknown topic", args: []string{"help", "nope"}, code: 2, stdout: "", stderr: "agentsync help"},
		{name: "missing agent", args: []string{"push"}, code: 2, stdout: "", stderr: "Назовите ИИ-агента."},
		{name: "missing apply args", args: []string{"apply", "Author"}, code: 2, stdout: "", stderr: "agentsync apply"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			code := run(tt.args, nil, &out, &err)
			if code != tt.code {
				t.Fatalf("code %d", code)
			}
			stdoutOK := strings.Contains(out.String(), tt.stdout)
			if tt.stdout == "" {
				stdoutOK = out.Len() == 0
			}
			stderrOK := strings.Contains(err.String(), tt.stderr)
			if tt.stderr == "" {
				stderrOK = err.Len() == 0
			}
			if !stdoutOK || !stderrOK {
				t.Fatalf("stdout %q stderr %q", out.String(), err.String())
			}
		})
	}
}
