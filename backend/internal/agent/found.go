package agent

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Found(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	type candidate struct {
		name   string
		dotted bool
	}
	found := make([]candidate, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()
		if skippedAgentDir(dirName) {
			continue
		}
		home := filepath.Join(root, dirName)
		ok, err := agentMarkers(home)
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				continue
			}
			return nil, err
		}
		if !ok {
			continue
		}
		files, err := readTree(home)
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				continue
			}
			return nil, err
		}
		if len(portableOnly(files)) == 0 {
			continue
		}
		found = append(found, candidate{name: agentDirName(dirName), dotted: strings.HasPrefix(dirName, ".")})
	}
	sort.Slice(found, func(i, j int) bool {
		left := strings.ToLower(found[i].name)
		right := strings.ToLower(found[j].name)
		if left != right {
			return left < right
		}
		if found[i].dotted != found[j].dotted {
			return !found[i].dotted
		}
		return found[i].name < found[j].name
	})
	names := make([]string, 0, len(found))
	seen := map[string]struct{}{}
	for _, item := range found {
		key := strings.ToLower(item.name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		names = append(names, item.name)
	}
	return names, nil
}

func agentDirName(dir string) string {
	if strings.HasPrefix(dir, ".") && len(dir) > 1 {
		return dir[1:]
	}
	return dir
}

func skippedAgentDir(name string) bool {
	switch strings.ToLower(name) {
	case ".agentsync", ".git", ".ssh", ".config", ".agents", "node_modules":
		return true
	default:
		return false
	}
}

func agentMarkers(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if entry.IsDir() {
			switch name {
			case "rules", "skills", "plugins", "theme", "hooks", "mcp", "config":
				return true, nil
			}
			continue
		}
		switch name {
		case "config.json", "settings.json", "agents.md", "claude.md", "hooks.json", "hook.json", "mcp.json":
			return true, nil
		}
		if strings.HasSuffix(name, ".mcp.json") {
			return true, nil
		}
	}
	return false, nil
}
