package agent

import (
	"path/filepath"
	"regexp"
	"strings"
)

var absolutePath = regexp.MustCompile(`(?i)(?:^|[\s"'=])(?:[a-z]:[\\/][^\s"']+|/(?:[a-z0-9._-]+/)*[a-z0-9._-]+)`)

func portableOnly(files []File) []File {
	kept := make([]File, 0, len(files))
	for _, file := range files {
		if isPortable(file.Path, file.Body) {
			kept = append(kept, file)
		}
	}
	return kept
}

func isPortable(path, body string) bool {
	if privatePath(path) {
		return false
	}
	if isHook(path) && absolutePath.MatchString(body) {
		return false
	}
	if isMCP(path) && (absolutePath.MatchString(body) || strings.Contains(strings.ToLower(body), "authorization")) {
		return false
	}
	return portablePath(path)
}

func privatePath(path string) bool {
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
	for _, seg := range strings.Split(clean, "/") {
		switch seg {
		case "credentials", "sessions", "session", "cache", "caches", "logs", "log", "memory", "history", "conversations", "node_modules", "vendor", "dist", "locks", "bundled", "marketplace-cache":
			return true
		}
	}
	return false
}

func portablePath(path string) bool {
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
		case "rules", "skills", "plugins", "theme", "hooks", "mcp", "config":
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
