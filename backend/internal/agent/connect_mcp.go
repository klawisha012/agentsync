package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var (
	secretVariable = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
	tomlHeader     = regexp.MustCompile(`^\s*\[\[?\s*(.+?)\s*\]\]?\s*(?:#.*)?$`)
	bareTOMLKey    = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

// mcpLedger is the machine's record of the MCP servers on wrote into one AI agent settings file.
// Replaced holds a foreign record on overwrote, so off can put it back.
type mcpLedger struct {
	File     string            `json:"file"`
	Own      []string          `json:"own"`
	Replaced map[string]string `json:"replaced,omitempty"`
}

// mcpSettings is an AI agent settings file seen as named MCP records in its own format.
type mcpSettings interface {
	get(name string) (string, bool)
	set(name, record string)
	remove(name string)
	encode() string
}

// mcpFile is the settings file with the AI agent's user MCP servers. The agent's home
// variable moves it, as CLAUDE_CONFIG_DIR moves ~/.claude.json.
func (r Receiver) mcpFile(root string) string {
	if r.Env != "" {
		if value := strings.TrimSpace(os.Getenv(r.Env)); value != "" {
			return filepath.Join(value, path.Base(r.MCP))
		}
	}
	return filepath.Join(root, filepath.FromSlash(r.MCP))
}

// canonMCP reads the servers of ~/.agents/mcp.json. ok is false when the canon has no MCP file.
func canonMCP(root string) (map[string]json.RawMessage, bool, error) {
	raw, err := os.ReadFile(filepath.Join(canonDir(root), "mcp.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	_, servers, ok := mcpServers(string(raw))
	if !ok {
		return nil, false, errors.New("Файл ~/.agents/mcp.json не разобран: нужен объект mcpServers.")
	}
	if msg := SecretExplanation("mcp.json", string(raw)); msg != "" {
		return nil, false, errors.New(msg)
	}
	return servers, true, nil
}

// connectMCP writes the canon servers into the agent settings file and removes its own
// records the canon no longer has. Foreign records with other names and other keys stay.
func connectMCP(root string, receiver Receiver, servers map[string]json.RawMessage) error {
	ledgerPath, ledger, err := loadLedger(receiver.Slug)
	if err != nil {
		return err
	}
	if len(servers) == 0 && len(ledger.Own) == 0 {
		return nil
	}
	file := receiver.mcpFile(root)
	if ledger.File != "" && !samePlace(ledger.File, file) {
		if err := releaseMCP(ledger); err != nil {
			return err
		}
		ledger = mcpLedger{}
	}
	settings, err := readSettings(file)
	if err != nil {
		return err
	}
	records := map[string]string{}
	for name, server := range servers {
		record, err := renderRecord(root, receiver.Slug, file, name, server)
		if err != nil {
			return err
		}
		records[name] = record
	}
	if ledger.Replaced == nil {
		ledger.Replaced = map[string]string{}
	}
	for _, name := range ledger.Own {
		if _, keep := records[name]; !keep {
			restore(settings, ledger, name)
		}
	}
	names := slices.Sorted(maps.Keys(records))
	for _, name := range names {
		if old, ok := settings.get(name); ok && !slices.Contains(ledger.Own, name) {
			ledger.Replaced[name] = old
		}
		settings.set(name, records[name])
	}
	ledger.File, ledger.Own = file, names
	// The ledger goes first: a record on wrote must never look foreign to the next on.
	if err := saveLedger(ledgerPath, ledger); err != nil {
		return err
	}
	return writeSettings(file, settings)
}

// disconnectMCP removes only the records on wrote and puts replaced foreign records back.
func disconnectMCP(slug string) error {
	ledgerPath, ledger, err := loadLedger(slug)
	if err != nil || ledger.File == "" {
		return err
	}
	if err := releaseMCP(ledger); err != nil {
		return err
	}
	return os.Remove(ledgerPath)
}

func releaseMCP(ledger mcpLedger) error {
	settings, err := readSettings(ledger.File)
	if err != nil {
		return err
	}
	for _, name := range ledger.Own {
		restore(settings, ledger, name)
	}
	return writeSettings(ledger.File, settings)
}

func restore(settings mcpSettings, ledger mcpLedger, name string) {
	settings.remove(name)
	if old, ok := ledger.Replaced[name]; ok {
		settings.set(name, old)
		delete(ledger.Replaced, name)
	}
}

func loadLedger(slug string) (string, mcpLedger, error) {
	dir, err := stateDir()
	if err != nil {
		return "", mcpLedger{}, err
	}
	file := filepath.Join(dir, "mcp", slug+".json")
	var ledger mcpLedger
	raw, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return file, ledger, nil
	}
	if err != nil {
		return file, ledger, err
	}
	if err := json.Unmarshal(raw, &ledger); err != nil {
		return file, ledger, fmt.Errorf("Учёт записей MCP %s не разобран.", file)
	}
	return file, ledger, nil
}

func saveLedger(file string, ledger mcpLedger) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(ledger, "", "  ")
	return os.WriteFile(file, raw, 0o600)
}

func isTOML(file string) bool {
	return strings.EqualFold(filepath.Ext(file), ".toml")
}

func readSettings(file string) (mcpSettings, error) {
	raw, err := os.ReadFile(file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if isTOML(file) {
		return &tomlSettings{lines: splitLines(string(raw))}, nil
	}
	settings := &jsonSettings{top: map[string]json.RawMessage{}, servers: map[string]json.RawMessage{}}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return settings, nil
	}
	if json.Unmarshal(raw, &settings.top) != nil || settings.top == nil {
		return nil, fmt.Errorf("Файл %s не разобран как JSON, MCP-серверы не вписаны.", file)
	}
	if found := settings.top["mcpServers"]; found != nil && json.Unmarshal(found, &settings.servers) != nil {
		return nil, fmt.Errorf("Ключ mcpServers в файле %s не разобран.", file)
	}
	if settings.servers == nil {
		settings.servers = map[string]json.RawMessage{}
	}
	return settings, nil
}

func writeSettings(file string, settings mcpSettings) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, []byte(settings.encode()), 0o600)
}

