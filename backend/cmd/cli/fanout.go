package main

import (
	"fmt"
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

type fanoutCall struct {
	agent   string
	targets []string
}

func runShare(name string, args []string) error {
	call, err := parseFanoutArgs(args)
	if err != nil {
		return err
	}
	root, err := agentRoot()
	if err != nil {
		return err
	}
	if name == "unfanout" {
		return agent.Unfanout(root, call.agent, call.targets)
	}
	return agent.Fanout(root, call.agent, call.targets)
}

func parseFanoutArgs(args []string) (fanoutCall, error) {
	var call fanoutCall
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--into" || strings.HasPrefix(arg, "--into="):
			value, next, err := flagValue(args, i, "--into")
			if err != nil {
				return fanoutCall{}, err
			}
			call.targets = append(call.targets, value)
			i = next
		case strings.HasPrefix(arg, "-"):
			return fanoutCall{}, fmt.Errorf("Неизвестный аргумент «%s».", arg)
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) == 0 {
		return fanoutCall{}, usageError{text: "Назовите ИИ-агента."}
	}
	if len(positional) > 1 {
		return fanoutCall{}, fmt.Errorf("Неизвестный аргумент «%s».", positional[1])
	}
	call.agent = positional[0]
	return call, nil
}

func agentRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv("AGENTSYNC_ROOT"))
	if root != "" {
		return root, nil
	}
	return os.UserHomeDir()
}
