package agent

import (
	"fmt"
	"slices"
	"strings"
)

// The agents vocabulary (ADR 0039): hook events and tools, in a fixed order so a lookup by a
// native name that two words share always gives the same word.
var (
	eventWords = []string{
		"session_start", "session_end", "prompt_submit", "before_tool", "after_tool", "after_tool_failure",
		"stop", "subagent_start", "subagent_stop", "notification", "pre_compact", "post_compact",
	}
	toolWords = []string{"shell", "read", "edit", "write", "glob", "grep", "list", "web_fetch", "web_search", "task"}
	// toolEvents are the events whose matcher names tools.
	toolEvents = []string{"before_tool", "after_tool", "after_tool_failure"}
)

// projectDirWord is the agents variable for the project folder in a hook command.
const projectDirWord = "AGENTS_PROJECT_DIR"

// dialect is one AI agent's native names for the words of the agents format. A word missing from
// a dialect means the AI agent has no such hook event or tool.
type dialect struct {
	slug   string
	events map[string]string
	// tools lists the native names of a tool word; all of them go into a matcher.
	tools map[string][]string
	// mcpTool formats an MCP tool name from the server and the tool, for example "mcp__%s__%s".
	mcpTool string
	// projectDir is the native variable with the project folder; "" means the agent has none.
	projectDir string
	// timeoutScale turns a timeout in seconds into the agent's unit.
	timeoutScale int
	// flat means one hook entry per command, {type, matcher, bash, timeoutSec}, in a file with "version": 1.
	flat bool
}

// dialects is the agents vocabulary: connect, apply and the transfer into agents read it.
var dialects = map[string]dialect{
	"claude-code": {
		slug: "claude-code",
		events: map[string]string{
			"session_start": "SessionStart", "session_end": "SessionEnd", "prompt_submit": "UserPromptSubmit",
			"before_tool": "PreToolUse", "after_tool": "PostToolUse", "after_tool_failure": "PostToolUseFailure",
			"stop": "Stop", "subagent_start": "SubagentStart", "subagent_stop": "SubagentStop",
			"notification": "Notification", "pre_compact": "PreCompact",
		},
		tools: map[string][]string{
			"shell": {"Bash"}, "read": {"Read"}, "write": {"Write"}, "edit": {"Edit", "MultiEdit"},
			"glob": {"Glob"}, "grep": {"Grep"}, "list": {"LS"}, "web_fetch": {"WebFetch"},
			"web_search": {"WebSearch"}, "task": {"Task"},
		},
		mcpTool:      "mcp__%s__%s",
		projectDir:   "CLAUDE_PROJECT_DIR",
		timeoutScale: 1,
	},
	"gemini-cli": {
		slug: "gemini-cli",
		events: map[string]string{
			"session_start": "SessionStart", "session_end": "SessionEnd", "prompt_submit": "BeforeAgent",
			"before_tool": "BeforeTool", "after_tool": "AfterTool", "stop": "AfterAgent",
			"notification": "Notification", "pre_compact": "PreCompress",
		},
		tools: map[string][]string{
			"shell": {"run_shell_command"}, "read": {"read_file"}, "write": {"write_file"}, "edit": {"replace"},
			"glob": {"glob"}, "grep": {"search_file_content"}, "list": {"list_directory"},
			"web_fetch": {"web_fetch"}, "web_search": {"google_web_search"},
		},
		mcpTool:      "mcp_%s_%s",
		projectDir:   "GEMINI_PROJECT_DIR",
		timeoutScale: 1000,
	},
	"grok": {
		slug: "grok",
		events: map[string]string{
			"session_start": "SessionStart", "session_end": "SessionEnd", "prompt_submit": "UserPromptSubmit",
			"before_tool": "PreToolUse", "after_tool": "PostToolUse", "after_tool_failure": "PostToolUseFailure",
			"stop": "Stop", "subagent_start": "SubagentStart", "subagent_stop": "SubagentStop",
			"notification": "Notification", "pre_compact": "PreCompact", "post_compact": "PostCompact",
		},
		tools: map[string][]string{
			"shell": {"run_terminal_command"}, "read": {"read_file"}, "edit": {"search_replace"},
			"write": {"search_replace"}, "grep": {"grep"}, "list": {"list_dir"}, "web_search": {"web_search"},
			"task": {"spawn_subagent"},
		},
		mcpTool:      "%s__%s",
		timeoutScale: 1,
	},
	"codex": {
		slug: "codex",
		events: map[string]string{
			"session_start": "SessionStart", "prompt_submit": "UserPromptSubmit", "before_tool": "PreToolUse",
			"after_tool": "PostToolUse", "stop": "Stop",
		},
		tools:        map[string][]string{"shell": {"Bash"}, "edit": {"apply_patch"}, "write": {"apply_patch"}},
		mcpTool:      "mcp__%s__%s",
		timeoutScale: 1,
	},
	"github-copilot": {
		slug: "github-copilot",
		events: map[string]string{
			"session_start": "sessionStart", "session_end": "sessionEnd", "prompt_submit": "userPromptSubmitted",
			"before_tool": "preToolUse", "after_tool": "postToolUse", "after_tool_failure": "postToolUseFailure",
			"stop": "agentStop", "subagent_start": "subagentStart", "subagent_stop": "subagentStop",
			"notification": "notification", "pre_compact": "preCompact",
		},
		tools: map[string][]string{
			"shell": {"bash"}, "read": {"view"}, "write": {"create"}, "edit": {"edit"}, "glob": {"glob"},
			"grep": {"grep"}, "web_fetch": {"web_fetch"}, "task": {"task"},
		},
		mcpTool:      "%s/%s",
		timeoutScale: 1,
		flat:         true,
	},
}

