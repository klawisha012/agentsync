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

// recordKind is one kind of named records on writes into a shared AI agent settings file.
type recordKind struct {
	ledger string // folder of the machine's ledgers in the state dir
	key    string // JSON object that holds the records
}

var (
	mcpRecords    = recordKind{ledger: "mcp", key: "mcpServers"}
	pluginRecords = recordKind{ledger: "plugins", key: "enabledPlugins"}
	marketRecords = recordKind{ledger: "marketplaces", key: "extraKnownMarketplaces"}
	hookRecords   = recordKind{ledger: "hooks", key: "hooks"}
	recordKinds   = []recordKind{mcpRecords, pluginRecords, marketRecords, hookRecords}
)

// canonRecordFile is the canon file that holds the records of a kind.
var canonRecordFile = map[recordKind]string{
	mcpRecords: "mcp.json", pluginRecords: "plugins.json", marketRecords: "plugins.json", hookRecords: "hooks.json",
}

// openSettings reads a settings file as the records of one kind.
func openSettings(kind recordKind, file string) (mcpSettings, error) {
	if kind == hookRecords {
		settings, err := readHookSettings(file)
		if err != nil {
			return nil, err
		}
		return settings, nil
	}
	return readSettings(file, kind.key)
}

// recordLedger is the machine's record of the records of one kind on wrote into one settings file.
// Replaced holds a foreign record on overwrote, so off can put it back.
type recordLedger struct {
	File     string            `json:"file"`
	Own      []string          `json:"own"`
	Replaced map[string]string `json:"replaced,omitempty"`
	Origins  map[string]string `json:"origins,omitempty"`
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

// canonPlugins reads ~/.agents/plugins.json: enabledPlugins and extraKnownMarketplaces in the
// Claude Code form. A plugin is a record only; the AI agent downloads it itself.
func canonPlugins(root string) (map[string]json.RawMessage, map[string]json.RawMessage, bool, error) {
	file := filepath.Join(canonDir(root), "plugins.json")
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return nil, nil, false, nil
	}
	plugins, err := readSettings(file, pluginRecords.key)
	if err != nil {
		return nil, nil, false, err
	}
	markets, err := readSettings(file, marketRecords.key)
	if err != nil {
		return nil, nil, false, err
	}
	return plugins.(*jsonSettings).servers, markets.(*jsonSettings).servers, true, nil
}

func recordBodies(records map[string]json.RawMessage) map[string]string {
	out := make(map[string]string, len(records))
	for name, record := range records {
		out[name] = string(record)
	}
	return out
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

// connectMCP writes the canon servers into the agent settings file in its format. It returns
// the secret variables the agent cannot expand and the machine has no value for.
func connectMCP(root string, receiver Receiver, servers map[string]json.RawMessage) ([]string, error) {
	file := receiver.mcpFile(root)
	records := map[string]string{}
	unset := map[string]bool{}
	lookup := secretLookup(root)
	for name, server := range servers {
		record, missing, err := renderRecord(lookup, receiver.Slug, file, name, server)
		if err != nil {
			return nil, err
		}
		for _, key := range missing {
			unset[key] = true
		}
		records[name] = record
	}
	return slices.Sorted(maps.Keys(unset)), writeRecords(mcpRecords, receiver.Slug, file, records, nil)
}

// writeRecords writes records into a settings file and removes its own records that are no
// longer asked for. Foreign records with other names and other keys stay. origins names the canon
// record each one came from, when it differs from the record itself.
func writeRecords(kind recordKind, slug, file string, records, origins map[string]string) error {
	ledgerPath, ledger, err := loadLedger(kind, slug)
	if err != nil {
		return err
	}
	if len(records) == 0 && len(ledger.Own) == 0 {
		return nil
	}
	if ledger.File != "" && !samePlace(ledger.File, file) {
		if err := releaseRecords(kind, ledger); err != nil {
			return err
		}
		ledger = recordLedger{}
	}
	settings, err := openSettings(kind, file)
	if err != nil {
		return err
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
	ledger.File, ledger.Own, ledger.Origins = file, names, origins
	// The ledger goes first: a record on wrote must never look foreign to the next on.
	if err := saveLedger(ledgerPath, ledger); err != nil {
		return err
	}
	return writeSettings(file, settings)
}

// disconnectRecords removes only the records on wrote and puts replaced foreign records back.
func disconnectRecords(slug string) error {
	for _, kind := range recordKinds {
		ledgerPath, ledger, err := loadLedger(kind, slug)
		if err != nil {
			return err
		}
		if ledger.File == "" {
			continue
		}
		if err := releaseRecords(kind, ledger); err != nil {
			return err
		}
		if err := os.Remove(ledgerPath); err != nil {
			return err
		}
	}
	return nil
}

func releaseRecords(kind recordKind, ledger recordLedger) error {
	settings, err := openSettings(kind, ledger.File)
	if err != nil {
		return err
	}
	for _, name := range ledger.Own {
		restore(settings, ledger, name)
	}
	return writeSettings(ledger.File, settings)
}

func restore(settings mcpSettings, ledger recordLedger, name string) {
	settings.remove(name)
	if old, ok := ledger.Replaced[name]; ok {
		settings.set(name, old)
		delete(ledger.Replaced, name)
	}
}

func loadLedger(kind recordKind, slug string) (string, recordLedger, error) {
	dir, err := stateDir()
	if err != nil {
		return "", recordLedger{}, err
	}
	file := filepath.Join(dir, kind.ledger, slug+".json")
	var ledger recordLedger
	raw, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return file, ledger, nil
	}
	if err != nil {
		return file, ledger, err
	}
	if err := json.Unmarshal(raw, &ledger); err != nil {
		return file, ledger, fmt.Errorf("Учёт записей %s не\u00a0разобран.", file)
	}
	return file, ledger, nil
}

func saveLedger(file string, ledger recordLedger) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(ledger, "", "  ")
	return os.WriteFile(file, raw, 0o600)
}

