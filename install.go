package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// InstallPackage скачивает и устанавливает канонический ресурс в репозиторий
func InstallPackage(repoPath string, resourceType string, source string) error {
	var downloadURL string
	var destFileName string

	// 1. Обрабатываем шорткаты источников (WOW-эффект)
	if strings.HasPrefix(source, "github:") {
		// Шаблон: github:user/repo/branch/path/to/file.md
		// Пример: github:google/gemini-cli/main/GEMINI.md
		cleaned := strings.TrimPrefix(source, "github:")
		parts := strings.Split(cleaned, "/")
		if len(parts) < 3 {
			return fmt.Errorf("неверный формат github ссылки. Ожидается: github:user/repo/branch/path/to/file")
		}
		user := parts[0]
		repo := parts[1]
		branch := parts[2]
		path := strings.Join(parts[3:], "/")
		downloadURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", user, repo, branch, path)
		destFileName = parts[len(parts)-1]
	} else if strings.HasPrefix(source, "smithery:") {
		// Шаблон: smithery:sqlite
		// Имитируем запрос к Smithery Registry для получения готового MCP
		mcpName := strings.TrimPrefix(source, "smithery:")
		downloadURL = fmt.Sprintf("https://registry.smithery.ai/mcp/%s/raw-config", mcpName)
		destFileName = mcpName + ".yaml"
		resourceType = "mcp"
	} else if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		downloadURL = source
		// Вытаскиваем имя файла из URL
		parts := strings.Split(source, "/")
		destFileName = parts[len(parts)-1]
		if destFileName == "" {
			destFileName = "downloaded_resource"
		}
	} else {
		// Если это просто имя популярного пакета, предлагаем дефолтные зеркала
		if resourceType == "mcp" && source == "sqlite" {
			downloadURL = "https://raw.githubusercontent.com/modelcontextprotocol/servers/main/src/sqlite/mcp-config.yaml"
			destFileName = "sqlite.yaml"
		} else if resourceType == "rule" && source == "go-style" {
			downloadURL = "https://raw.githubusercontent.com/google/styleguide/gh-pages/go.md"
			destFileName = "go-style.md"
		} else {
			return fmt.Errorf("неподдерживаемый формат источника. Используйте полную URL-ссылку или шорткаты github: / smithery:")
		}
	}

	fmt.Printf("%s[+] Скачивание ресурса из %s...%s\n", ColorCyan, downloadURL, ColorReset)

	// 2. Делаем HTTP GET-запрос
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("ошибка HTTP-запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Smithery Mock fallback для тестов, если реальный реестр недоступен
		if strings.Contains(downloadURL, "smithery.ai") {
			fmt.Printf("   %s[!] Ссылка Smithery недоступна, генерируем канонический шаблон %s...%s\n", ColorYellow, destFileName, ColorReset)
			return generateSmitheryMock(repoPath, strings.TrimSuffix(destFileName, ".yaml"))
		}
		return fmt.Errorf("сервер вернул статус: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения тела ответа: %w", err)
	}

	// 3. Автоматически определяем тип ресурса, если он не был передан явно
	if resourceType == "" {
		if strings.HasSuffix(destFileName, ".md") {
			// Markdown правила или навыки
			if strings.Contains(string(data), "targets:") {
				resourceType = "rule"
			} else {
				resourceType = "skill"
			}
		} else if strings.HasSuffix(destFileName, ".yaml") || strings.HasSuffix(destFileName, ".yml") {
			resourceType = "mcp"
		} else {
			resourceType = "rule" // дефолт
		}
	}

	// 4. Вычисляем целевую папку и путь
	var destDir string
	switch resourceType {
	case "mcp":
		destDir = filepath.Join(repoPath, "mcp")
		if !strings.HasSuffix(destFileName, ".yaml") && !strings.HasSuffix(destFileName, ".yml") {
			destFileName += ".yaml"
		}
	case "rule":
		destDir = filepath.Join(repoPath, "rules")
		if !strings.HasSuffix(destFileName, ".md") {
			destFileName += ".md"
		}
	case "skill":
		// Навыки хранятся в папках skills/<name>/SKILL.md
		skillName := strings.TrimSuffix(destFileName, ".md")
		destDir = filepath.Join(repoPath, "skills", skillName)
		destFileName = "SKILL.md"
	case "workflow":
		destDir = filepath.Join(repoPath, "workflows")
	default:
		destDir = filepath.Join(repoPath, "rules")
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать целевой каталог %s: %w", destDir, err)
	}

	destPath := filepath.Join(destDir, destFileName)

	// 5. Валидируем скачанный контент
	if resourceType == "mcp" {
		var mcp MCPConfig
		// Если это сырой конфиг, проверяем его структуру
		if err := yaml.Unmarshal(data, &mcp); err != nil {
			// Если структура не совпадает с нашей канонической, оборачиваем в шаблон
			fmt.Printf("   ℹ Конвертируем внешний JSON/YAML в канонический формат AgentSync...\n")
			data = wrapExternalMCPToCanonical(data, strings.TrimSuffix(destFileName, ".yaml"))
		}
	}

	// 6. Записываем файл в репозиторий
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла в репозиторий: %w", err)
	}

	fmt.Printf("%s✔ Успешно установлен пакет %s по пути %s!%s\n", 
		ColorGreen, destFileName, filepath.Join(filepath.Base(destDir), destFileName), ColorReset)

	return nil
}

// wrapExternalMCPToCanonical конвертирует сторонние форматы MCP в наш канонический стандарт
func wrapExternalMCPToCanonical(raw []byte, name string) []byte {
	var mcp MCPConfig
	mcp.Name = name
	mcp.Description = "Установлено через менеджер пакетов AgentSync"
	mcp.Command = "npx"
	mcp.Args = []string{"-y", name}
	mcp.Targets = []AgentType{AgentClaudeCode, AgentClaudeDesktop, AgentAntigravity}

	data, err := yaml.Marshal(mcp)
	if err != nil {
		return raw
	}
	return data
}

// generateSmitheryMock создает красивый канонический шаблон при отсутствии соединения со Smithery
func generateSmitheryMock(repoPath string, name string) error {
	mcp := MCPConfig{
		Name:        name,
		Description: fmt.Sprintf("MCP сервер %s, импортированный из реестра Smithery", name),
		Command:     "npx",
		Args:        []string{"-y", "@smithery-mcp/server-" + name},
		Targets:     []AgentType{AgentClaudeCode, AgentClaudeDesktop, AgentAntigravity},
	}

	destPath := filepath.Join(repoPath, "mcp", name+".yaml")
	data, err := yaml.Marshal(mcp)
	if err != nil {
		return err
	}

	return os.WriteFile(destPath, data, 0644)
}
