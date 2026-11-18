package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/klawisha012/agentsync/internal/agent"
)

func parseExportArgs(args []string) ([]string, []string, error) {
	var names, kinds []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--only" || strings.HasPrefix(arg, "--only=") {
			raw, next, err := flagValue(args, i, "--only")
			if err != nil {
				return nil, nil, usageError{text: err.Error()}
			}
			i = next
			parts := strings.Split(raw, ",")
			added := 0
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					return nil, nil, usageError{text: "Укажите значение --only."}
				}
				norm, ok := agent.NormalizeExportKind(part)
				if !ok {
					return nil, nil, usageError{text: fmt.Sprintf("Неизвестный вид «%s». Доступны: %s.", part, strings.Join(agent.ExportKinds(), ", "))}
				}
				kinds = append(kinds, norm)
				added++
			}
			if added == 0 {
				return nil, nil, usageError{text: "Укажите значение --only."}
			}
			continue
		}
		if strings.HasPrefix(arg, "--") {
			if norm, ok := agent.NormalizeExportKind(strings.TrimPrefix(arg, "--")); ok {
				kinds = append(kinds, norm)
				continue
			}
			return nil, nil, usageError{text: fmt.Sprintf("Неизвестный аргумент «%s».", arg)}
		}
		if strings.HasPrefix(arg, "-") {
			return nil, nil, usageError{text: fmt.Sprintf("Неизвестный аргумент «%s».", arg)}
		}
		names = append(names, arg)
	}
	return names, kinds, nil
}

func runExport(args []string, stdout io.Writer) error {
	names, kinds, err := parseExportArgs(args)
	if err != nil {
		return err
	}
	root, err := agentRoot()
	if err != nil {
		return err
	}
	account := agent.EnvOrHome("AGENTSYNC_ACCOUNT", "account")
	items, err := agent.Export(root, account, names, kinds)
	if err != nil {
		return err
	}
	if len(names) == 1 && len(items) == 1 {
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
