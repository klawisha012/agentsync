package server

import (
	"strings"
	"unicode"
)

func safeAgentName(name string) string {
	runes := []rune(name)
	if len(runes) == 0 || len(runes) > 64 || name == "." || name == ".." || strings.Contains(name, "..") {
		return "Имя ИИ-агента не подходит."
	}
	if runes[0] == '.' || runes[len(runes)-1] == '.' || runes[0] == ' ' || runes[len(runes)-1] == ' ' {
		return "Имя ИИ-агента не подходит."
	}
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '.' || r == '-' || r == '_' {
			continue
		}
		return "Имя ИИ-агента не подходит."
	}
	return ""
}

func machineID(id string) string {
	if len(id) < 32 || len(id) > 128 {
		return "Идентификатор компьютера должен быть длинной случайной строкой."
	}
	for _, r := range id {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return "Идентификатор компьютера должен быть длинной случайной строкой."
		}
	}
	return ""
}
