package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/klawisha012/agentsync/internal/agent"
)

func runExport(args []string, stdout io.Writer) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return usageError{text: fmt.Sprintf("Неизвестный аргумент «%s».", arg)}
		}
	}
	root, err := agentRoot()
	if err != nil {
		return err
	}
	account := agent.EnvOrHome("AGENTSYNC_ACCOUNT", "account")
	items, err := agent.Export(root, account, args)
	if err != nil {
		return err
	}
	if len(args) == 1 && len(items) == 1 {
		lines := items[0].Report.Lines()
		if len(lines) == 0 {
			fmt.Fprintln(stdout, "В\u00a0agents нечего переносить.")
		}
		for _, line := range lines {
			fmt.Fprintln(stdout, line)
		}
		return items[0].Err
	}
	printed := false
	failed := 0
	for _, item := range items {
		for _, line := range item.Report.Lines() {
			fmt.Fprintf(stdout, "%s: %s\n", item.Name, line)
			printed = true
		}
		if item.Err != nil {
			failed++
			fmt.Fprintf(stdout, "%s: %s\n", item.Name, item.Err.Error())
			printed = true
		}
	}
	if !printed {
		fmt.Fprintln(stdout, "В\u00a0agents нечего переносить.")
	}
	if failed > 0 {
		return errors.New("Не все ИИ-агенты перенесены в\u00a0agents.")
	}
	return nil
}