func isTOML(file string) bool {
	return strings.EqualFold(filepath.Ext(file), ".toml")
}

// readSettings reads the records under key of a JSON settings file, or the MCP tables of a TOML one.
func readSettings(file, key string) (mcpSettings, error) {
	raw, err := os.ReadFile(file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if isTOML(file) {
		return &tomlSettings{lines: splitLines(string(raw))}, nil
	}
	settings := &jsonSettings{top: map[string]json.RawMessage{}, key: key, servers: map[string]json.RawMessage{}}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return settings, nil
	}
	if json.Unmarshal(raw, &settings.top) != nil || settings.top == nil {
		return nil, fmt.Errorf("Файл %s не\u00a0разобран как JSON, записи не\u00a0вписаны.", file)
	}
	if found := settings.top[key]; found != nil && json.Unmarshal(found, &settings.servers) != nil {
		return nil, fmt.Errorf("Ключ %s в\u00a0файле %s не\u00a0разобран.", key, file)
	}
	if settings.servers == nil {
		settings.servers = map[string]json.RawMessage{}
	}
	return settings, nil
}

// writeSettings writes a settings file in place, not by rename: a link (even to a missing
// target), a hard link, the owner and the ACL stay, and a file another program holds open
// on Windows can still be written.
func writeSettings(file string, settings mcpSettings) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, []byte(settings.encode()), 0o600)
}

// UnsetVariables names secret variables written as is: the AI agent does not expand them and
// the machine has no value for them. The command itself succeeded.
type UnsetVariables []string

func (u UnsetVariables) Error() string {
	return "Переменные секрета вписаны как есть, их значения нет в\u00a0окружении и\u00a0~/.agents/.env: " + strings.Join(u, ", ") + "."
}

// renderRecord turns a canon server into the record of the settings format. JSON agents
// expand ${NAME} themselves; Codex does not, so the value comes from the environment or ~/.agents/.env.
// A variable without a value stays as written and comes back in missing.
func renderRecord(lookup func(string) (string, bool), slug, file, name string, server json.RawMessage) (string, []string, error) {
	if !isTOML(file) {
		return geminiRecord(slug, server), nil, nil
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
		return "", nil, fmt.Errorf("Сервер %s в ~/.agents/mcp.json не разобран.", name)
	}
	var missing []string
	expand := func(value string) string {
		return secretVariable.ReplaceAllStringFunc(value, func(match string) string {
			key := secretVariable.FindStringSubmatch(match)[1]
			found, ok := lookup(key)
			if !ok {
				missing = append(missing, key)
				return match
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
	return strings.Join(lines, "\n") + "\n", missing, nil
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
	key     string
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
	s.top[s.key] = marshalPlain(s.servers)
	return encodeTop(s.top)
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
