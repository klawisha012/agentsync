package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const canonMCPBody = `{"mcpServers":{
	"github":{"command":"npx","args":["-y","gh"],"env":{"GITHUB_TOKEN":"${GITHUB_TOKEN}"}},
	"docs":{"type":"http","url":"https://docs.example/mcp","headers":{"Authorization":"Bearer ${DOCS_TOKEN}"},"disabled":true}
}}`

func claudeServers(t *testing.T, root string) (map[string]any, map[string]any) {
	t.Helper()
	var top map[string]any
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(root, ".claude.json"))), &top); err != nil {
		t.Fatal(err)
	}
	servers, _ := top["mcpServers"].(map[string]any)
	return top, servers
}

func TestOnWritesCanonMCPIntoClaudeSettings(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "mcp.json", canonMCPBody)
	writeRel(t, root, ".claude.json", `{"numStartups":3,"mcpServers":{"mine":{"command":"mine"},"docs":{"command":"old-docs"}}}`)

	code, stdout, stderr := runCLI(t, "claude", "on")
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, stderr)
	}
	if !strings.Contains(stdout, "MCP-серверы") {
		t.Fatalf("stdout %s", stdout)
	}
	top, servers := claudeServers(t, root)
	if top["numStartups"] != float64(3) || servers["mine"] == nil {
		t.Fatalf("foreign keys changed: %v", top)
	}
	github, _ := servers["github"].(map[string]any)
	if env, _ := github["env"].(map[string]any); env["GITHUB_TOKEN"] != "${GITHUB_TOKEN}" {
		t.Fatalf("github %v", github)
	}
	if docs, _ := servers["docs"].(map[string]any); docs["url"] != "https://docs.example/mcp" {
		t.Fatalf("docs not replaced %v", docs)
	}
	if strings.Contains(readFile(t, filepath.Join(root, ".agents", "mcp.json")), "old-docs") {
		t.Fatal("canon changed")
	}

	writeRel(t, canon, "mcp.json", `{"mcpServers":{"docs":{"type":"http","url":"https://docs.example/mcp"}}}`)
	if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if _, servers := claudeServers(t, root); servers["github"] != nil || servers["mine"] == nil || servers["docs"] == nil {
		t.Fatalf("second on %v", servers)
	}

	if code, _, stderr := runCLI(t, "claude", "off"); code != 0 {
		t.Fatal(stderr)
	}
	top, servers = claudeServers(t, root)
	if docs, _ := servers["docs"].(map[string]any); docs["command"] != "old-docs" {
		t.Fatalf("foreign record not back %v", servers)
	}
	if servers["mine"] == nil || servers["github"] != nil || top["numStartups"] != float64(3) {
		t.Fatalf("off %v", top)
	}
	if code, _, stderr := runCLI(t, "claude", "off"); code != 0 {
		t.Fatalf("second off %s", stderr)
	}
}

func TestOnTranslatesCanonMCPIntoCodexTOML(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "mcp.json", canonMCPBody)
	writeRel(t, canon, ".env", "# secrets\nexport DOCS_TOKEN=\"fromfile\"\n")
	t.Setenv("GITHUB_TOKEN", "fromenv")
	config := filepath.Join(root, ".codex", "config.toml")
	own := "model = \"o3\"\n\n[mcp_servers.mine]\ncommand = \"mine\"\n\n[mcp_servers.docs]\ncommand = \"old\"\nargs = [\n  \"--a\",\n  [\"nested\"]\n]\n[mcp_servers.docs.env]\nA = \"b\"\n\n[profiles.x]\nmodel = \"y\"\n"
	writeRel(t, root, ".codex/config.toml", own)

	code, stdout, stderr := runCLI(t, "codex", "on")
	if code != 0 || !strings.Contains(stdout, "MCP-серверы") {
		t.Fatalf("code %d stdout %s stderr %s", code, stdout, stderr)
	}
	body := readFile(t, config)
	for _, want := range []string{`model = "o3"`, "[mcp_servers.mine]", "[profiles.x]", "[mcp_servers.github]",
		`command = "npx"`, `args = ["-y", "gh"]`, `GITHUB_TOKEN = "fromenv"`, `url = "https://docs.example/mcp"`,
		`Authorization = "Bearer fromfile"`, "enabled = false"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s in\n%s", want, body)
		}
	}
	if strings.Contains(body, `command = "old"`) || strings.Contains(body, "nested") || strings.Contains(body, "${") {
		t.Fatalf("body\n%s", body)
	}

	if code, _, stderr := runCLI(t, "codex", "off"); code != 0 {
		t.Fatal(stderr)
	}
	body = readFile(t, config)
	for _, want := range []string{`model = "o3"`, "[mcp_servers.mine]", "[profiles.x]", `command = "old"`, "[mcp_servers.docs.env]"} {
		if !strings.Contains(body, want) {
			t.Fatalf("off lost %s in\n%s", want, body)
		}
	}
	if strings.Contains(body, "github") || strings.Contains(body, "fromfile") {
		t.Fatalf("own records stay\n%s", body)
	}

	writeRel(t, canon, "mcp.json", `{"mcpServers":{"x":{"command":"x","env":{"K":"${NOPE_NOT_SET}"}}}}`)
	code, stdout, stderr = runCLI(t, "codex", "on")
	if code != 0 || !strings.Contains(stdout, "NOPE_NOT_SET") {
		t.Fatalf("missing variable %d %s %s", code, stdout, stderr)
	}
	if got := readFile(t, config); !strings.Contains(got, `K = "${NOPE_NOT_SET}"`) {
		t.Fatalf("variable not written as is\n%s", got)
	}
}

