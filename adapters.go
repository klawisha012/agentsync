package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// BaseAdapter содержит общий вспомогательный функционал для всех адаптеров
type BaseAdapter struct {
	homeDir string
	appData string
}

func NewBaseAdapter() BaseAdapter {
	home, _ := os.UserHomeDir()
	appData := os.Getenv("APPDATA")
	return BaseAdapter{homeDir: home, appData: appData}
}

// -------------------------------------------------------------
// Claude Code Адаптер
// -------------------------------------------------------------
type ClaudeCodeAdapter struct {
	BaseAdapter
}

func NewClaudeCodeAdapter() *ClaudeCodeAdapter {
	return &ClaudeCodeAdapter{BaseAdapter: NewBaseAdapter()}
}

func (a *ClaudeCodeAdapter) Name() AgentType {
	return AgentClaudeCode
}

func (a *ClaudeCodeAdapter) Detect() bool {
	configPath := filepath.Join(a.homeDir, ".claude.json")
	_, err := os.Stat(configPath)
	return err == nil
}

func (a *ClaudeCodeAdapter) DeployMCP(mcp MCPConfig, secrets map[string]string) error {
	configPath := filepath.Join(a.homeDir, ".claude.json")
	return a.deployMCPToPath(configPath, a.Name(), mcp, secrets)
}

func (a *ClaudeCodeAdapter) ClearMCPs() error {
	configPath := filepath.Join(a.homeDir, ".claude.json")
	return a.clearMCPsInPath(configPath, a.Name())
}

func (a *ClaudeCodeAdapter) DeployRule(rule Rule) error {
	rulesPath := filepath.Join(a.homeDir, ".claude", "CLAUDE.md")
	return a.deployRuleToPath(rulesPath, a.Name(), rule)
}

// Вспомогательный метод очистки MCP-серверов в JSON с созданием бэкапа
func (a *BaseAdapter) clearMCPsInPath(path string, agent AgentType) error {
	if err := CreateBackup(path, agent); err != nil {
		fmt.Printf("   %s[!] Предупреждение бэкапа: %v%s\n", ColorYellow, err, ColorReset)
	}

	if !fileExists(path) {
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

// Вспомогательный метод деплоя MCP в JSON с созданием бэкапа
func (a *BaseAdapter) deployMCPToPath(path string, agent AgentType, mcp MCPConfig, secrets map[string]string) error {
	fixWindowsNpxCommand(&mcp)
	if err := CreateBackup(path, agent); err != nil {
		fmt.Printf("   %s[!] Предупреждение бэкапа: %v%s\n", ColorYellow, err, ColorReset)
	}

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

	updatedJSON, err := MergeJSONConfig(existingJSON, mcp, secrets)
	if err != nil {
		return fmt.Errorf("ошибка слияния JSON: %w", err)
	}

	if err := os.WriteFile(path, updatedJSON, 0644); err != nil {
		return fmt.Errorf("ошибка записи в %s: %w", path, err)
	}

	return nil
}

// Вспомогательный метод деплоя правила в Markdown с созданием бэкапа
func (a *BaseAdapter) deployRuleToPath(path string, agent AgentType, rule Rule) error {
	if err := CreateBackup(path, agent); err != nil {
		fmt.Printf("   %s[!] Предупреждение бэкапа: %v%s\n", ColorYellow, err, ColorReset)
	}

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
	updatedContent := MergeMarkdownConfig(existingContent, ruleContent)

	if err := os.WriteFile(path, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("ошибка записи в %s: %w", path, err)
	}

	return nil
}

// -------------------------------------------------------------
// Claude Desktop Адаптер
// -------------------------------------------------------------
type ClaudeDesktopAdapter struct {
	BaseAdapter
}

func NewClaudeDesktopAdapter() *ClaudeDesktopAdapter {
	return &ClaudeDesktopAdapter{BaseAdapter: NewBaseAdapter()}
}

func (a *ClaudeDesktopAdapter) Name() AgentType {
	return AgentClaudeDesktop
}

func (a *ClaudeDesktopAdapter) Detect() bool {
	configPath := ResolveClaudeDesktopPath()
	_, err := os.Stat(configPath)
	return err == nil
}

func (a *ClaudeDesktopAdapter) DeployMCP(mcp MCPConfig, secrets map[string]string) error {
	configPath := ResolveClaudeDesktopPath()
	return a.deployMCPToPath(configPath, a.Name(), mcp, secrets)
}

func (a *ClaudeDesktopAdapter) ClearMCPs() error {
	configPath := ResolveClaudeDesktopPath()
	return a.clearMCPsInPath(configPath, a.Name())
}

func (a *ClaudeDesktopAdapter) DeployRule(rule Rule) error {
	return nil
}

// -------------------------------------------------------------
// Antigravity Адаптер
// -------------------------------------------------------------
type AntigravityAdapter struct {
	BaseAdapter
}

func NewAntigravityAdapter() *AntigravityAdapter {
	return &AntigravityAdapter{BaseAdapter: NewBaseAdapter()}
}

func (a *AntigravityAdapter) Name() AgentType {
	return AgentAntigravity
}

func (a *AntigravityAdapter) Detect() bool {
	configPath := filepath.Join(a.homeDir, ".gemini", "config", "mcp_config.json")
	_, err := os.Stat(configPath)
	return err == nil
}

func (a *AntigravityAdapter) DeployMCP(mcp MCPConfig, secrets map[string]string) error {
	configPath := filepath.Join(a.homeDir, ".gemini", "config", "mcp_config.json")
	return a.deployMCPToPath(configPath, a.Name(), mcp, secrets)
}

func (a *AntigravityAdapter) ClearMCPs() error {
	configPath := filepath.Join(a.homeDir, ".gemini", "config", "mcp_config.json")
	return a.clearMCPsInPath(configPath, a.Name())
}

func (a *AntigravityAdapter) DeployRule(rule Rule) error {
	rulesPath := filepath.Join(a.homeDir, ".gemini", "GEMINI.md")
	return a.deployRuleToPath(rulesPath, a.Name(), rule)
}

// fixWindowsNpxCommand адаптирует запуск "npx" на Windows, переписывая команду на "cmd.exe /c npx ..."
// Это позволяет обойти баг NodeJS, который пытается разрешать npx.cmd до полного пути с пробелами,
// что ломает запуск из-за некорректного экранирования.
func fixWindowsNpxCommand(mcp *MCPConfig) {
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
