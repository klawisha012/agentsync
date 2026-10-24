package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/klawisha012/agentsync/internal/agent"
)

type usageError struct {
	text string
}

func (e usageError) Error() string {
	return e.text
}

// connectArgs turns "agentsync <ии-агент> on|off" into "agentsync on|off <ии-агент>".
func connectArgs(args []string) []string {
	if len(args) != 2 || (args[1] != "on" && args[1] != "off") {
		return args
	}
	if _, known := commandByName(args[0]); known {
		return args
	}
	return []string{args[1], args[0]}
}

func runConnect(name string, args []string, stdout io.Writer) error {
	if len(args) > 1 {
		return usageError{text: fmt.Sprintf("Неизвестный аргумент «%s».", args[1])}
	}
	root, err := agentRoot()
	if err != nil {
		return err
	}
	if name == "off" {
		if err := agent.Disconnect(root, args[0]); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "ИИ-агент %s отключён от agents.\n", args[0])
		return nil
	}
	report, err := agent.Connect(root, agent.EnvOrHome("AGENTSYNC_ACCOUNT", "account"), args[0])
	if err != nil {
		return err
	}
	if len(report.Got) == 0 {
		fmt.Fprintf(stdout, "ИИ-агент %s ничего не получил из agents.\n", args[0])
	} else {
		fmt.Fprintf(stdout, "ИИ-агент %s видит из agents: %s.\n", args[0], strings.Join(report.Got, ", "))
	}
	if len(report.Unset) > 0 {
		fmt.Fprintf(stdout, "Переменные секрета вписаны как есть, их значения нет в\u00a0окружении и\u00a0~/.agents/.env: %s.\n", strings.Join(report.Unset, ", "))
	}
	if len(report.Skipped) > 0 {
		fmt.Fprintf(stdout, "Пропущены виды, которых ИИ-агент не понимает: %s.\n", strings.Join(report.Skipped, ", "))
	}
	for _, line := range report.Warnings {
		fmt.Fprintln(stdout, line)
	}
	return nil
}

// runHook is agentsync hook --from <ИИ-агент> --to <формат> --event <слово> -- <команда>.
func runHook(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := map[string]string{}
	for len(args) >= 2 && args[0] != "--" {
		flags[args[0]], args = args[1], args[2:]
	}
	from, to, event := flags["--from"], flags["--to"], flags["--event"]
	if len(args) < 2 || args[0] != "--" || from == "" || to == "" || event == "" {
		fmt.Fprintln(stderr, "Вызов: agentsync hook --from <ИИ-агент> --to <формат> --event <событие> -- <команда>")
		return 2
	}
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	return agent.RunHook(from, to, event, strings.Join(args[1:], " "), stdin, stdout, stderr)
}

func agentRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv("AGENTSYNC_ROOT"))
	if root != "" {
		return root, nil
	}
	return os.UserHomeDir()
}
