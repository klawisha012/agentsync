package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// wrapCommand runs a hook command through agentsync when the hook reads its input in another
// format than the running agent's (ADR 0039). The command stays one shell word after "--".
func wrapCommand(from, to, event, command string) string {
	return "agentsync hook --from " + from + " --to " + to + " --event " + event + " -- " + shellQuote(command)
}

// unwrapCommand gives back the input format and the command of a wrapped hook command.
func unwrapCommand(command string) (string, string, bool) {
	rest, ok := strings.CutPrefix(command, "agentsync hook ")
	if !ok {
		return "", "", false
	}
	flags, quoted, ok := strings.Cut(rest, " -- ")
	if !ok {
		return "", "", false
	}
	fields := strings.Fields(flags)
	to := ""
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == "--to" {
			to = fields[i+1]
		}
	}
	inner, ok := shellUnquote(quoted)
	return to, inner, ok && to != ""
}

func shellQuote(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}

func shellUnquote(text string) (string, bool) {
	if len(text) < 2 || text[0] != '\'' || text[len(text)-1] != '\'' {
		return "", false
	}
	return strings.ReplaceAll(text[1:len(text)-1], `'\''`, "'"), true
}

// hookPayload is what one format calls the parts of a hook input.
type hookPayload struct {
	event, tool, input, response string
	// inputText means the tool input travels as a JSON string, as Copilot's toolArgs.
	inputText bool
}

var (
	claudePayload  = hookPayload{event: "hook_event_name", tool: "tool_name", input: "tool_input", response: "tool_response"}
	agentsPayload  = hookPayload{event: "event", tool: "tool", input: "tool_input", response: "tool_response"}
	copilotPayload = hookPayload{tool: "toolName", input: "toolArgs", inputText: true}
)

func payloadOf(format string) hookPayload {
	switch format {
	case "agents":
		return agentsPayload
	case "github-copilot":
		return copilotPayload
	}
	return claudePayload
}

// translateInput turns the running agent's hook input into the format the hook reads. A field the
// target format has no place for is left out; a field the source lacks stays absent.
func translateInput(from, to, word string, raw []byte) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, errors.New("вход хука не разобран как JSON")
	}
	src, dst := payloadOf(from), payloadOf(to)
	out := map[string]json.RawMessage{}
	moved := map[string]bool{}
	move := func(fromKey, toKey string, value json.RawMessage) {
		moved[fromKey] = true
		if toKey != "" && value != nil {
			out[toKey] = value
		}
	}
	if dst.event != "" {
		name := word
		if to != "agents" {
			name = dialects[to].events[word]
		}
		out[dst.event] = marshalPlain(name)
	}
	moved[src.event] = true
	if value, ok := fields[src.tool]; ok {
		var native string
		if json.Unmarshal(value, &native) != nil {
			return nil, errors.New("имя инструмента во входе хука не разобрано")
		}
		move(src.tool, dst.tool, marshalPlain(renameTool(from, to, native)))
	}
	if value, ok := fields[src.input]; ok {
		if src.inputText {
			var text string
			if json.Unmarshal(value, &text) != nil {
				return nil, errors.New("аргументы инструмента во входе хука не разобраны")
			}
			value = json.RawMessage(text)
		}
		if dst.inputText {
			value = marshalPlain(string(value))
		}
		move(src.input, dst.input, value)
	}
	if value, ok := fields[src.response]; ok {
		move(src.response, dst.response, value)
	}
	// session_id, cwd, prompt and the like keep their names in every format.
	for key, value := range fields {
		if !moved[key] {
			out[key] = value
		}
	}
	return marshalPlain(out), nil
}

// renameTool gives the hook's name for the running agent's tool; an unknown tool keeps its name.
func renameTool(from, to, native string) string {
	word, ok := native, true
	if from != "agents" {
		word, ok = dialects[from].agentsTool(native)
	}
	if !ok {
		return native
	}
	if to == "agents" {
		return word
	}
	if names, ok := dialects[to].nativeTool(word); ok && len(names) > 0 {
		return names[0]
	}
	return native
}

