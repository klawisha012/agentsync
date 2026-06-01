package service

import (
	"agentsync/internal/agent"
	"agentsync/internal/domain"
	"agentsync/internal/repository"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type DeployService struct {
	repoPath string
	logger   Logger
}

func NewDeployService(repoPath string, logger Logger) *DeployService {
	return &DeployService{repoPath: repoPath, logger: logger}
}

func (s *DeployService) log(msg string, level string) {
	if s.logger != nil {
		s.logger.Log(msg, level)
	}
}

// Deploy запускает слияние и развертывание канонических настроек по всем активным агентам
func (s *DeployService) Deploy(activeBundleId string) error {
	s.log("Запуск Smart Merge деплоя...", "info")

	// Импортируем существующие локальные серверы
	_ = s.SmartImportMCPs()

	warnings, err := repository.AuditRepository(s.repoPath)
	if err == nil && len(warnings) > 0 {
		s.log("ВНИМАНИЕ: Обнаружены потенциальные утечки сырых секретов перед деплоем:", "warn")
		for _, w := range warnings {
			s.log("  "+w, "warn")
		}
	}

	store := repository.NewStore(s.repoPath)

	manifest, err := store.LoadManifest()
	if err != nil {
		return fmt.Errorf("ошибка загрузки манифеста: %w", err)
	}

	mcps, err := store.LoadMCPs()
	if err != nil {
		return fmt.Errorf("ошибка загрузки MCP-серверов: %w", err)
	}

	rules, err := store.LoadRules()
	if err != nil {
		return fmt.Errorf("ошибка загрузки правил: %w", err)
	}

	secrets, err := repository.LoadSecrets(s.repoPath)
	if err != nil {
		s.log(fmt.Sprintf("Предупреждение загрузки секретов: %v", err), "warn")
	}

	// Загружаем все доступные бандлы конфигурации
	bundles, err := store.LoadBundles()
	if err != nil {
		return fmt.Errorf("ошибка загрузки конфигураций бандлов: %w", err)
	}

	// Загружаем активный бандл конфигурации
	if activeBundleId == "" {
		activeBundleId = manifest.ActiveBundle
	}

	var activeBundle *domain.ConfigBundle
	if activeBundleId != "" {
		for i := range bundles {
			if bundles[i].ID == activeBundleId {
				activeBundle = &bundles[i]
				break
			}
		}
	}

	// Если активный бандл не найден или не задан, пробуем взять первый из доступных
	if activeBundle == nil && len(bundles) > 0 {
		activeBundle = &bundles[0]
		activeBundleId = activeBundle.ID
		manifest.ActiveBundle = activeBundleId
		if err := store.SaveManifest(manifest); err == nil {
			s.log(fmt.Sprintf("Автоматически выбран активный бандл: %s (%s)", activeBundle.Name, activeBundleId), "info")
		}
	}

	if activeBundle == nil {
		return fmt.Errorf("отсутствует активный бандл (конфигурация) для деплоя. Пожалуйста, создайте бандл в Agent Manager")
	}

	s.log(fmt.Sprintf("Деплой на основе активной конфигурации: %s (%s)...", activeBundle.Name, activeBundleId), "info")

	// Инициализируем менеджер изолированных окружений
	runtimeMgr, err := NewRuntimeManager(s.logger)
	if err != nil {
		return fmt.Errorf("ошибка менеджера изолированных окружений: %w", err)
	}

	adapters := []agent.AgentAdapter{
		agent.NewClaudeCodeAdapter(),
		agent.NewClaudeDesktopAdapter(),
		agent.NewAntigravityAdapter(),
	}

	deployedCount := 0

	for _, activeAgent := range manifest.ActiveAgents {
		var targetAdapter agent.AgentAdapter
		for _, a := range adapters {
			if a.Name() == activeAgent {
				targetAdapter = a
				break
			}
		}

		if targetAdapter == nil {
			continue
		}

		if !targetAdapter.Detect() {
			s.log(fmt.Sprintf("Пропуск %s (агент не установлен на этой машине)", activeAgent), "info")
			continue
		}

		s.log(fmt.Sprintf("Деплой для %s...", activeAgent), "info")

		// Очищаем старые MCP-серверы перед деплоем бандла
		if err := targetAdapter.ClearMCPs(); err != nil {
			s.log(fmt.Sprintf("   Предупреждение очистки MCP: %v", err), "warn")
		}

		for _, mcp := range mcps {
			inBundle := false
			for _, item := range activeBundle.Components {
				if item.Type == domain.ComponentMCP && item.Name == mcp.Name {
					inBundle = true
					break
				}
			}

			if inBundle {
				mcpCopy := mcp

				// Разрешаем изолированное окружение (Node.js/Python), если требуется
				if mcp.Runtime != "" || mcp.Package != "" {
					s.log(fmt.Sprintf("   -> Изоляция окружения для MCP: %s...", mcp.Name), "info")
					exePath, err := runtimeMgr.ResolveRuntimePath(mcp.Runtime)
					if err != nil {
						s.log(fmt.Sprintf("      Ошибка рантайма: %v", err), "error")
						continue
					}

					if exePath != "" {
						packagePath, err := runtimeMgr.InstallMCPPackage(exePath, mcp.Package)
						if err != nil {
							s.log(fmt.Sprintf("      Ошибка пакета: %v", err), "error")
							continue
						}

						if packagePath != "" {
							mcpCopy.Command = exePath
							// Определяем точку входа js-файла
							var entryPoint string
							if deployFileExists(filepath.Join(packagePath, "dist", "index.js")) {
								entryPoint = filepath.Join(packagePath, "dist", "index.js")
							} else if deployFileExists(filepath.Join(packagePath, "index.js")) {
								entryPoint = filepath.Join(packagePath, "index.js")
							} else {
								entryPoint = filepath.Join(packagePath, "dist", "index.js")
							}
							mcpCopy.Args = []string{entryPoint}
						}
					}
				}

				// 1. Автоматическая фильтрация SSE/HTTP транспортов для CLI-агентов (Claude Desktop / Claude Code)
				if mcpCopy.ServerURL != "" && (activeAgent == domain.AgentClaudeDesktop || activeAgent == domain.AgentClaudeCode) {
					s.log(fmt.Sprintf("   -> Пропуск MCP: %s (SSE/HTTP не поддерживается агентом %s)", mcpCopy.Name, activeAgent), "info")
					continue
				}

				// 2. Фильтрация на основе уже установленных плагинов/инструментов из manifest.yaml
				isInstalledPlugin := false
				if manifest.InstalledPlugins != nil {
					if plugins, ok := manifest.InstalledPlugins[activeAgent]; ok {
						for _, p := range plugins {
							if strings.EqualFold(p, mcpCopy.Name) {
								isInstalledPlugin = true
								break
							}
						}
					}
				}
				if isInstalledPlugin {
					s.log(fmt.Sprintf("   -> Пропуск MCP: %s (уже установлен как плагин/инструмент в %s)", mcpCopy.Name, activeAgent), "info")
					continue
				}

				s.log(fmt.Sprintf("   -> Слияние MCP: %s... ", mcpCopy.Name), "info")
				if err := targetAdapter.DeployMCP(mcpCopy, secrets); err != nil {
					s.log(fmt.Sprintf("ОШИБКА: %v", err), "error")
				} else {
					s.log("УСПЕШНО", "info")
					deployedCount++
				}
			}
		}

		// Объединяем и деплоим правила
		var combinedRules []domain.Rule
		for _, rule := range rules {
			inBundle := false
			for _, item := range activeBundle.Components {
				if item.Type == domain.ComponentRule && item.Name == rule.Header.Name {
					inBundle = true
					break
				}
			}
			if inBundle {
				combinedRules = append(combinedRules, rule)
			}
		}

		var ruleContent string
		if len(combinedRules) > 0 {
			var sb strings.Builder
			for i, r := range combinedRules {
				if i > 0 {
					sb.WriteString("\n\n")
				}
				sb.WriteString(fmt.Sprintf("# Rule: %s\n%s", r.Header.Name, r.Content))
			}
			ruleContent = sb.String()
		}

		combinedRule := domain.Rule{
			Header: domain.RuleFrontmatter{
				Name: "Combined Rules",
			},
			Content: ruleContent,
		}

		s.log(fmt.Sprintf("   -> Синхронизация правил бандла (%d шт.)... ", len(combinedRules)), "info")
		if err := targetAdapter.DeployRule(combinedRule); err != nil {
			s.log(fmt.Sprintf("ОШИБКА: %v", err), "error")
		} else {
			s.log("УСПЕШНО", "info")
			deployedCount++
		}
	}

	s.log(fmt.Sprintf("Деплой успешно завершен. Применено операций слияния: %d", deployedCount), "info")
	return nil
}

// SmartImportMCPs импортирует существующие локальные MCP-серверы из конфигов агентов в репозиторий
func (s *DeployService) SmartImportMCPs() error {
	mcpDir := filepath.Join(s.repoPath, "data", "mcp")
	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		return err
	}

	home, _ := os.UserHomeDir()

	targets := []struct {
		AgentName domain.AgentType
		Path      string
	}{
		{domain.AgentClaudeCode, filepath.Join(home, ".claude.json")},
		{domain.AgentClaudeDesktop, agent.ResolveClaudeDesktopPath()},
		{domain.AgentAntigravity, filepath.Join(home, ".gemini", "config", "mcp_config.json")},
	}

	for _, t := range targets {
		if !deployFileExists(t.Path) {
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

			canonPath := filepath.Join(mcpDir, name+".yaml")
			
			var mcp domain.MCPConfig
			if deployFileExists(canonPath) {
				canonData, err := os.ReadFile(canonPath)
				if err == nil {
					_ = yaml.Unmarshal(canonData, &mcp)
				}
			}

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

			yamlData, err := yaml.Marshal(mcp)
			if err == nil {
				_ = os.WriteFile(canonPath, yamlData, 0644)
			}
		}
	}

	return nil
}

func deployFileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
