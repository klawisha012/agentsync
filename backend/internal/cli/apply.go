package cli

import (
	"fmt"
	"strconv"
	"strings"
)

type applyCall struct {
	author  string
	version int
	paths   []string
}

func isCanonArg(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	return v == "agents" || v == ".agents"
}

func parseApplyArgs(args []string) (applyCall, error) {
	var call applyCall
	seenVersion := false
	versionIdx := -1
	var rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--version" || strings.HasPrefix(arg, "--version="):
			if seenVersion {
				return applyCall{}, fmt.Errorf("Номер версии указан дважды.")
			}
			value, next, err := flagValue(args, i, "--version")
			if err != nil {
				return applyCall{}, err
			}
			number, err := strconv.Atoi(value)
			if err != nil || number < 1 {
				return applyCall{}, fmt.Errorf("Назовите номер версии.")
			}
			call.version = number
			seenVersion = true
			versionIdx = len(rest)
			i = next
		case strings.HasPrefix(arg, "-"):
			return applyCall{}, usageError{text: fmt.Sprintf("Неизвестный аргумент «%s».", arg)}
		default:
			if call.author == "" {
				call.author = arg
				continue
			}
			rest = append(rest, arg)
		}
	}
	if call.author == "" {
		return applyCall{}, usageError{text: "Назовите автора."}
	}
	if !seenVersion {
		if len(rest) == 0 || (len(rest) == 1 && isCanonArg(rest[0])) {
			return call, nil
		}
		if len(rest) == 1 && !strings.Contains(rest[0], "/") {
			return applyCall{}, fmt.Errorf("Публикуется и\u00a0применяется только agents: запустите agentsync apply %s.", call.author)
		}
		return applyCall{}, fmt.Errorf("Назовите номер версии.")
	}
	// Accept optional "agents" right after <author> when --version follows with element paths.
	if versionIdx == 1 && len(rest) >= 2 && isCanonArg(rest[0]) {
		rest = rest[1:]
	}
	if len(rest) == 0 {
		return applyCall{}, fmt.Errorf("Назовите элемент.")
	}
	call.paths = rest
	return call, nil
}

func flagValue(args []string, i int, name string) (string, int, error) {
	arg := args[i]
	prefix := name + "="
	if strings.HasPrefix(arg, prefix) {
		value := strings.TrimPrefix(arg, prefix)
		if strings.TrimSpace(value) == "" {
			return "", i, fmt.Errorf("Укажите значение %s.", name)
		}
		return value, i, nil
	}
	if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
		return "", i, fmt.Errorf("Укажите значение %s.", name)
	}
	return args[i+1], i + 1, nil
}