// hookAnswer is what a hook decided: block the action with a reason, or let it go with its output.
type hookAnswer struct {
	block  bool
	reason string
	code   int
	stdout []byte
	stderr []byte
}

// readAnswer reads a hook's exit code and output in its own format.
func readAnswer(format string, code int, stdout, stderr []byte) hookAnswer {
	answer := hookAnswer{code: code, stdout: stdout, stderr: stderr}
	if code == 2 && format != "github-copilot" {
		return hookAnswer{block: true, reason: strings.TrimSpace(string(stderr))}
	}
	var decision struct {
		Decision           string `json:"decision"`
		Reason             string `json:"reason"`
		PermissionDecision string `json:"permissionDecision"`
		PermissionReason   string `json:"permissionDecisionReason"`
		Specific           struct {
			PermissionDecision string `json:"permissionDecision"`
			PermissionReason   string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if code != 0 || json.Unmarshal(bytes.TrimSpace(stdout), &decision) != nil {
		return answer
	}
	switch {
	case decision.Decision == "block" || decision.Decision == "deny":
		return hookAnswer{block: true, reason: decision.Reason}
	case decision.PermissionDecision == "deny":
		return hookAnswer{block: true, reason: decision.PermissionReason}
	case decision.Specific.PermissionDecision == "deny":
		return hookAnswer{block: true, reason: decision.Specific.PermissionReason}
	}
	return answer
}

// writeAnswer gives the running agent the hook's answer in its own format and returns the exit code.
func writeAnswer(format string, answer hookAnswer, stdout, stderr io.Writer) int {
	if !answer.block {
		_, _ = stdout.Write(answer.stdout)
		_, _ = stderr.Write(answer.stderr)
		return answer.code
	}
	if format == "github-copilot" {
		raw, _ := json.Marshal(map[string]string{"permissionDecision": "deny", "permissionDecisionReason": answer.reason})
		fmt.Fprintln(stdout, string(raw))
		return 0
	}
	fmt.Fprintln(stderr, answer.reason)
	return 2
}

// RunHook is agentsync hook: it translates the input of the running agent (from) into the format
// the hook reads (to), runs the command and translates the answer back. A failure of its own on a
// before_* event blocks the action; on any other event it lets the action go.
func RunHook(from, to, word, command string, stdin io.Reader, stdout, stderr io.Writer) int {
	failed := func(err error) int {
		reason := "agentsync hook: " + err.Error()
		if strings.HasPrefix(word, "before_") {
			return writeAnswer(from, hookAnswer{block: true, reason: reason}, stdout, stderr)
		}
		fmt.Fprintln(stderr, reason)
		return 0
	}
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return failed(err)
	}
	input, err := translateInput(from, to, word, raw)
	if err != nil {
		return failed(err)
	}
	var out, errOut bytes.Buffer
	cmd := shellCommand(command)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader(input), &out, &errOut
	code := 0
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			return failed(err)
		}
		code = exit.ExitCode()
	}
	return writeAnswer(from, readAnswer(to, code, out.Bytes(), errOut.Bytes()), stdout, stderr)
}

// shellCommand runs a hook command as the agents run it: through sh, or cmd on Windows without sh.
func shellCommand(command string) *exec.Cmd {
	if _, err := exec.LookPath("sh"); err == nil || runtime.GOOS != "windows" {
		// The command travels in the environment: MSYS sh on Windows misreads a -c argument that starts with a quote.
		cmd := exec.Command("sh", "-c", `eval "$AGENTSYNC_HOOK_COMMAND"`)
		cmd.Env = append(os.Environ(), "AGENTSYNC_HOOK_COMMAND="+command)
		return cmd
	}
	return exec.Command("cmd", "/C", command)
}

// agentsyncMissing reports that hooks with a wrapper will not run: agentsync is not on PATH.
func agentsyncMissing() bool {
	_, err := exec.LookPath("agentsync")
	return err != nil
}
