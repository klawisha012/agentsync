package cli

import (
	"fmt"
	"strconv"
	"strings"
)

type applyCall struct {
	author  string
	source  string
	version int
	paths   []string
}

func parseApplyArgs(args []string) (applyCall, error) {
	var call applyCall
	var positional []string
	seenVersion := false
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
			i = next
		case strings.HasPrefix(arg, "-"):
			return applyCall{}, fmt.Errorf("Неизвестный аргумент «%s».", arg)
		default:
			if len(positional) < 2 {
				positional = append(positional, arg)
				continue
			}
			call.paths = append(call.paths, arg)
		}
	}
	if len(positional) < 2 {
		return applyCall{}, fmt.Errorf("Назовите автора и ИИ-агента.")
	}
	if seenVersion && len(call.paths) == 0 {
		return applyCall{}, fmt.Errorf("Назовите элемент.")
	}
	if !seenVersion && len(call.paths) > 0 {
		return applyCall{}, fmt.Errorf("Назовите номер версии.")
	}
	call.author = positional[0]
	call.source = positional[1]
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
