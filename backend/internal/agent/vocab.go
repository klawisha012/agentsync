package agent

// dialect is one AI agent's native names for the words of the agents format. A word missing from
// a dialect means the AI agent has no such hook event or tool.
type dialect struct {
	slug   string
	events map[string]string
	// tools lists the native names of a tool word; the first one is the name written.
	tools map[string][]string
}

// dialects is the agents vocabulary (ADR 0039): connect and the transfer into agents read it.
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
	},
}

// shortcut: ~/.agents/hooks.json still holds Claude Code names, the matcher is a "|"-split regex and a
// tool without a counterpart stays as written; the agents format with name lists and refusals replaces this in #41.
var canonDialect = dialects["claude-code"]

func (d dialect) eventWord(native string) (string, bool) {
	for word, name := range d.events {
		if name == native {
			return word, true
		}
	}
	return "", false
}

func (d dialect) toolWord(native string) (string, bool) {
	for word, names := range d.tools {
		for _, name := range names {
			if name == native {
				return word, true
			}
		}
	}
	return "", false
}
