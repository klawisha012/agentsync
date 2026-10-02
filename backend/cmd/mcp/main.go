package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/klawisha012/agentsync/internal/agent"
	"github.com/klawisha012/agentsync/internal/mcp"
)

func main() {
	root := os.Getenv("AGENTSYNC_ROOT")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		root = home
	}
	server := os.Getenv("AGENTSYNC_SERVER")
	if server == "" {
		server = "https://zwarder.ru/api"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := mcp.Run(ctx, os.Stdin, os.Stdout, mcp.Env{
		Root:    root,
		Server:  server,
		Token:   agent.EnvOrHome("AGENTSYNC_TOKEN", "token"),
		Account: agent.EnvOrHome("AGENTSYNC_ACCOUNT", "account"),
		Cookie:  agent.SessionCookie(),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
