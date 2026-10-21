package agent

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// AbsolutePath finds a path bound to one machine in a file body.
var AbsolutePath = regexp.MustCompile(`(?i)(?:^|[\s"'=])(?:[a-z]:[\\/][^\s"']+|/(?:[a-z0-9._-]+/)*[a-z0-9._-]+)`)

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
	if body, rest, ok := portableHooks(file.Path, file.Body); ok {
		if IsHook(file.Path) && AbsolutePath.MatchString(rest) {
			return file, false
		}
		file.Body = body
	} else if IsHook(file.Path) && AbsolutePath.MatchString(file.Body) {
		return file, false
	}
	if IsMCP(file.Path) {
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
		return body, !AbsolutePath.MatchString(body)
	}
	kept := map[string]json.RawMessage{}
	for name, server := range servers {
		if !AbsolutePath.Match(server) {
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
		if AbsolutePath.Match(server) {
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
	top["mcpServers"] = marshalPlain(servers)
	return encodeTop(top)
}

// marshalPlain encodes JSON without HTML escaping, so "&&" in a command stays as written.
func marshalPlain(value any) json.RawMessage {
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(value)
	return bytes.TrimRight(out.Bytes(), "\n")
}

// encodeTop writes a settings object back as indented JSON.
func encodeTop(top map[string]json.RawMessage) string {
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(top)
	return out.String()
}

// writeHomeFile writes a portable file into an AI agent home. An MCP file keeps the
// machine servers already on disk; on a name clash the machine server wins.
func writeHomeFile(home string, file File) error {
	if IsMCP(file.Path) {
		if raw, err := os.ReadFile(filepath.Join(home, filepath.FromSlash(file.Path))); err == nil {
			if machine := machineServers(string(raw)); len(machine) > 0 {
				if top, servers, ok := mcpServers(file.Body); ok {
					maps.Copy(servers, machine)
					file.Body = encodeMCP(top, servers)
				}
			}
		}
	}
	if raw, err := os.ReadFile(filepath.Join(home, filepath.FromSlash(file.Path))); err == nil {
		file.Body = keepMachineHooks(raw, file.Body)
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
	if clean == MarkFile || IsHook(clean) || IsMCP(clean) {
		return true
	}
	base := strings.ToLower(filepath.Base(clean))
	switch base {
	case "config.json", "settings.json", "agents.md", "claude.md", "gemini.md", "plugins.json":
		return true
	}
	for _, seg := range strings.Split(strings.ToLower(clean), "/") {
		switch seg {
		case "rules", "skills", "plugins", "theme", "hooks", "mcp", "config", "agents", "commands", "prompts":
			return true
		}
	}
	return false
}

// IsHook reports a hook file.
func IsHook(path string) bool {
	base := strings.ToLower(filepath.Base(filepath.ToSlash(path)))
	return base == "hooks.json" || base == "hook.json" || pathSegment(path, "hooks")
}

// IsMCP reports an MCP settings file.
func IsMCP(path string) bool {
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
