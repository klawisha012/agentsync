package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Gemini CLI names Claude Code hook events and tools its own way. A Claude event missing
// here has no Gemini counterpart, so its hooks are skipped.
var (
	geminiHookEvents = map[string]string{
		"PreToolUse": "BeforeTool", "PostToolUse": "AfterTool", "UserPromptSubmit": "BeforeAgent",
		"Stop": "AfterAgent", "SessionStart": "SessionStart", "SessionEnd": "SessionEnd",
		"PreCompact": "PreCompress", "Notification": "Notification",
	}
	geminiHookTools = map[string]string{
		"Bash": "run_shell_command", "Write": "write_file", "Edit": "replace", "Read": "read_file",
	}
)

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

func hookName(event string, group json.RawMessage) string {
	return event + " " + compactJSON(group)
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
		if compactJSON(have) == group {
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
		raw, _ := json.Marshal(s.events)
		s.top["hooks"] = raw
	}
	out, _ := json.MarshalIndent(s.top, "", "  ")
	return string(out) + "\n"
}

// machineHooks lists the hook groups bound to this machine by an absolute path.
func (s *hookSettings) machineHooks() map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for name, group := range s.records() {
		if AbsolutePath.Match(group) {
			out[name] = group
		}
	}
	return out
}

// portableHooks drops machine hook groups from a JSON body with a hooks object. ok is false when
// the body has no hooks object; then the caller judges the file whole.
func portableHooks(body string) (string, bool) {
	settings, ok := parseHooks([]byte(body))
	if !ok || settings.top["hooks"] == nil {
		return body, false
	}
	machine := settings.machineHooks()
	if len(machine) == 0 {
		return body, true
	}
	for name := range machine {
		settings.remove(name)
	}
	return settings.encode(), true
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

// canonHooks reads ~/.agents/hooks.json in the Claude Code form. ok is false without the file.
func canonHooks(root string) (map[string]json.RawMessage, bool, error) {
	file := filepath.Join(canonDir(root), "hooks.json")
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	settings, err := readHookSettings(file)
	if err != nil {
		return nil, false, err
	}
	return settings.records(), true, nil
}

// agentHooks translates canon hook groups into the AI agent's events and tool names. It returns
// the records and the canon events the agent does not have.
func agentHooks(hooks map[string]json.RawMessage, receiver Receiver) (map[string]string, []string) {
	out := map[string]string{}
	missing := map[string]bool{}
	for name, group := range hooks {
		event, _, _ := strings.Cut(name, " ")
		target, record, ok := translateHook(event, group, receiver.HookEvents, receiver.HookTools)
		if !ok {
			missing[event] = true
			continue
		}
		out[hookName(target, json.RawMessage(record))] = record
	}
	return out, slices.Sorted(maps.Keys(missing))
}

// translateHook renames the event and the tools in the matcher. Nil maps keep the Claude names.
func translateHook(event string, group json.RawMessage, events, tools map[string]string) (string, string, bool) {
	if events == nil {
		return event, compactJSON(group), true
	}
	target, ok := events[event]
	if !ok {
		return "", "", false
	}
	var fields map[string]json.RawMessage
	var matcher string
	if json.Unmarshal(group, &fields) != nil || json.Unmarshal(fields["matcher"], &matcher) != nil {
		return target, compactJSON(group), true
	}
	parts := strings.Split(matcher, "|")
	for i, part := range parts {
		if renamed, ok := tools[strings.TrimSpace(part)]; ok {
			parts[i] = renamed
		}
	}
	fields["matcher"], _ = json.Marshal(strings.Join(parts, "|"))
	raw, _ := json.Marshal(fields)
	return target, string(raw), true
}

func invert(names map[string]string) map[string]string {
	if names == nil {
		return nil
	}
	out := make(map[string]string, len(names))
	for from, to := range names {
		out[to] = from
	}
	return out
}
