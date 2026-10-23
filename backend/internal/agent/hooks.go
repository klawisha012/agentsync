package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// hookPathVar is a path under a variable such as "$CLAUDE_PROJECT_DIR"/: it travels with the hook.
var hookPathVar = regexp.MustCompile(`\$\{?[A-Za-z_][A-Za-z0-9_]*\}?(?:\\?")?/`)

// hookSettings sees the hooks object of a JSON settings file as records named
// "<event> <group JSON>": one matcher group under one event.
type hookSettings struct {
	top    map[string]json.RawMessage
	events map[string][]json.RawMessage
}

func parseHooks(raw []byte) (*hookSettings, bool) {
	settings := &hookSettings{top: map[string]json.RawMessage{}, events: map[string][]json.RawMessage{}}
	if len(bytes.TrimSpace(raw)) == 0 {
		return settings, true
	}
	if json.Unmarshal(raw, &settings.top) != nil || settings.top == nil {
		return nil, false
	}
	if found := settings.top["hooks"]; found != nil && json.Unmarshal(found, &settings.events) != nil {
		return nil, false
	}
	if settings.events == nil {
		settings.events = map[string][]json.RawMessage{}
	}
	return settings, true
}

func readHookSettings(file string) (*hookSettings, error) {
	raw, err := os.ReadFile(file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	settings, ok := parseHooks(raw)
	if !ok {
		return nil, fmt.Errorf("Файл %s не разобран как JSON с хуками, записи не вписаны.", file)
	}
	return settings, nil
}

func compactJSON(raw json.RawMessage) string {
	var out bytes.Buffer
	if json.Compact(&out, raw) != nil {
		return string(raw)
	}
	return out.String()
}

// canonicalJSON is the form two equal groups share whatever their key order and escaping.
func canonicalJSON(raw json.RawMessage) string {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return compactJSON(raw)
	}
	return string(marshalPlain(value))
}

func hookName(event string, group json.RawMessage) string {
	return event + " " + canonicalJSON(group)
}

// records lists every hook group by its record name.
func (s *hookSettings) records() map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for event, groups := range s.events {
		for _, group := range groups {
			out[hookName(event, group)] = group
		}
	}
	return out
}

func (s *hookSettings) find(name string) (string, int) {
	event, group, _ := strings.Cut(name, " ")
	for i, have := range s.events[event] {
		if canonicalJSON(have) == group {
			return event, i
		}
	}
	return event, -1
}

func (s *hookSettings) get(name string) (string, bool) {
	event, i := s.find(name)
	if i < 0 {
		return "", false
	}
	return string(s.events[event][i]), true
}

func (s *hookSettings) set(name, record string) {
	if event, i := s.find(name); i < 0 {
		s.events[event] = append(s.events[event], json.RawMessage(record))
	}
}

func (s *hookSettings) remove(name string) {
	if event, i := s.find(name); i >= 0 {
		s.events[event] = slices.Delete(s.events[event], i, i+1)
	}
}

func (s *hookSettings) encode() string {
	for event, groups := range s.events {
		if len(groups) == 0 {
			delete(s.events, event)
		}
	}
	if len(s.events) == 0 {
		delete(s.top, "hooks")
	} else {
		s.top["hooks"] = marshalPlain(s.events)
	}
	return encodeTop(s.top)
}

// machineHooks lists the hook groups bound to this machine by an absolute path. /dev/null and a
// path under a variable such as "$CLAUDE_PROJECT_DIR"/ are not bound to one machine.
func (s *hookSettings) machineHooks() map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for name, group := range s.records() {
		text := strings.ReplaceAll(hookPathVar.ReplaceAllString(string(group), ""), "/dev/null", "")
		if AbsolutePath.MatchString(text) {
			out[name] = group
		}
	}
	return out
}

// hookFile reports a file that may hold hooks: a hook file or an AI agent settings file.
func hookFile(path string) bool {
	base := strings.ToLower(filepath.Base(filepath.ToSlash(path)))
	return IsHook(path) || base == "settings.json" || base == "settings.local.json"
}

// portableHooks drops machine hook groups from a hook or settings file with a hooks object.
// rest is the body without the hooks object, for the caller's own machine-path check. ok is
// false for any other file; then the caller judges it whole.
func portableHooks(path, body string) (string, string, bool) {
	if !hookFile(path) {
		return body, "", false
	}
	settings, ok := parseHooks([]byte(body))
	if !ok || settings.top["hooks"] == nil {
		return body, "", false
	}
	rest := maps.Clone(settings.top)
	delete(rest, "hooks")
	machine := settings.machineHooks()
	if len(machine) == 0 {
		return body, encodeTop(rest), true
	}
	for name := range machine {
		settings.remove(name)
	}
	return settings.encode(), encodeTop(rest), true
}

// onlyMachineHooks is a body with just the machine hook groups; "" when there are none.
func onlyMachineHooks(path, body string) string {
	if !hookFile(path) {
		return ""
	}
	settings, ok := parseHooks([]byte(body))
	if !ok {
		return ""
	}
	machine := settings.machineHooks()
	if len(machine) == 0 {
		return ""
	}
	kept := &hookSettings{top: map[string]json.RawMessage{}, events: map[string][]json.RawMessage{}}
	// The agents hooks file stays in the agents format.
	if version, ok := settings.top["version"]; ok {
		kept.top["version"] = version
	}
	for name, group := range machine {
		kept.set(name, string(group))
	}
	return kept.encode()
}

// keepMachineHooks adds the machine hook groups of the file on disk to a new body.
func keepMachineHooks(onDisk []byte, body string) string {
	old, ok := parseHooks(onDisk)
	if !ok {
		return body
	}
	machine := old.machineHooks()
	if len(machine) == 0 {
		return body
	}
	settings, ok := parseHooks([]byte(body))
	if !ok {
		return body
	}
	for name, group := range machine {
		settings.set(name, string(group))
	}
	return settings.encode()
}
