package agent

import (
	"agentsync/internal/domain"
	"agentsync/internal/repository"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var _ AgentAdapter = (*ClaudeDesktopAdapter)(nil)

type ClaudeDesktopAdapter struct {
	homeDir string
	appData string
}

func NewClaudeDesktopAdapter() *ClaudeDesktopAdapter {
	home, _ := os.UserHomeDir()
	appData := os.Getenv("APPDATA")
	return &ClaudeDesktopAdapter{homeDir: home, appData: appData}
}

func (a *ClaudeDesktopAdapter) Name() domain.AgentType {
	return domain.AgentClaudeDesktop
}

func (a *ClaudeDesktopAdapter) Detect() bool {
	configPath := ResolveClaudeDesktopPath()
	_, err := os.Stat(configPath)
	return err == nil
}

func (a *ClaudeDesktopAdapter) DeployMCP(mcp domain.MCPConfig, secrets map[string]string) error {
	configPath := ResolveClaudeDesktopPath()
	return deployMCPToPath(configPath, a.Name(), mcp, secrets)
}

func (a *ClaudeDesktopAdapter) ClearMCPs() error {
	configPath := ResolveClaudeDesktopPath()
	return clearMCPsInPath(configPath, a.Name())
}

func (a *ClaudeDesktopAdapter) DeployRule(rule domain.Rule) error {
	// Claude Desktop не поддерживает нативные Markdown-правила
	return nil
}

// ResolveClaudeDesktopPath находит путь к конфигу Claude Desktop
func ResolveClaudeDesktopPath() string {
	home, _ := os.UserHomeDir()
	appData := os.Getenv("APPDATA")
	localAppData := os.Getenv("LOCALAPPDATA")

	// 1. Виртуализированный путь Microsoft Store (UWP/MSIX пакеты) - ПРИОРИТЕТ 1
	packagesDir := filepath.Join(localAppData, "Packages")
	if entries, err := os.ReadDir(packagesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(strings.ToLower(entry.Name()), "claude_") {
				storePath := filepath.Join(packagesDir, entry.Name(), "LocalCache", "Roaming", "Claude", "claude_desktop_config.json")
				if adapterFileExists(storePath) {
					return storePath
				}
			}
		}
	}

	// 2. Альтернативный виртуализированный путь Microsoft Store - ПРИОРИТЕТ 2
	packagesDirAlt := filepath.Join(home, "AppData", "Local", "Packages")
	if entries, err := os.ReadDir(packagesDirAlt); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(strings.ToLower(entry.Name()), "claude_") {
				storePath := filepath.Join(packagesDirAlt, entry.Name(), "LocalCache", "Roaming", "Claude", "claude_desktop_config.json")
				if adapterFileExists(storePath) {
					return storePath
				}
			}
		}
	}

	// 3. Стандартный Win32 путь - ПРИОРИТЕТ 3
	standardPath := filepath.Join(appData, "Claude", "claude_desktop_config.json")
	if adapterFileExists(standardPath) {
		return standardPath
	}

	return standardPath
}

// Вспомогательные функции деплоя для переиспользования адаптерами

func deployMCPToPath(path string, agent domain.AgentType, mcp domain.MCPConfig, secrets map[string]string) error {
	fixWindowsNpxCommand(&mcp)
	_ = repository.CreateBackup(path, agent) // Игнорируем ошибку бэкапа, продолжаем деплой

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ошибка создания каталога %s: %w", dir, err)
	}

	var existingJSON []byte
	if _, err := os.Stat(path); err == nil {
		var errRead error
		existingJSON, errRead = os.ReadFile(path)
		if errRead != nil {
			return fmt.Errorf("ошибка чтения %s: %w", path, errRead)
		}
	}

	updatedJSON, err := repository.MergeJSONConfig(existingJSON, mcp, secrets)
	if err != nil {
		return fmt.Errorf("ошибка слияния JSON: %w", err)
	}

	if err := os.WriteFile(path, updatedJSON, 0644); err != nil {
		return fmt.Errorf("ошибка записи в %s: %w", path, err)
	}

	return nil
}

func clearMCPsInPath(path string, agent domain.AgentType) error {
	_ = repository.CreateBackup(path, agent)

	if !adapterFileExists(path) {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(data, &configMap); err != nil {
		configMap = make(map[string]interface{})
	}

	// Очищаем mcpServers
	configMap["mcpServers"] = make(map[string]interface{})

	updatedJSON, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, updatedJSON, 0644)
}

func deployRuleToPath(path string, agent domain.AgentType, rule domain.Rule) error {
	_ = repository.CreateBackup(path, agent)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ошибка создания каталога %s: %w", dir, err)
	}

	var existingContent string
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("ошибка чтения %s: %w", path, err)
		}
		existingContent = string(data)
	}

	var ruleContent string
	if rule.Header.Name == "Combined Rules" {
		ruleContent = rule.Content
	} else {
		ruleContent = fmt.Sprintf("# Rule: %s\n%s", rule.Header.Name, rule.Content)
	}
	updatedContent := repository.MergeMarkdownConfig(existingContent, ruleContent)

	if err := os.WriteFile(path, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("ошибка записи в %s: %w", path, err)
	}

	return nil
}

// fixWindowsNpxCommand адаптирует запуск "npx" на Windows, переписывая команду на "cmd.exe /c npx ..."
func fixWindowsNpxCommand(mcp *domain.MCPConfig) {
	if runtime.GOOS != "windows" {
		return
	}

	if mcp.Command == "npx" {
		mcp.Command = "cmd.exe"
		newArgs := make([]string, 0, len(mcp.Args)+2)
		newArgs = append(newArgs, "/c", "npx")
		newArgs = append(newArgs, mcp.Args...)
		mcp.Args = newArgs
	}
}

func adapterFileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
