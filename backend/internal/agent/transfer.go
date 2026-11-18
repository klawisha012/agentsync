package agent

import (
	"encoding/json"
	"errors"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// TransferReport names what an export moved into agents and what it left behind.
type TransferReport struct {
	Moved []string
	Kept  []string
}

// Lines are the report as the CLI prints it.
func (r TransferReport) Lines() []string {
	var out []string
	if len(r.Moved) > 0 {
		out = append(out, "В agents перенесены: "+strings.Join(r.Moved, ", ")+".")
	}
	if len(r.Kept) > 0 {
		out = append(out, "Не перенесены в agents: "+strings.Join(r.Kept, ", ")+".")
	}
	return out
}

// ownMove is one own record of an AI agent on its way into agents.
type ownMove struct {
	kind   recordKind
	file   string // the agent's settings file
	native string // its record name there
	canon  string // its record name in agents
	body   string
}

// ownHooks lists the agent's own hook groups in the agents format. Machine hooks stay.
func ownHooks(root string, receiver Receiver, dir string) ([]ownMove, error) {
	if receiver.Hooks == "" {
		return nil, nil
	}
	file := filepath.Join(filepath.Dir(dir), filepath.FromSlash(receiver.Hooks))
	settings, err := readHookSettings(file)
	if err != nil {
		return nil, err
	}
	// agents must be readable before anything moves into it.
	if _, err := readCanonHooks(filepath.Join(canonDir(root), "hooks.json")); err != nil {
		return nil, err
	}
	_, ledger, err := loadLedger(hookRecords, receiver.Slug)
	if err != nil {
		return nil, err
	}
	machine := settings.machineHooks()
	var moves []ownMove
	records := settings.records()
	for _, name := range slices.Sorted(maps.Keys(records)) {
		group := records[name]
		if slices.Contains(ledger.Own, name) || machine[name] != nil {
			continue
		}
		if msg := SecretExplanation(file, string(group)); msg != "" {
			return nil, errors.New(msg)
		}
		event, _, _ := strings.Cut(name, " ")
		canonName, body := fromNative(dialects[receiver.Slug], event, group)
		moves = append(moves, ownMove{kind: hookRecords, file: file, native: name, canon: canonName, body: body})
	}
	return moves, nil
}

// fromGemini gives a Gemini HTTP server, httpUrl, back its agents form.
func fromGemini(server json.RawMessage) json.RawMessage {
	var fields map[string]json.RawMessage
	if json.Unmarshal(server, &fields) != nil || fields["httpUrl"] == nil {
		return server
	}
	fields["url"], fields["type"] = fields["httpUrl"], marshalPlain("http")
	delete(fields, "httpUrl")
	return marshalPlain(fields)
}

// names lists the records of the TOML table.
func (s *tomlSettings) names() []string {
	var names []string
	for _, line := range s.lines {
		match := tomlHeader.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if keys := tomlKeys(match[1]); len(keys) >= 2 && keys[0] == s.table && !slices.Contains(names, keys[1]) {
			names = append(names, keys[1])
		}
	}
	return names
}

// parseTOMLServer reads one [mcp_servers.<name>] block of Codex or Grok into the agents form.
// ok is false for anything beyond the MCP fields agents holds.
func parseTOMLServer(block string) (json.RawMessage, bool) {
	server := map[string]any{}
	section := ""
	for _, line := range splitLines(block) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if match := tomlHeader.FindStringSubmatch(line); match != nil {
			keys := tomlKeys(match[1])
			section = ""
			if len(keys) == 3 {
				section = keys[2]
			}
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return nil, false
		}
		key = strings.Trim(strings.TrimSpace(key), `"`)
		value, ok := tomlValue(strings.TrimSpace(raw))
		if !ok {
			return nil, false
		}
		if section != "" {
			table, _ := server[section].(map[string]any)
			if table == nil {
				table = map[string]any{}
			}
			table[key] = value
			server[section] = table
			continue
		}
		server[key] = value
	}
	out := map[string]any{}
	for key, value := range server {
		switch key {
		case "command", "args", "env", "url":
			out[key] = value
		case "http_headers", "headers":
			out["headers"] = value
		case "enabled":
			if value == false {
				out["disabled"] = true
			}
		default:
			return nil, false
		}
	}
	if out["url"] != nil && out["command"] == nil {
		out["type"] = "http"
	}
	return marshalPlain(out), true
}

// tomlValue reads a one-line TOML value: a string, a boolean, an array of strings or an inline
// table of strings.
func tomlValue(raw string) (any, bool) {
	switch {
	case raw == "true" || raw == "false":
		return raw == "true", true
	case strings.HasPrefix(raw, `"`):
		text, err := strconv.Unquote(raw)
		return text, err == nil
	case strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'") && len(raw) >= 2:
		return raw[1 : len(raw)-1], true
	case strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]"):
		var items []any
		for _, part := range splitTOMLList(raw[1 : len(raw)-1]) {
			value, ok := tomlValue(part)
			if !ok {
				return nil, false
			}
			items = append(items, value)
		}
		return items, true
	case strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}"):
		table := map[string]any{}
		for _, part := range splitTOMLList(raw[1 : len(raw)-1]) {
			key, value, ok := strings.Cut(part, "=")
			if !ok {
				return nil, false
			}
			parsed, ok := tomlValue(strings.TrimSpace(value))
			if !ok {
				return nil, false
			}
			table[strings.Trim(strings.TrimSpace(key), `"`)] = parsed
		}
		return table, true
	}
	return nil, false
}

// splitTOMLList splits the inside of an array or an inline table at the commas outside strings.
func splitTOMLList(inner string) []string {
	var parts []string
	var current strings.Builder
	quote := byte(0)
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		switch {
		case quote != 0 && c == '\\' && quote == '"' && i+1 < len(inner):
			current.WriteByte(c)
			i++
			c = inner[i]
		case quote != 0 && c == quote:
			quote = 0
		case quote == 0 && (c == '"' || c == '\''):
			quote = c
		case quote == 0 && c == ',':
			if part := strings.TrimSpace(current.String()); part != "" {
				parts = append(parts, part)
			}
			current.Reset()
			continue
		}
		current.WriteByte(c)
	}
	if part := strings.TrimSpace(current.String()); part != "" {
		parts = append(parts, part)
	}
	return parts
}
