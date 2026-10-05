package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/klawisha012/agentsync/internal/agent"
)

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		if piped(stdin) {
			return applyStream(stdin, stderr)
		}
		fmt.Fprint(stdout, rootHelp())
		return 0
	}
	if isHelpFlag(args[0]) {
		fmt.Fprint(stdout, rootHelp())
		return 0
	}
	if args[0] == "help" {
		return runHelp(args[1:], stdout, stderr)
	}
	item, ok := commandByName(args[0])
	if !ok {
		fmt.Fprintf(stderr, "Неизвестная команда «%s».\nСправка: agentsync help\n", args[0])
		return 2
	}
	if hasHelpFlag(args[1:]) {
		fmt.Fprint(stdout, item.help())
		return 0
	}
	if len(args)-1 < item.minArgs {
		fmt.Fprintln(stderr, item.missing)
		fmt.Fprint(stderr, item.help())
		return 2
	}
	if err := item.execute(args[1:], stdout); err != nil {
		var usage usageError
		if errors.As(err, &usage) {
			fmt.Fprintln(stderr, usage.Error())
			fmt.Fprint(stderr, item.help())
			return 2
		}
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func runHelp(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || isHelpFlag(args[0]) {
		fmt.Fprint(stdout, rootHelp())
		return 0
	}
	item, ok := commandByName(args[0])
	if !ok {
		fmt.Fprintf(stderr, "Неизвестная команда «%s».\nСправка: agentsync help\n", args[0])
		return 2
	}
	fmt.Fprint(stdout, item.help())
	return 0
}

func isHelpFlag(arg string) bool {
	return arg == "-h" || arg == "--help"
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if isHelpFlag(arg) {
			return true
		}
	}
	return false
}

func (item spec) execute(args []string, stdout io.Writer) error {
	if item.name == "fanout" || item.name == "unfanout" {
		return runShare(item.name, args)
	}
	root, server, token, account, cookie, err := clientSession()
	if err != nil {
		return err
	}
	ctx := context.Background()
	if (item.name == "push" || item.name == "record") && token == "" && cookie == "" {
		return errors.New("Войдите в\u00a0аккаунт: agentsync login")
	}
	switch item.name {
	case "login":
		email := ""
		if len(args) > 0 {
			email = args[0]
		}
		return login(ctx, server, email, stdout)
	case "push":
		return agent.Push(
			ctx,
			server,
			root,
			args[0],
			token,
			cookie,
		)
	case "record":
		raw, err := agent.Record(
			ctx,
			server,
			root,
			args[0],
			token,
			cookie,
		)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(raw))
		return nil
	case "store":
		number := ""
		if len(args) >= 2 {
			number = args[1]
		}
		raw, err := agent.ReadStore(
			ctx,
			server,
			root,
			args[0],
			number,
			token,
			cookie,
		)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(raw))
		return nil
	case "revert":
		return agent.Revert(root, account, args[0])
	case "skills":
		call, err := parseSkillArgs(args)
		if err != nil {
			return err
		}
		call, err = fillSkillChoice(call)
		if err != nil {
			return err
		}
		return agent.CopySkills(ctx, server, root, account, token, cookie, agent.SkillCopy{
			Author:  call.author,
			Source:  call.source,
			Version: call.version,
			Skills:  call.skills,
			Targets: call.targets,
			Place:   call.place,
		})
	case "apply":
		return agent.Apply(
			ctx,
			server,
			root,
			args[0],
			args[1],
			account,
			token,
			cookie,
		)
	default:
		return fmt.Errorf("неизвестная команда")
	}
}

func applyStream(stdin io.Reader, stderr io.Writer) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if strings.TrimSpace(string(raw)) == "" {
		fmt.Fprintln(stderr, "Пустая публикация.")
		return 1
	}
	root, server, token, account, cookie, err := clientSession()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if err := agent.ApplyBody(context.Background(), server, root, account, token, cookie, raw); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

func piped(stdin io.Reader) bool {
	file, ok := stdin.(*os.File)
	if !ok {
		return stdin != nil
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}

func clientSession() (string, string, string, string, string, error) {
	root, err := agentRoot()
	if err != nil {
		return "", "", "", "", "", err
	}
	server := strings.TrimSpace(os.Getenv("AGENTSYNC_SERVER"))
	if server == "" {
		server = "https://zwarder.ru/api"
	}
	return root, server, agent.EnvOrHome("AGENTSYNC_TOKEN", "token"), agent.EnvOrHome("AGENTSYNC_ACCOUNT", "account"), agent.SessionCookie(), nil
}