func TestOnSkipsMCPForAnAgentWithoutAFormatter(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "mcp.json", canonMCPBody)
	code, stdout, stderr := runCLI(t, "cursor", "on")
	if code != 0 || !strings.Contains(stdout, "Пропущены") || !strings.Contains(stdout, "MCP-серверы") {
		t.Fatalf("code %d stdout %s stderr %s", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, ".cursor", "mcp.json")); !os.IsNotExist(err) {
		t.Fatal("mcp written for cursor")
	}
}

func TestOnGivesGeminiAnHTTPServerUnderHTTPURL(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "mcp.json", canonMCPBody)
	if code, _, stderr := runCLI(t, "gemini", "on"); code != 0 {
		t.Fatal(stderr)
	}
	var top struct {
		Servers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(root, ".gemini", "settings.json"))), &top); err != nil {
		t.Fatal(err)
	}
	if docs := top.Servers["docs"]; docs["httpUrl"] != "https://docs.example/mcp" || docs["url"] != nil || docs["type"] != nil {
		t.Fatalf("docs %v", docs)
	}
}

func TestCanonApplyReconnectsConnectedAgents(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "mcp.json", `{"mcpServers":{"old":{"command":"old"}}}`)
	if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if code, stderr := pipeApply(t, `{"author":"zwarder","agent":"agents","files":[{"path":"mcp.json","body":"{\"mcpServers\":{\"fresh\":{\"command\":\"fresh\"}}}"}]}`); code != 0 {
		t.Fatalf("canon apply %s", stderr)
	}
	if _, servers := claudeServers(t, root); servers["fresh"] == nil || servers["old"] != nil {
		t.Fatalf("connected agent not updated %v", servers)
	}
}

func TestOnKeepsALinkedSettingsFileALink(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "mcp.json", `{"mcpServers":{"docs":{"url":"https://docs.example/mcp"}}}`)
	dotfiles := t.TempDir()
	writeRel(t, dotfiles, "claude.json", `{"numStartups":1}`)
	if err := os.Symlink(filepath.Join(dotfiles, "claude.json"), filepath.Join(root, ".claude.json")); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if info, err := os.Lstat(filepath.Join(root, ".claude.json")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("the link was replaced by a file")
	}
	if body := readFile(t, filepath.Join(dotfiles, "claude.json")); !strings.Contains(body, "docs") || !strings.Contains(body, "numStartups") {
		t.Fatalf("target %s", body)
	}
}

func TestCanonApplyNamesAnUnsetVariableOfAReconnectedAgent(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "mcp.json", `{"mcpServers":{"x":{"command":"x"}}}`)
	if code, _, stderr := runCLI(t, "codex", "on"); code != 0 {
		t.Fatal(stderr)
	}
	var out, errOut bytes.Buffer
	payload := `{"author":"zwarder","agent":"agents","files":[{"path":"mcp.json","body":"{\"mcpServers\":{\"x\":{\"command\":\"x\",\"env\":{\"K\":\"${NOPE_UNSET_2}\"}}}}"}]}`
	if code := Run(nil, strings.NewReader(payload), &out, &errOut); code != 0 || !strings.Contains(out.String(), "NOPE_UNSET_2") {
		t.Fatalf("code %d stdout %s stderr %s", code, out.String(), errOut.String())
	}
}
