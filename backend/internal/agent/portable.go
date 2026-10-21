package agent

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var absolutePath = regexp.MustCompile(`(?i)(?:^|[\s"'=])(?:[a-z]:[\\/][^\s"']+|/(?:[a-z0-9._-]+/)*[a-z0-9._-]+)`)

func portableOnly(files []File) []File {
	kept := make([]File, 0, len(files))
	for _, file := range files {
		if file, ok := portableFile(file); ok {
			kept = append(kept, file)
		}
	}
	return kept
}

// portableFile returns the part of a file that travels; an MCP file loses its machine servers.
func portableFile(file File) (File, bool) {
	if PrivatePath(file.Path) || !PortablePath(file.Path) {
		return file, false
	}
	if isHook(file.Path) && absolutePath.MatchString(file.Body) {
		return file, false
	}
	if isMCP(file.Path) {
		body, ok := PortableMCP(file.Body)
		file.Body = body
		return file, ok
	}
	return file, true
}

// PortableMCP drops the servers bound to this machine by an absolute path. A file without
// an mcpServers object is judged whole. It returns false when nothing portable is left.
func PortableMCP(body string) (string, bool) {
	top, servers, ok := mcpServers(body)
	if !ok {
		return body, !absolutePath.MatchString(body)
	}
	kept := map[string]json.RawMessage{}
	for name, server := range servers {
		if !absolutePath.Match(server) {
			kept[name] = server
		}
	}
	if len(kept) == len(servers) {
		return body, true
	}
	if len(kept) == 0 {
		return "", false
	}
	return encodeMCP(top, kept), true
}

// machineServers lists the servers of an MCP file that never leave this machine.
func machineServers(body string) map[string]json.RawMessage {
	_, servers, ok := mcpServers(body)
	if !ok {
		return nil
	}
	machine := map[string]json.RawMessage{}
	for name, server := range servers {
		if absolutePath.Match(server) {
			machine[name] = server
		}
	}
	return machine
}

func mcpServers(body string) (map[string]json.RawMessage, map[string]json.RawMessage, bool) {
	var top map[string]json.RawMessage
	if json.Unmarshal([]byte(body), &top) != nil {
		return nil, nil, false
	}
	var servers map[string]json.RawMessage
	if json.Unmarshal(top["mcpServers"], &servers) != nil || servers == nil {
		return nil, nil, false
	}
	return top, servers, true
}

func encodeMCP(top, servers map[string]json.RawMessage) string {
	raw, _ := json.Marshal(servers)
	top["mcpServers"] = raw
	out, _ := json.MarshalIndent(top, "", "  ")
	return string(out) + "\n"
}

// writeHomeFile writes a portable file into an AI agent home. An MCP file keeps the
// machine servers already on disk; on a name clash the machine server wins.
func writeHomeFile(home string, file File) error {
	if isMCP(file.Path) {
		if raw, err := os.ReadFile(filepath.Join(home, filepath.FromSlash(file.Path))); err == nil {
			if machine := machineServers(string(raw)); len(machine) > 0 {
				if top, servers, ok := mcpServers(file.Body); ok {
					maps.Copy(servers, machine)
					file.Body = encodeMCP(top, servers)
				}
			}
		}
	}
	return writeFile(home, file)
}

// PrivatePath reports machine-only files that never leave the computer.
func PrivatePath(path string) bool {
	clean := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(clean)
	switch base {
	case "credentials.json", "credentials", "auth.json", "session.json", ".env":
		return true
	}
	if strings.HasPrefix(base, ".env") {
		return true
	}
	if strings.HasSuffix(base, ".lock") || strings.HasSuffix(base, ".log") {
		return true
	}
	segs := strings.Split(clean, "/")
	for _, seg := range segs {
		switch seg {
		case "credentials", "sessions", "session", "cache", "caches", "logs", "log", "memory", "history", "conversations", "node_modules", "vendor", "dist", "locks", "bundled", "marketplace-cache":
			return true
		}
	}
	// Plugin clones and catalogs are re-downloaded by the agent; only the settings entries travel.
	if slices.Contains(segs, "plugins") {
		for _, seg := range segs {
			if strings.Contains(seg, "marketplace") || strings.Contains(seg, "cache") {
				return true
			}
		}
	}
	return false
}

// PortablePath reports files that make up the portable settings of an AI agent.
func PortablePath(path string) bool {
	clean, ok := CleanRel(path)
	if !ok {
		return false
	}
	if isHook(clean) || isMCP(clean) {
		return true
	}
	base := strings.ToLower(filepath.Base(clean))
	switch base {
	case "config.json", "settings.json", "agents.md", "claude.md":
		return true
	}
	for _, seg := range strings.Split(strings.ToLower(clean), "/") {
		switch seg {
		case "rules", "skills", "plugins", "theme", "hooks", "mcp", "config", "agents", "commands":
			return true
		}
	}
	return false
}

func isHook(path string) bool {
	base := strings.ToLower(filepath.Base(filepath.ToSlash(path)))
	return base == "hooks.json" || base == "hook.json" || pathSegment(path, "hooks")
}

func isMCP(path string) bool {
	base := strings.ToLower(filepath.Base(filepath.ToSlash(path)))
	return base == "mcp.json" || strings.HasSuffix(base, ".mcp.json") || pathSegment(path, "mcp")
}

func pathSegment(path, want string) bool {
	for _, seg := range strings.Split(strings.ToLower(filepath.ToSlash(path)), "/") {
		if seg == want {
			return true
		}
	}
	return false
}
