package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// TransferReport names what a snapshot of an AI agent moved into agents and what it left with the agent.
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

// transferOwn is the transfer into agents (ADR 0039), the first step of a snapshot of an AI agent:
// its own hooks, MCP servers, subagents and commands go into agents in the agents format, and the agent then sees them
// as records of its connection. A name agents already has stays with the agent. A break puts
// agents and the agent back. The caller holds the root lock.
func transferOwn(root, account, agentName string) (TransferReport, error) {
	var report TransferReport
	receiver, found := FindReceiver(agentName)
	if !found || isCanonCard(agentName) {
		return report, nil
	}
	dir, have := receiver.GlobalDir(root)
	if !have || insideCanon(root, dir) {
		return report, nil
	}
	hooks, err := ownHooks(root, receiver, dir)
	if err != nil {
		return report, err
	}
	servers, err := ownServers(root, receiver, &report)
	if err != nil {
		return report, err
	}
	docs, err := ownDocs(root, receiver, dir, &report)
	if err != nil {
		return report, err
	}
	moves := append(hooks, servers...)
	if len(moves)+len(docs) == 0 {
		return report, nil
	}
	snapshot, err := snapshotTree(root, account, canonCard, canonDir(root))
	if err != nil {
		return report, err
	}
	var undo applyUndo
	err = keepRecords(&undo, root, receiver.Slug)
	if err == nil {
		err = writeMoves(root, moves, &undo)
	}
	for _, move := range docs {
		if err == nil {
			err = writeDocMove(root, move, &undo)
		}
	}
	if err == nil {
		_, err = connect(root, account, receiver.Slug, "", "")
	}
	if err != nil {
		undoErr := undo.rollback()
		if undoErr == nil {
			undoErr = removeAll([]string{snapshot})
		}
		return report, errors.Join(err, undoErr)
	}
	if count := len(hooks); count > 0 {
		report.Moved = append(report.Moved, fmt.Sprintf("хуки (%d)", count))
	}
	if len(servers) > 0 {
		names := make([]string, len(servers))
		for i, move := range servers {
			names[i] = move.canon
		}
		slices.Sort(names)
		report.Moved = append(report.Moved, "MCP-серверы: "+strings.Join(names, ", "))
	}
	for _, kind := range []docKind{subagentDoc, commandDoc} {
		var names []string
		for _, move := range docs {
			if move.kind.key == kind.key {
				names = append(names, strings.TrimSuffix(strings.TrimPrefix(move.rel, kind.key+"/"), ".md"))
			}
		}
		if len(names) > 0 {
			report.Moved = append(report.Moved, kind.name+": "+strings.Join(names, ", "))
		}
	}
	return report, nil
}

// writeDocMove puts the file into agents and takes the agent's own file away; connect then gives it back.
func writeDocMove(root string, move docMove, undo *applyUndo) error {
	target := filepath.Join(canonDir(root), filepath.FromSlash(move.rel))
	for _, file := range []string{target, move.file} {
		if err := undo.keep(file); err != nil {
			return err
		}
	}
	if err := writeFile(canonDir(root), File{Path: move.rel, Body: move.body}); err != nil {
		return err
	}
	return os.Remove(move.file)
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

// ownServers lists the agent's own MCP servers in the agents format. A machine server stays; a
// server whose name agents has, or that the agents format cannot hold, stays and goes into the report.
func ownServers(root string, receiver Receiver, report *TransferReport) ([]ownMove, error) {
	if receiver.MCP == "" {
		return nil, nil
	}
	file := receiver.mcpFile(root)
	settings, err := readSettings(file, mcpRecords.key)
	if err != nil {
		return nil, err
	}
	canon, err := readSettings(filepath.Join(canonDir(root), "mcp.json"), mcpRecords.key)
	if err != nil {
		return nil, err
	}
	_, ledger, err := loadLedger(mcpRecords, receiver.Slug)
	if err != nil {
		return nil, err
	}
	servers := map[string]json.RawMessage{}
	switch own := settings.(type) {
	case *jsonSettings:
		for name, server := range own.servers {
			servers[name] = fromGemini(server)
		}
	case *tomlSettings:
		for _, name := range own.names() {
			block, _ := own.get(name)
			server, ok := parseTOMLServer(block)
			if !ok {
				if !slices.Contains(ledger.Own, name) {
					report.Kept = append(report.Kept, "MCP-сервер "+name)
				}
				continue
			}
			servers[name] = server
		}
	}
	var moves []ownMove
	for _, name := range slices.Sorted(maps.Keys(servers)) {
		server := servers[name]
		if slices.Contains(ledger.Own, name) || AbsolutePath.Match(server) {
			continue
		}
		if msg := SecretExplanation(file, string(server)); msg != "" {
			return nil, errors.New(msg)
		}
		if _, taken := canon.get(name); taken {
			report.Kept = append(report.Kept, "MCP-сервер "+name+" (имя занято в agents)")
			continue
		}
		moves = append(moves, ownMove{kind: mcpRecords, file: file, native: name, canon: name, body: string(server)})
	}
	return moves, nil
}

// writeMoves puts the records into agents and takes them out of the agent's files.
func writeMoves(root string, moves []ownMove, undo *applyUndo) error {
	canon := canonDir(root)
	files := map[string]mcpSettings{}
	open := func(kind recordKind, file string, inCanon bool) (mcpSettings, error) {
		if settings, ok := files[file]; ok {
			return settings, nil
		}
		if err := undo.keep(file); err != nil {
			return nil, err
		}
		var settings mcpSettings
		var err error
		if inCanon && kind == hookRecords {
			settings, err = readCanonHooks(file)
		} else {
			settings, err = openSettings(kind, file)
		}
		files[file] = settings
		return settings, err
	}
	for _, move := range moves {
		target, err := open(move.kind, filepath.Join(canon, canonRecordFile[move.kind]), true)
		if err != nil {
			return err
		}
		target.set(move.canon, move.body)
		source, err := open(move.kind, move.file, false)
		if err != nil {
			return err
		}
		source.remove(move.native)
	}
	for file, settings := range files {
		if err := writeSettings(file, settings); err != nil {
			return err
		}
	}
	return nil
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