// renderRecord turns a canon server into the record of the settings format. JSON agents
// expand ${NAME} themselves; Codex does not, so the value comes from the environment or ~/.agents/.env.
func renderRecord(root, slug, file, name string, server json.RawMessage) (string, error) {
	if !isTOML(file) {
		return geminiRecord(slug, server), nil
	}
	var canon struct {
		Command  string            `json:"command"`
		Args     []string          `json:"args"`
		Env      map[string]string `json:"env"`
		URL      string            `json:"url"`
		Headers  map[string]string `json:"headers"`
		Disabled bool              `json:"disabled"`
	}
	if json.Unmarshal(server, &canon) != nil {
		return "", fmt.Errorf("Сервер %s в ~/.agents/mcp.json не разобран.", name)
	}
	lookup := secretLookup(root)
	var missing []string
	expand := func(value string) string {
		return secretVariable.ReplaceAllStringFunc(value, func(match string) string {
			key := secretVariable.FindStringSubmatch(match)[1]
			found, ok := lookup(key)
			if !ok {
				missing = append(missing, key)
			}
			return found
		})
	}
	lines := []string{"[mcp_servers." + tomlKey(name) + "]"}
	if canon.Command != "" {
		lines = append(lines, "command = "+tomlString(expand(canon.Command)))
	}
	if len(canon.Args) > 0 {
		args := make([]string, len(canon.Args))
		for i, arg := range canon.Args {
			args[i] = tomlString(expand(arg))
		}
		lines = append(lines, "args = ["+strings.Join(args, ", ")+"]")
	}
	if len(canon.Env) > 0 {
		lines = append(lines, "env = "+tomlTable(canon.Env, expand))
	}
	if canon.URL != "" {
		lines = append(lines, "url = "+tomlString(expand(canon.URL)))
	}
	if len(canon.Headers) > 0 {
		lines = append(lines, "http_headers = "+tomlTable(canon.Headers, expand))
	}
	if canon.Disabled {
		lines = append(lines, "enabled = false")
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("Переменная секрета %s не задана: задайте её в окружении или в ~/.agents/.env.", missing[0])
	}
	return strings.Join(lines, "\n") + "\n", nil
}

// geminiRecord gives Gemini CLI a streamable HTTP server under httpUrl: its url means SSE.
func geminiRecord(slug string, server json.RawMessage) string {
	var fields map[string]json.RawMessage
	if slug != "gemini-cli" || json.Unmarshal(server, &fields) != nil || string(fields["type"]) != `"http"` || fields["url"] == nil {
		return string(server)
	}
	fields["httpUrl"] = fields["url"]
	delete(fields, "url")
	delete(fields, "type")
	raw, _ := json.Marshal(fields)
	return string(raw)
}

