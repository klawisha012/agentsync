package main

import (
	"context"
	"fmt"
	"os"

	"github.com/klawisha012/agentsync/internal/agent"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "команда: push, revert")
		os.Exit(2)
	}
	root := os.Getenv("AGENTSYNC_ROOT")
	server := os.Getenv("AGENTSYNC_SERVER")
	token := os.Getenv("AGENTSYNC_TOKEN")
	account := os.Getenv("AGENTSYNC_ACCOUNT")
	cookie := os.Getenv("AGENTSYNC_COOKIE")
	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "push":
		if len(os.Args) < 3 {
			err = fmt.Errorf("назовите ИИ-агента")
			break
		}
		err = agent.Push(ctx, server, root, os.Args[2], token, cookie)
	case "revert":
		if len(os.Args) < 3 {
			err = fmt.Errorf("назовите ИИ-агента")
			break
		}
		err = agent.Revert(root, account, os.Args[2])
	case "apply":
		if len(os.Args) < 4 {
			err = fmt.Errorf("назовите автора и ИИ-агента")
			break
		}
		err = agent.Apply(ctx, server, root, os.Args[2], os.Args[3], token, cookie)
	default:
		err = fmt.Errorf("неизвестная команда")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