func (d dialect) eventWord(native string) (string, bool) {
	for _, word := range eventWords {
		if name, ok := d.events[word]; ok && name == native {
			return word, true
		}
	}
	return "", false
}

func (d dialect) toolWord(native string) (string, bool) {
	for _, word := range toolWords {
		if slices.Contains(d.tools[word], native) {
			return word, true
		}
	}
	return "", false
}

// nativeTool turns one matcher name of the agents format into the agent's names: a tool word or
// "mcp:<server>/<tool>" with "*" for every tool of the server. ok is false when the agent has no such tool.
func (d dialect) nativeTool(name string) ([]string, bool) {
	if server, tool, ok := mcpName(name); ok {
		if d.mcpTool == "" {
			return nil, false
		}
		if tool == "*" {
			tool = ".*"
		}
		return []string{fmt.Sprintf(d.mcpTool, server, tool)}, true
	}
	names := d.tools[name]
	return names, len(names) > 0
}

// agentsTool turns one native matcher part into its agents name; ok is false when the part is not
// a plain tool name the vocabulary knows (a regex, a tool of one agent only).
func (d dialect) agentsTool(native string) (string, bool) {
	if word, ok := d.toolWord(native); ok {
		return word, true
	}
	if d.mcpTool == "" {
		return "", false
	}
	prefix, middle, _ := strings.Cut(d.mcpTool, "%s")
	middle, _, _ = strings.Cut(middle, "%s")
	rest, ok := strings.CutPrefix(native, prefix)
	if !ok {
		return "", false
	}
	server, tool, ok := strings.Cut(rest, middle)
	if !ok || server == "" || tool == "" {
		return "", false
	}
	if tool == ".*" {
		tool = "*"
	}
	if !plainName(server) || (tool != "*" && !plainName(tool)) {
		return "", false
	}
	return "mcp:" + server + "/" + tool, true
}

// mcpName splits "mcp:<server>/<tool>".
func mcpName(name string) (string, string, bool) {
	rest, ok := strings.CutPrefix(name, "mcp:")
	if !ok {
		return "", "", false
	}
	server, tool, ok := strings.Cut(rest, "/")
	if !ok || !plainName(server) || (tool != "*" && !plainName(tool)) {
		return "", "", false
	}
	return server, tool, true
}

// plainName is a tool or server name without regex characters.
func plainName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !(r == '_' || r == '-' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return true
}

// knownTool reports a matcher name of the agents format: a tool word or an MCP tool.
func knownTool(name string) bool {
	_, _, mcp := mcpName(name)
	return mcp || slices.Contains(toolWords, name)
}
