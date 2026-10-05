package main

import (
	"fmt"
	"strconv"
	"strings"
)

type skillCall struct {
	author  string
	source  string
	version int
	skills  []string
	targets []string
	place   string
}

func parseSkillArgs(args []string) (skillCall, error) {
	var call skillCall
	var positional []string
	seenVersion := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--version" || strings.HasPrefix(arg, "--version="):
			if seenVersion {
				return skillCall{}, fmt.Errorf("Номер версии указан дважды.")
			}
			value, next, err := flagValue(args, i, "--version")
			if err != nil {
				return skillCall{}, err
			}
			number, err := strconv.Atoi(value)
			if err != nil || number < 1 {
				return skillCall{}, fmt.Errorf("Назовите номер версии.")
			}
			call.version = number
			seenVersion = true
			i = next
		case arg == "--skill" || strings.HasPrefix(arg, "--skill="):
			value, next, err := flagValue(args, i, "--skill")
			if err != nil {
				return skillCall{}, err
			}
			call.skills = append(call.skills, value)
			i = next
		case arg == "--global" || arg == "--project":
			if call.place != "" {
				return skillCall{}, fmt.Errorf("Укажите одно место: глобальный каталог или проект.")
			}
			call.place = strings.TrimPrefix(arg, "--")
		case arg == "--into" || strings.HasPrefix(arg, "--into="):
			value, next, err := flagValue(args, i, "--into")
			if err != nil {
				return skillCall{}, err
			}
			call.targets = append(call.targets, value)
			i = next
		case strings.HasPrefix(arg, "-"):
			return skillCall{}, fmt.Errorf("Неизвестный аргумент «%s».", arg)
		default:
			if len(positional) < 2 {
				positional = append(positional, arg)
				continue
			}
			call.skills = append(call.skills, arg)
		}
	}
	if len(positional) < 2 {
		return skillCall{}, fmt.Errorf("Назовите автора и ИИ-агента.")
	}
	if !seenVersion {
		return skillCall{}, fmt.Errorf("Назовите номер версии.")
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
