package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var errOldHooks = errors.New("Хуки в ~/.agents/hooks.json в старом формате: нужен \"version\": 1 и слова agents. Перепишите файл по странице «Хуки».")

var projectDirVar = regexp.MustCompile(`\$\{` + projectDirWord + `\}|\$` + projectDirWord + `\b`)

// canonHookFile is ~/.agents/hooks.json in the agents format: hook groups under agents words, and
// extension sections under an AI agent's catalog name with that agent's native events. Its record
// names are "<word> <group>" for the core and "@<slug> <native event> <group>" for an extension.
type canonHookFile struct {
	top  map[string]json.RawMessage
	core *hookSettings
	ext  map[string]*hookSettings
	// key is the top key an extension section is written under.
	key map[string]string
}

// canonHooks reads ~/.agents/hooks.json. ok is false without the file.
func canonHooks(root string) (*canonHookFile, bool, error) {
	file := filepath.Join(canonDir(root), "hooks.json")
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	hooks, err := readCanonHooks(file)
	return hooks, err == nil, err
}

func readCanonHooks(file string) (*canonHookFile, error) {
	raw, err := os.ReadFile(file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return parseCanonHooks(raw)
}

func parseCanonHooks(raw []byte) (*canonHookFile, error) {
	core, ok := parseHooks(raw)
	if !ok {
		return nil, errors.New("Файл ~/.agents/hooks.json не разобран как JSON с хуками.")
	}
	f := &canonHookFile{top: core.top, core: core, ext: map[string]*hookSettings{}, key: map[string]string{}}
	if len(f.top) == 0 {
		return f, nil
	}
	if string(f.top["version"]) != "1" {
		return nil, errOldHooks
	}
	for key, value := range f.top {
		receiver, found := FindReceiver(key)
		if key == "version" || key == "hooks" || !found {
			continue
		}
		section, ok := parseHooks(value)
		if !ok {
			return nil, fmt.Errorf("Раздел «%s» в ~/.agents/hooks.json не разобран.", key)
		}
		f.ext[receiver.Slug], f.key[receiver.Slug] = section, key
	}
	return f, f.validate()
}

// validate checks the core against the agents vocabulary.
func (f *canonHookFile) validate() error {
	for event, groups := range f.core.events {
		if !slices.Contains(eventWords, event) {
			return fmt.Errorf("В ~/.agents/hooks.json событие «%s» не из словаря agents.", event)
		}
		for _, group := range groups {
			var fields map[string]json.RawMessage
			if json.Unmarshal(group, &fields) != nil {
				return fmt.Errorf("Хук события «%s» в ~/.agents/hooks.json не разобран.", event)
			}
			if raw := fields["matcher"]; raw != nil && string(raw) != `"*"` {
				var names []string
				if json.Unmarshal(raw, &names) != nil {
					return fmt.Errorf("Matcher хука события «%s» в ~/.agents/hooks.json — список имён или \"*\".", event)
				}
				if !slices.Contains(toolEvents, event) {
					return fmt.Errorf("Matcher в ~/.agents/hooks.json есть только у событий инструментов, а не у «%s».", event)
				}
				for _, name := range names {
					if !knownTool(name) {
						return fmt.Errorf("В ~/.agents/hooks.json инструмент «%s» не из словаря agents.", name)
					}
				}
			}
			var input string
			if raw := fields["input"]; raw != nil && json.Unmarshal(raw, &input) != nil {
				return fmt.Errorf("Формат входа хука события «%s» в ~/.agents/hooks.json — строка.", event)
			}
			if _, found := FindReceiver(input); input != "" && input != "none" && input != "agents" && !found {
				return fmt.Errorf("В ~/.agents/hooks.json неизвестный формат входа хука «%s».", input)
			}
		}
	}
	return nil
}

func (f *canonHookFile) part(name string) (*hookSettings, string) {
	rest, ok := strings.CutPrefix(name, "@")
	if !ok {
		return f.core, name
	}
	slug, native, _ := strings.Cut(rest, " ")
	if f.ext[slug] == nil {
		f.ext[slug] = &hookSettings{top: map[string]json.RawMessage{}, events: map[string][]json.RawMessage{}}
		f.key[slug] = slug
	}
	return f.ext[slug], native
}

func (f *canonHookFile) get(name string) (string, bool) {
	part, native := f.part(name)
	return part.get(native)
}

func (f *canonHookFile) set(name, record string) {
	part, native := f.part(name)
	part.set(native, record)
}

func (f *canonHookFile) remove(name string) {
	part, native := f.part(name)
	part.remove(native)
}

func (f *canonHookFile) encode() string {
	for slug, section := range f.ext {
		section.encode()
		if len(section.events) == 0 {
			delete(f.top, f.key[slug])
			continue
		}
		f.top[f.key[slug]] = marshalPlain(section.top)
	}
	f.top["version"] = json.RawMessage("1")
	return f.core.encode()
}

// canonRecordBody is the group a canon record name holds.
func canonRecordBody(name string) string {
	skip := 1
	if strings.HasPrefix(name, "@") {
		skip = 2
	}
	parts := strings.SplitN(name, " ", skip+1)
	return parts[len(parts)-1]
}

// hookSkips collects what an AI agent does not get from the canon hooks.
type hookSkips struct {
	events, tools []string
}

func (s hookSkips) report() []string {
	var out []string
	if len(s.events) > 0 {
		out = append(out, "хуки событий "+strings.Join(sortedUnique(s.events), ", "))
	}
	if len(s.tools) > 0 {
		out = append(out, "инструменты хуков "+strings.Join(sortedUnique(s.tools), ", "))
	}
	return out
}

func sortedUnique(values []string) []string {
	out := slices.Clone(values)
	slices.Sort(out)
	return slices.Compact(out)
}

// agentHooks translates the canon hooks into the AI agent's records. origins names the canon record
// each one came from, so a card of this agent gives the canon back its own records.
func agentHooks(file *canonHookFile, receiver Receiver) (map[string]string, map[string]string, hookSkips) {
	d := dialects[receiver.Slug]
	records, origins := map[string]string{}, map[string]string{}
	var skips hookSkips
	for name, group := range file.core.records() {
		word, _, _ := strings.Cut(name, " ")
		event, bodies, dropped, skip := toNative(d, word, group)
		skips.tools = append(skips.tools, dropped...)
		switch skip {
		case "event":
			skips.events = append(skips.events, word)
			continue
		case "tools":
			continue
		}
		for _, body := range bodies {
			native := hookName(event, json.RawMessage(body))
			records[native], origins[native] = body, name
		}
	}
	if section := file.ext[receiver.Slug]; section != nil {
		for name, group := range section.records() {
			records[name], origins[name] = compactJSON(group), "@"+receiver.Slug+" "+name
		}
	}
	return records, origins, skips
}

// toNative turns one canon group into the agent's records: one group, or one entry per command
// for a flat dialect. dropped are the matcher names the agent
// has no tool for; skip is "event" when the agent has no such event and "tools" when no matcher name is left.
func toNative(d dialect, word string, group json.RawMessage) (string, []string, []string, string) {
	event, ok := d.events[word]
	if !ok {
		return "", nil, nil, "event"
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(group, &fields) != nil {
		return "", nil, nil, "event"
	}
	delete(fields, "input")
	var dropped []string
	if raw := fields["matcher"]; raw != nil {
		var names []string
		if json.Unmarshal(raw, &names) != nil {
			delete(fields, "matcher")
		} else {
			var parts []string
			for _, name := range names {
				natives, ok := d.nativeTool(name)
				if !ok {
					dropped = append(dropped, name)
					continue
				}
				for _, native := range natives {
					if !slices.Contains(parts, native) {
						parts = append(parts, native)
					}
				}
			}
			// An empty list never means every tool.
			if len(parts) == 0 {
				return "", nil, dropped, "tools"
			}
			fields["matcher"] = marshalPlain(strings.Join(parts, "|"))
		}
	}
	native := "${PWD}"
	if d.projectDir != "" {
		native = "${" + d.projectDir + "}"
	}
	fields["hooks"] = mapHooks(fields["hooks"], func(command string) string {
		return projectDirVar.ReplaceAllLiteralString(command, native)
	}, func(seconds float64) float64 { return seconds * float64(d.timeoutScale) })
	if !d.flat {
		return event, []string{string(marshalPlain(fields))}, dropped, ""
	}
	var hooks []map[string]json.RawMessage
	_ = json.Unmarshal(fields["hooks"], &hooks)
	bodies := make([]string, 0, len(hooks))
	for _, hook := range hooks {
		renameKey(hook, "command", "bash")
		renameKey(hook, "timeout", "timeoutSec")
		if fields["matcher"] != nil {
			hook["matcher"] = fields["matcher"]
		}
		bodies = append(bodies, string(marshalPlain(hook)))
	}
	return event, bodies, dropped, ""
}

func renameKey(fields map[string]json.RawMessage, from, to string) {
	if value, ok := fields[from]; ok {
		fields[to] = value
		delete(fields, from)
	}
}

// fromNative turns one native group of the agent into a canon record name and body. A group the
// vocabulary cannot express (an event or a tool of this agent only, a regex) goes to its extension section.
func fromNative(d dialect, event string, group json.RawMessage) (string, string) {
	ext := "@" + d.slug + " " + hookName(event, group)
	word, ok := d.eventWord(event)
	var fields map[string]json.RawMessage
	if !ok || json.Unmarshal(group, &fields) != nil {
		return ext, compactJSON(group)
	}
	if d.flat {
		// One flat entry is a group with one command.
		if fields["bash"] == nil {
			return ext, compactJSON(group)
		}
		hook, matcher := maps.Clone(fields), fields["matcher"]
		delete(hook, "matcher")
		renameKey(hook, "bash", "command")
		renameKey(hook, "timeoutSec", "timeout")
		fields = map[string]json.RawMessage{"hooks": marshalPlain([]map[string]json.RawMessage{hook})}
		if matcher != nil {
			fields["matcher"] = matcher
		}
	}
	if raw := fields["matcher"]; raw != nil {
		var matcher string
		if json.Unmarshal(raw, &matcher) != nil {
			return ext, compactJSON(group)
		}
		matcher = strings.TrimSpace(matcher)
		switch {
		case matcher == "" || matcher == "*" || matcher == ".*":
			delete(fields, "matcher")
		case !slices.Contains(toolEvents, word):
			return ext, compactJSON(group)
		default:
			var names []string
			for _, part := range strings.Split(matcher, "|") {
				name, ok := d.agentsTool(strings.TrimSpace(part))
				if !ok {
					return ext, compactJSON(group)
				}
				if !slices.Contains(names, name) {
					names = append(names, name)
				}
			}
			fields["matcher"] = marshalPlain(names)
		}
	}
	nativeDir := regexp.MustCompile(`\$\{` + regexp.QuoteMeta(d.projectDir) + `\}|\$` + regexp.QuoteMeta(d.projectDir) + `\b`)
	fields["hooks"] = mapHooks(fields["hooks"], func(command string) string {
		if d.projectDir == "" {
			return command
		}
		return nativeDir.ReplaceAllLiteralString(command, "${"+projectDirWord+"}")
	}, func(value float64) float64 {
		seconds := value / float64(d.timeoutScale)
		if seconds != float64(int(seconds)) {
			seconds = float64(int(seconds) + 1)
		}
		return seconds
	})
	fields["input"] = marshalPlain(d.slug)
	body := string(marshalPlain(fields))
	return hookName(word, json.RawMessage(body)), body
}

// mapHooks rewrites the command and the timeout of each hook of a group.
func mapHooks(raw json.RawMessage, command func(string) string, timeout func(float64) float64) json.RawMessage {
	var hooks []map[string]json.RawMessage
	if json.Unmarshal(raw, &hooks) != nil {
		return raw
	}
	for _, hook := range hooks {
		var text string
		if json.Unmarshal(hook["command"], &text) == nil {
			hook["command"] = marshalPlain(command(text))
		}
		var value float64
		if json.Unmarshal(hook["timeout"], &value) == nil {
			hook["timeout"] = json.RawMessage(strconv.FormatFloat(timeout(value), 'f', -1, 64))
		}
	}
	return marshalPlain(hooks)
}

// checkCanon refuses a snapshot of agents whose hooks are not in the agents format.
func checkCanon(root, agentName string) error {
	if !isCanonCard(agentName) {
		return nil
	}
	_, _, err := canonHooks(root)
	return err
}

// startVersionedFile creates a hooks file that must carry "version": 1, as Copilot's does.
func startVersionedFile(file string) error {
	if _, err := os.Stat(file); !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, []byte("{\"version\": 1}\n"), 0o600)
}
