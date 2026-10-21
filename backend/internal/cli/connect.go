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
		fmt.Fprintf(stdout, "ИИ-агент %s отключён от канона.\n", args[0])
		return nil
	}
	if err := agent.Connect(root, agent.EnvOrHome("AGENTSYNC_ACCOUNT", "account"), args[0]); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "ИИ-агент %s видит навыки канона.\n", args[0])
	return nil
}

func agentRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv("AGENTSYNC_ROOT"))
	if root != "" {
		return root, nil
	}
	return os.UserHomeDir()
}
