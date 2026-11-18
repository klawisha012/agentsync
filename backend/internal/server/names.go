package server

import "strings"

const canonAgent = "agents"

func isCanonAgentName(name string) bool {
	return sameAgent(strings.TrimSpace(name), canonAgent) || sameAgent(strings.TrimSpace(name), ".agents")
}

func resolveCanonAgent(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || isCanonAgentName(trimmed) {
		return canonAgent, true
	}
	return "", false
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
