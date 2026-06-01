package repository

import (
	"agentsync/internal/domain"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Regexp для поиска и замены секретов вида ${secrets.VAR_NAME}
var secretPlaceholderRegex = regexp.MustCompile(`\$\{secrets\.([a-zA-Z0-9_]+)\}`)

// MergeJSONConfig выполняет умное слияние MCP-сервера в существующий JSON конфиг агента
func MergeJSONConfig(existingJSON []byte, mcp domain.MCPConfig, secrets map[string]string) ([]byte, error) {
	var configMap map[string]interface{}

	if len(existingJSON) == 0 || strings.TrimSpace(string(existingJSON)) == "" {
		configMap = make(map[string]interface{})
	} else {
		if err := json.Unmarshal(existingJSON, &configMap); err != nil {
			return nil, fmt.Errorf("ошибка парсинга существующего JSON: %w", err)
		}
	}

	// Получаем или создаем mcpServers
	mcpServersRaw, exists := configMap["mcpServers"]
	var mcpServers map[string]interface{}

	if !exists {
		mcpServers = make(map[string]interface{})
		configMap["mcpServers"] = mcpServers
	} else {
		var ok bool
		mcpServers, ok = mcpServersRaw.(map[string]interface{})
		if !ok {
			// На всякий случай пересоздаем, если там не map
			mcpServers = make(map[string]interface{})
			configMap["mcpServers"] = mcpServers
		}
	}

	// Готовим настройки для нашего сервера
	serverSettings := make(map[string]interface{})

	if mcp.ServerURL != "" {
		// HTTP/SSE сервер
		serverSettings["serverUrl"] = mcp.ServerURL
		if len(mcp.Headers) > 0 {
			headersCopy := make(map[string]interface{})
			for k, v := range mcp.Headers {
				headersCopy[k] = InjectSecrets(v, secrets)
			}
			serverSettings["headers"] = headersCopy
		}
	} else {
		// Стандартный CLI сервер
		serverSettings["command"] = mcp.Command

		// Копируем аргументы
		argsCopy := make([]interface{}, len(mcp.Args))
		for i, arg := range mcp.Args {
			argsCopy[i] = InjectSecrets(arg, secrets)
		}
		serverSettings["args"] = argsCopy

		// Обрабатываем переменные окружения с инъекцией секретов
		if len(mcp.Env) > 0 {
			envSettings := make(map[string]interface{})
			for k, v := range mcp.Env {
				envSettings[k] = InjectSecrets(v, secrets)
			}
			serverSettings["env"] = envSettings
		}
	}

	// Добавляем/перезаписываем только этот сервер
	mcpServers[mcp.Name] = serverSettings

	// Маршалим обратно с красивыми отступами
	result, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации результирующего JSON: %w", err)
	}

	return result, nil
}

// MergeMarkdownConfig выполняет слияние правил в общий Markdown файл по маркерам
func MergeMarkdownConfig(existingContent string, ruleContent string) string {
	const startMarker = "<!-- >>> agentsync managed >>> -->"
	const endMarker = "<!-- <<< agentsync managed <<< -->"

	normalizedExisting := strings.ReplaceAll(existingContent, "\r\n", "\n")
	normalizedNew := strings.ReplaceAll(ruleContent, "\r\n", "\n")

	startIdx := strings.Index(normalizedExisting, startMarker)
	endIdx := strings.Index(normalizedExisting, endMarker)

	managedBlock := fmt.Sprintf("%s\n%s\n%s", startMarker, strings.TrimSpace(normalizedNew), endMarker)

	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		// Маркеров нет, добавляем в конец файла
		if len(normalizedExisting) == 0 {
			return managedBlock
		}
		if strings.HasSuffix(normalizedExisting, "\n") {
			return normalizedExisting + managedBlock + "\n"
		}
		return normalizedExisting + "\n\n" + managedBlock + "\n"
	}

	// Заменяем блок между маркерами
	before := normalizedExisting[:startIdx]
	after := normalizedExisting[endIdx+len(endMarker):]

	// Убираем лишние переносы строк на стыках
	before = strings.TrimSuffix(before, "\n")
	after = strings.TrimPrefix(after, "\n")

	var result strings.Builder
	result.WriteString(before)
	if len(before) > 0 {
		result.WriteString("\n\n")
	}
	result.WriteString(managedBlock)
	if len(after) > 0 {
		result.WriteString("\n\n")
		result.WriteString(after)
	} else {
		result.WriteString("\n")
	}

	return result.String()
}

// InjectSecrets заменяет ${secrets.VAR} реальными значениями
func InjectSecrets(val string, secrets map[string]string) string {
	return secretPlaceholderRegex.ReplaceAllStringFunc(val, func(match string) string {
		submatches := secretPlaceholderRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		secretKey := submatches[1]
		if secretVal, ok := secrets[secretKey]; ok {
			return secretVal
		}
		// Если секрета нет, оставляем плейсхолдер
		return match
	})
}