// secretLookup finds a secret variable in the environment, then in ~/.agents/.env.
func secretLookup(root string) func(string) (string, bool) {
	vars := map[string]string{}
	if raw, err := os.ReadFile(filepath.Join(canonDir(root), ".env")); err == nil {
		for _, line := range splitLines(string(raw)) {
			line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "export "))
			name, value, ok := strings.Cut(line, "=")
			if !ok || strings.HasPrefix(line, "#") {
				continue
			}
			vars[strings.TrimSpace(name)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return func(name string) (string, bool) {
		if value, ok := os.LookupEnv(name); ok {
			return value, true
		}
		value, ok := vars[name]
		return value, ok
	}
}

type jsonSettings struct {
	top     map[string]json.RawMessage
	servers map[string]json.RawMessage
}

func (s *jsonSettings) get(name string) (string, bool) {
	record, ok := s.servers[name]
	return string(record), ok
}

func (s *jsonSettings) set(name, record string) {
	s.servers[name] = json.RawMessage(record)
}

func (s *jsonSettings) remove(name string) {
	delete(s.servers, name)
}

func (s *jsonSettings) encode() string {
	return encodeMCP(s.top, s.servers)
}

// tomlSettings edits [mcp_servers.<name>] tables as text, so comments and other tables stay.
// shortcut: a server written as dotted keys or an inline table under [mcp_servers] is not seen; parse TOML fully if users write that form.
type tomlSettings struct {
	lines []string
}

// blocks returns the line ranges of the tables that belong to a server: its header up to the next header.
// A line inside a multi-line array or string is never a header.
func (s *tomlSettings) blocks(name string) [][2]int {
	var found [][2]int
	start := -1
	depth, multi := 0, ""
	for i, line := range s.lines {
		inside := depth > 0 || multi != ""
		depth, multi = tomlNesting(line, depth, multi)
		match := tomlHeader.FindStringSubmatch(line)
		if inside || match == nil {
			continue
		}
		depth = 0
		if start >= 0 {
			found = append(found, [2]int{start, i})
			start = -1
		}
		if keys := tomlKeys(match[1]); len(keys) >= 2 && keys[0] == "mcp_servers" && keys[1] == name {
			start = i
		}
	}
	if start >= 0 {
		found = append(found, [2]int{start, len(s.lines)})
	}
	return found
}

func (s *tomlSettings) get(name string) (string, bool) {
	var parts []string
	for _, block := range s.blocks(name) {
		parts = append(parts, s.lines[block[0]:block[1]]...)
	}
	if len(parts) == 0 {
		return "", false
	}
	return strings.TrimRight(strings.Join(parts, "\n"), "\r\n") + "\n", true
}

func (s *tomlSettings) set(name, record string) {
	s.remove(name)
	for len(s.lines) > 0 && strings.TrimSpace(s.lines[len(s.lines)-1]) == "" {
		s.lines = s.lines[:len(s.lines)-1]
	}
	if len(s.lines) > 0 {
		s.lines = append(s.lines, "")
	}
	s.lines = append(s.lines, splitLines(strings.TrimRight(record, "\r\n"))...)
}

func (s *tomlSettings) remove(name string) {
	blocks := s.blocks(name)
	for i := len(blocks) - 1; i >= 0; i-- {
		s.lines = slices.Delete(s.lines, blocks[i][0], blocks[i][1])
	}
}

func (s *tomlSettings) encode() string {
	return strings.TrimRight(strings.Join(s.lines, "\n"), "\r\n") + "\n"
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// tomlNesting carries the open-bracket depth and an open multi-line string past one line.
func tomlNesting(line string, depth int, multi string) (int, string) {
	for i := 0; i < len(line); i++ {
		if multi != "" {
			if strings.HasPrefix(line[i:], multi) {
				i += len(multi) - 1
				multi = ""
			}
			continue
		}
		switch c := line[i]; {
		case strings.HasPrefix(line[i:], `"""`) || strings.HasPrefix(line[i:], "'''"):
			multi = line[i : i+3]
			i += 2
		case c == '"' || c == '\'':
			end := strings.IndexByte(line[i+1:], c)
			if end < 0 {
				return depth, multi
			}
			i += end + 1
		case c == '#':
			return depth, multi
		case c == '[':
			depth++
		case c == ']' && depth > 0:
			depth--
		}
	}
	return depth, multi
}

// tomlKeys splits a dotted TOML key, quoted parts included.
func tomlKeys(key string) []string {
	var keys []string
	var part strings.Builder
	quote := rune(0)
	for i := 0; i < len(key); i++ {
		c := rune(key[i])
		switch {
		case quote != 0 && c == '\\' && quote == '"' && i+1 < len(key):
			i++
			part.WriteByte(key[i])
		case quote != 0 && c == quote:
			quote = 0
		case quote != 0:
			part.WriteByte(key[i])
		case c == '"' || c == '\'':
			quote = c
		case c == '.':
			keys = append(keys, strings.TrimSpace(part.String()))
			part.Reset()
		case c != ' ' && c != '\t':
			part.WriteByte(key[i])
		}
	}
	return append(keys, strings.TrimSpace(part.String()))
}

func tomlKey(key string) string {
	if bareTOMLKey.MatchString(key) {
		return key
	}
	return tomlString(key)
}

func tomlString(value string) string {
	var out strings.Builder
	out.WriteByte('"')
	for _, r := range value {
		switch {
		case r == '"' || r == '\\':
			out.WriteByte('\\')
			out.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&out, "\\u%04X", r)
		default:
			out.WriteRune(r)
		}
	}
	out.WriteByte('"')
	return out.String()
}

func tomlTable(values map[string]string, expand func(string) string) string {
	keys := slices.Sorted(maps.Keys(values))
	parts := make([]string, len(keys))
	for i, key := range keys {
		parts[i] = tomlKey(key) + " = " + tomlString(expand(values[key]))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}
