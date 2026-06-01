package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ResolveClaudeDesktopPath эвристически находит путь к конфигу Claude Desktop на Windows
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
				if fileExists(storePath) {
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
				if fileExists(storePath) {
					return storePath
				}
			}
		}
	}

	// 3. Стандартный Win32 путь - ПРИОРИТЕТ 3
	standardPath := filepath.Join(appData, "Claude", "claude_desktop_config.json")
	if fileExists(standardPath) {
		return standardPath
	}

	return standardPath
}

// SmartImportMCPs импортирует существующие локальные MCP-серверы из конфигов агентов в репозиторий
func SmartImportMCPs(repoPath string) error {
	mcpDir := filepath.Join(repoPath, "mcp")
	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		return err
	}

	home, _ := os.UserHomeDir()

	// Описываем, откуда сканируем
	targets := []struct {
		AgentName AgentType
		Path      string
	}{
		{AgentClaudeCode, filepath.Join(home, ".claude.json")},
		{AgentClaudeDesktop, ResolveClaudeDesktopPath()},
		{AgentAntigravity, filepath.Join(home, ".gemini", "config", "mcp_config.json")},
	}

	for _, t := range targets {
		if !fileExists(t.Path) {
			continue
		}

		data, err := os.ReadFile(t.Path)
		if err != nil {
			continue
		}

		var configMap map[string]interface{}
		if err := json.Unmarshal(data, &configMap); err != nil {
			continue
		}

		mcpServersRaw, exists := configMap["mcpServers"]
		if !exists {
			continue
		}

		mcpServers, ok := mcpServersRaw.(map[string]interface{})
		if !ok {
			continue
		}

		for name, settingsRaw := range mcpServers {
			settings, ok := settingsRaw.(map[string]interface{})
			if !ok {
				continue
			}

			// Если в репозитории уже есть канонический файл для этого сервера, то просто добавляем таргет
			canonPath := filepath.Join(mcpDir, name+".yaml")
			
			var mcp MCPConfig
			if fileExists(canonPath) {
				canonData, err := os.ReadFile(canonPath)
				if err == nil {
					_ = yaml.Unmarshal(canonData, &mcp)
				}
			}

			// Заполняем поля, если файла не было
			if mcp.Name == "" {
				mcp.Name = name
				mcp.Description = fmt.Sprintf("Импортировано из локальной конфигурации %s", t.AgentName)
				
				if cmd, ok := settings["command"].(string); ok {
					mcp.Command = cmd
				}

				if argsRaw, ok := settings["args"].([]interface{}); ok {
					mcp.Args = []string{}
					for _, arg := range argsRaw {
						if str, ok := arg.(string); ok {
							mcp.Args = append(mcp.Args, str)
						} else {
							mcp.Args = append(mcp.Args, fmt.Sprintf("%v", arg))
						}
					}
				} else if argsRaw, ok := settings["args"].([]string); ok {
					mcp.Args = argsRaw
				}

				if envRaw, ok := settings["env"].(map[string]interface{}); ok {
					mcp.Env = make(map[string]string)
					for k, v := range envRaw {
						mcp.Env[k] = fmt.Sprintf("%v", v)
					}
				}

				if serverURL, ok := settings["serverUrl"].(string); ok {
					mcp.ServerURL = serverURL
				}

				if headersRaw, ok := settings["headers"].(map[string]interface{}); ok {
					mcp.Headers = make(map[string]string)
					for k, v := range headersRaw {
						mcp.Headers[k] = fmt.Sprintf("%v", v)
					}
				}
			}

			// Добавляем агента в Targets, если его там нет
			hasTarget := false
			for _, target := range mcp.Targets {
				if target == t.AgentName {
					hasTarget = true
					break
				}
			}
			if !hasTarget {
				mcp.Targets = append(mcp.Targets, t.AgentName)
			}

			// Записываем обратно в YAML
			yamlData, err := yaml.Marshal(mcp)
			if err == nil {
				_ = os.WriteFile(canonPath, yamlData, 0644)
			}
		}
	}

	return nil
}
