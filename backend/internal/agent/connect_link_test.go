package agent

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestConnectExplainsAForbiddenLink(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AGENTSYNC_STATE", t.TempDir())
	for rel, body := range map[string]string{".agents/skills/note/SKILL.md": "note", ".cursor/skills/mine/SKILL.md": "mine"} {
		target := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	symlink = func(string, string) error {
		return &os.LinkError{Op: "symlink", Err: syscall.Errno(1314)}
	}
	t.Cleanup(func() { symlink = os.Symlink })

	_, err := Connect(root, "acc", "cursor")
	if err == nil || !strings.Contains(err.Error(), "режим разработчика") {
		t.Fatalf("err %v", err)
	}
	entries, _ := os.ReadDir(filepath.Join(root, ".cursor", "skills"))
	if len(entries) != 1 || entries[0].Name() != "mine" {
		t.Fatalf("home changed %v", entries)
	}
}
