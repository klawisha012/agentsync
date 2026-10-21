package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

const canonPluginsBody = `{"enabledPlugins":{"commit@anthropic":true,"lint@acme":true},
"extraKnownMarketplaces":{"acme":{"source":{"source":"github","repo":"acme/plugins"}}}}`

func claudeSettings(t *testing.T, root string) map[string]any {
	t.Helper()
	var top map[string]any
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(root, ".claude", "settings.json"))), &top); err != nil {
		t.Fatal(err)
	}
	return top
}

func TestOnWritesCanonPluginsIntoClaudeSettings(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "plugins.json", canonPluginsBody)
	writeRel(t, root, ".claude/settings.json", `{"theme":"dark","enabledPlugins":{"mine@x":true,"lint@acme":false}}`)

	code, stdout, stderr := runCLI(t, "claude", "on")
	if code != 0 || !strings.Contains(stdout, "плагины") {
		t.Fatalf("code %d stdout %s stderr %s", code, stdout, stderr)
	}
	top := claudeSettings(t, root)
	plugins, _ := top["enabledPlugins"].(map[string]any)
	markets, _ := top["extraKnownMarketplaces"].(map[string]any)
	if top["theme"] != "dark" || plugins["mine@x"] != true || plugins["commit@anthropic"] != true || plugins["lint@acme"] != true || markets["acme"] == nil {
		t.Fatalf("settings %v", top)
	}

	writeRel(t, canon, "plugins.json", `{"enabledPlugins":{"lint@acme":true},"extraKnownMarketplaces":{"acme":{"source":{"source":"github","repo":"acme/plugins"}}}}`)
	if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if plugins, _ := claudeSettings(t, root)["enabledPlugins"].(map[string]any); plugins["commit@anthropic"] != nil || plugins["lint@acme"] != true {
		t.Fatalf("second on %v", plugins)
	}

	if code, _, stderr := runCLI(t, "claude", "off"); code != 0 {
		t.Fatal(stderr)
	}
	top = claudeSettings(t, root)
	plugins, _ = top["enabledPlugins"].(map[string]any)
	markets, _ = top["extraKnownMarketplaces"].(map[string]any)
	if plugins["mine@x"] != true || plugins["lint@acme"] != false || markets["acme"] != nil || top["theme"] != "dark" {
		t.Fatalf("off %v", top)
	}
}

func TestOnSkipsPluginsForCodex(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "plugins.json", canonPluginsBody)
	code, stdout, stderr := runCLI(t, "codex", "on")
	if code != 0 || !strings.Contains(stdout, "Пропущены") || !strings.Contains(stdout, "плагины") {
		t.Fatalf("code %d stdout %s stderr %s", code, stdout, stderr)
	}
}

func TestPipeApplyClaudeCardMovesMarkedPluginsToTheCanon(t *testing.T) {
	root := connectRoot(t)
	code, stderr := pipeApply(t, `{"author":"zwarder","agent":"claude","files":[
		{"path":"settings.json","body":"{\"theme\":\"light\",\"enabledPlugins\":{\"commit@anthropic\":true,\"mine@x\":true}}"},
		{"path":"agentsync.json","body":"{\"mcp\":[],\"plugins\":[\"commit@anthropic\"],\"marketplaces\":[]}"}]}`)
	if code != 0 {
		t.Fatalf("apply %d %s", code, stderr)
	}
	canon := readFile(t, filepath.Join(root, ".agents", "plugins.json"))
	if !strings.Contains(canon, "commit@anthropic") || strings.Contains(canon, "mine@x") {
		t.Fatalf("canon plugins %s", canon)
	}
	top := claudeSettings(t, root)
	plugins, _ := top["enabledPlugins"].(map[string]any)
	if top["theme"] != "light" || plugins["mine@x"] != true || plugins["commit@anthropic"] != true {
		t.Fatalf("home settings %v", top)
	}
}
