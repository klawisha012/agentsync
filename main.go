package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ANSI цвета для красивого CLI вывода
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorMagenta= "\033[35m"
	ColorCyan   = "\033[36m"
	ColorBold   = "\033[1m"
)

func main() {
	if len(os.Args) < 2 {
		if err := StartTUI(); err != nil {
			fmt.Printf("[-] Ошибка запуска TUI: %v\n", err)
			printWelcome()
		}
		return
	}

	command := os.Args[1]

	switch command {
	case "init":
		runInit()
	case "deploy":
		runDeploy("")
	case "install":
		runInstall()
	case "share":
		runShare()
	case "pull-from":
		runPullFrom()
	case "doctor":
		discoverCmd := flag.NewFlagSet("doctor", flag.ExitOnError)
		discoverOnly := discoverCmd.Bool("discover", false, "Запустить автообнаружение установленных AI-агентов")
		_ = discoverCmd.Parse(os.Args[2:])
		if *discoverOnly {
			runDiscover()
		} else {
			runDoctorDiagnostics()
		}
	case "tui":
		if err := StartTUI(); err != nil {
			fmt.Printf("[-] Ошибка запуска TUI: %v\n", err)
		}
	case "web", "gui":
		cwd, _ := os.Getwd()
		err := StartWebServer(8080, cwd)
		if err != nil {
			fmt.Printf("[-] Ошибка запуска веб-сервера: %v\n", err)
		}
	default:
		fmt.Printf("%s%sНеизвестная команда: %s%s\n", ColorRed, ColorBold, os.Args[1], ColorReset)
		printUsage()
	}
}

func printWelcome() {
	fmt.Printf("%s%s=== AgentSync CLI v0.1.0 ===%s\n", ColorCyan, ColorBold, ColorReset)
	fmt.Println("Ультимативный инструмент синхронизации конфигураций AI-агентов")
	fmt.Println()
	printUsage()
}

func printUsage() {
	fmt.Println("Доступные команды:")
	fmt.Println("  agentsync                 Запустить интерактивный TUI дашборд (по умолчанию)")
	fmt.Println("  agentsync init            Инициализировать канонический репозиторий в текущей папке")
	fmt.Println("  agentsync deploy          Раскрыть канонические настройки по всем активным агентам")
	fmt.Println("  agentsync install [type] <src> Скачать и установить MCP/rule/skill из внешних хабов")
	fmt.Println("  agentsync share           Запустить P2P AirDrop раздачу текущей конфигурации")
	fmt.Println("  agentsync pull-from <host> --pin <PIN> Скачать и применить конфигурацию друга по P2P")
	fmt.Println("  agentsync doctor          Запустить полную диагностику системы и аудит безопасности")
	fmt.Println("  agentsync doctor --discover Автообнаружение установленных агентов и путей")
	fmt.Println("  agentsync tui             Запустить стильный интерактивный TUI дашборд")
	fmt.Println("  agentsync web             Запустить интерактивный Web GUI дашборд в браузере")
}

func runInit() {
	fmt.Printf("%s%s[+] Инициализация канонического репозитория AgentSync...%s\n", ColorBlue, ColorBold, ColorReset)

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("%s[-] Ошибка получения текущей директории: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	// 1. Создаем папки mcp/ и rules/
	mcpDir := filepath.Join(cwd, "mcp")
	rulesDir := filepath.Join(cwd, "rules")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		fmt.Printf("%s[-] Ошибка создания папки mcp: %v%s\n", ColorRed, err, ColorReset)
		return
	}
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		fmt.Printf("%s[-] Ошибка создания папки rules: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	// 2. Создаем manifest.yaml
	manifestPath := filepath.Join(cwd, "manifest.yaml")
	manifestContent := `active_agents:
  - claude-code
  - claude-desktop
  - antigravity
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		fmt.Printf("%s[-] Ошибка записи manifest.yaml: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	// 3. Создаем дефолтный secrets.env
	secretsPath := filepath.Join(cwd, "secrets.env")
	secretsContent := `# Локальные секреты для AgentSync (НЕ коммитить в Git)
# Шаблон: ИМЯ_СЕКРЕТА=значение
GITHUB_TOKEN=ghp_exampleTokenValue1234567890abcdef
`
	if err := os.WriteFile(secretsPath, []byte(secretsContent), 0644); err != nil {
		fmt.Printf("%s[-] Ошибка записи secrets.env: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	// 4. Создаем тестовый MCP github.yaml
	githubMcpPath := filepath.Join(mcpDir, "github.yaml")
	githubMcpContent := `name: github
description: "MCP сервер интеграции с GitHub API"
runtime: nodejs@20
package: "@modelcontextprotocol/server-github"
env:
  GITHUB_PERSONAL_ACCESS_TOKEN: "${secrets.GITHUB_TOKEN}"
targets:
  - claude-code
  - claude-desktop
  - antigravity
`
	if err := os.WriteFile(githubMcpPath, []byte(githubMcpContent), 0644); err != nil {
		fmt.Printf("%s[-] Ошибка записи mcp/github.yaml: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	// 5. Создаем тестовое правило rules/react-style.md
	reactRulePath := filepath.Join(rulesDir, "react-style.md")
	reactRuleContent := `---
name: react-style
description: "Стандарты разработки React компонентов"
targets:
  - claude-code
  - antigravity
---
# Правила React разработки
1. Всегда используйте функциональные компоненты и хуки.
2. Предпочитайте CSS-модули для стилизации.
3. Проверяйте зависимости в useEffect.
`
	if err := os.WriteFile(reactRulePath, []byte(reactRuleContent), 0644); err != nil {
		fmt.Printf("%s[-] Ошибка записи rules/react-style.md: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	fmt.Printf("%s✔ Репозиторий успешно инициализирован в %s%s\n", ColorGreen, cwd, ColorReset)
	fmt.Println("Созданы manifest.yaml, secrets.env, mcp/github.yaml, rules/react-style.md")
	fmt.Println("Добавьте secrets.env в ваш .gitignore перед фиксацией репозитория!")

	fmt.Println("[+] Автоматический импорт существующих локальных MCP серверов...")
	_ = SmartImportMCPs(cwd)
}

func runDeploy(activeBundleId string) {
	fmt.Printf("%s%s[+] Запуск Smart Merge деплоя...%s\n", ColorBlue, ColorBold, ColorReset)

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("%s[-] Ошибка: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	// Импортируем существующие локальные серверы, чтобы они попали в канонический репозиторий
	_ = SmartImportMCPs(cwd)

	warnings, err := AuditRepository(cwd)
	if err == nil && len(warnings) > 0 {
		fmt.Printf("%s[!] ВНИМАНИЕ: Обнаружены потенциальные утечки сырых секретов перед деплоем:%s\n", ColorYellow, ColorReset)
		for _, w := range warnings {
			fmt.Println("  ", w)
		}
		fmt.Println()
	}

	store := NewStore(cwd)

	manifest, err := store.LoadManifest()
	if err != nil {
		fmt.Printf("%s[-] Ошибка загрузки манифеста: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	mcps, err := store.LoadMCPs()
	if err != nil {
		fmt.Printf("%s[-] Ошибка загрузки MCP-серверов: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	rules, err := store.LoadRules()
	if err != nil {
		fmt.Printf("%s[-] Ошибка загрузки правил: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	secrets, err := LoadSecrets(cwd)
	if err != nil {
		fmt.Printf("%s[-] Предупреждение загрузки секретов: %v%s\n", ColorYellow, err, ColorReset)
	}

	// Загружаем все доступные бандлы конфигурации
	bundles, err := store.LoadBundles()
	if err != nil {
		fmt.Printf("%s[-] Ошибка загрузки конфигураций бандлов: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	// Загружаем активный бандл конфигурации, если он выбран
	if activeBundleId == "" {
		activeBundleId = manifest.ActiveBundle
	}

	var activeBundle *ConfigBundle
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
			fmt.Printf("%s[!] Автоматически выбран активный бандл: %s (%s)%s\n", ColorYellow, activeBundle.Name, activeBundleId, ColorReset)
		}
	}

	if activeBundle == nil {
		fmt.Printf("%s[-] Ошибка: Отсутствует активный бандл (конфигурация) для деплоя. Пожалуйста, создайте бандл в Agent Manager.%s\n", ColorRed, ColorReset)
		return
	}

	fmt.Printf("%s%s[+] Деплой на основе активной конфигурации: %s (%s)...%s\n", ColorBlue, ColorBold, activeBundle.Name, activeBundleId, ColorReset)

	// Инициализируем менеджер изолированных окружений (Zero-Dependency)
	runtimeMgr, err := NewRuntimeManager()
	if err != nil {
		fmt.Printf("%s[-] Ошибка менеджера изолированных окружений: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	adapters := []Adapter{
		NewClaudeCodeAdapter(),
		NewClaudeDesktopAdapter(),
		NewAntigravityAdapter(),
	}

	deployedCount := 0

	for _, activeAgent := range manifest.ActiveAgents {
		var targetAdapter Adapter
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
			fmt.Printf("ℹ Пропуск %s (агент не установлен на этой машине)\n", activeAgent)
			continue
		}

		fmt.Printf("%s%s[+] Деплой для %s...%s\n", ColorMagenta, ColorBold, activeAgent, ColorReset)

		// Очищаем старые MCP-серверы перед деплоем бандла
		if err := targetAdapter.ClearMCPs(); err != nil {
			fmt.Printf("   %s[-] Предупреждение очистки MCP: %v%s\n", ColorYellow, err, ColorReset)
		}

		for _, mcp := range mcps {
			inBundle := false
			for _, item := range activeBundle.Components {
				if item.Type == ComponentMCP && item.Name == mcp.Name {
					inBundle = true
					break
				}
			}

			if inBundle {
				mcpCopy := mcp

				// Разрешаем изолированное окружение (Node.js/Python), если требуется
				if mcp.Runtime != "" || mcp.Package != "" {
					fmt.Printf("   -> Изоляция окружения для MCP: %s...\n", mcp.Name)
					exePath, err := runtimeMgr.ResolveRuntimePath(mcp.Runtime)
					if err != nil {
						fmt.Printf("      %s[-] Ошибка рантайма: %v%s\n", ColorRed, err, ColorReset)
						continue
					}

					if exePath != "" {
						packagePath, err := runtimeMgr.InstallMCPPackage(exePath, mcp.Package)
						if err != nil {
							fmt.Printf("      %s[-] Ошибка пакета: %v%s\n", ColorRed, err, ColorReset)
							continue
						}

						if packagePath != "" {
							// Переопределяем команду на запуск через портативный node.exe
							mcpCopy.Command = exePath
							// Определяем точку входа js-файла
							var entryPoint string
							if fileExists(filepath.Join(packagePath, "dist", "index.js")) {
								entryPoint = filepath.Join(packagePath, "dist", "index.js")
							} else if fileExists(filepath.Join(packagePath, "index.js")) {
								entryPoint = filepath.Join(packagePath, "index.js")
							} else {
								entryPoint = filepath.Join(packagePath, "dist", "index.js")
							}
							mcpCopy.Args = []string{entryPoint}
						}
					}
				}

				// 1. Автоматическая фильтрация SSE/HTTP транспортов для CLI-агентов (Claude Desktop / Claude Code)
				if mcpCopy.ServerURL != "" && (activeAgent == AgentClaudeDesktop || activeAgent == AgentClaudeCode) {
					fmt.Printf("   -> Пропуск MCP: %s (SSE/HTTP не поддерживается агентом %s)\n", mcpCopy.Name, activeAgent)
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
					fmt.Printf("   -> Пропуск MCP: %s (уже установлен как плагин/инструмент в %s)\n", mcpCopy.Name, activeAgent)
					continue
				}

				fmt.Printf("   -> Слияние MCP: %s... ", mcpCopy.Name)
				if err := targetAdapter.DeployMCP(mcpCopy, secrets); err != nil {
					fmt.Printf("%sОШИБКА: %v%s\n", ColorRed, err, ColorReset)
				} else {
					fmt.Printf("%sУСПЕШНО%s\n", ColorGreen, ColorReset)
					deployedCount++
				}
			}
		}

		// Объединяем и деплоим правила
		var combinedRules []Rule
		for _, rule := range rules {
			inBundle := false
			for _, item := range activeBundle.Components {
				if item.Type == ComponentRule && item.Name == rule.Header.Name {
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

		combinedRule := Rule{
			Header: RuleFrontmatter{
				Name: "Combined Rules",
			},
			Content: ruleContent,
		}

		fmt.Printf("   -> Синхронизация правил бандла (%d шт.)... ", len(combinedRules))
		if err := targetAdapter.DeployRule(combinedRule); err != nil {
			fmt.Printf("%sОШИБКА: %v%s\n", ColorRed, err, ColorReset)
		} else {
			fmt.Printf("%sУСПЕШНО%s\n", ColorGreen, ColorReset)
			deployedCount++
		}
	}

	fmt.Printf("\n%s%s✔ Деплой успешно завершен. Применено операций слияния: %d%s\n", ColorGreen, ColorBold, deployedCount, ColorReset)
}

func runInstall() {
	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	typeFlag := installCmd.String("type", "", "Тип ресурса: mcp, rule, skill, workflow")
	_ = installCmd.Parse(os.Args[2:])

	args := installCmd.Args()
	if len(args) == 0 {
		fmt.Println("Использование: agentsync install [mcp|rule|skill|workflow] <источник_или_url>")
		fmt.Println("Примеры:")
		fmt.Println("  agentsync install mcp smithery:sqlite")
		fmt.Println("  agentsync install rule github:google/styleguide/gh-pages/go.md")
		fmt.Println("  agentsync install mcp https://raw.githubusercontent.com/.../server.yaml")
		return
	}

	var resourceType string
	var source string

	if len(args) >= 2 {
		resourceType = args[0]
		source = args[1]
	} else {
		source = args[0]
		// тип определится автоматически по расширению/структуре
	}

	if *typeFlag != "" {
		resourceType = *typeFlag
	}

	cwd, _ := os.Getwd()
	err := InstallPackage(cwd, resourceType, source)
	if err != nil {
		fmt.Printf("%s[-] Ошибка установки: %v%s\n", ColorRed, err, ColorReset)
	} else {
		runDeploy("")
	}
}

func runShare() {
	shareCmd := flag.NewFlagSet("share", flag.ExitOnError)
	pinFlag := shareCmd.String("pin", "7788", "Одноразовый PIN-код для авторизации друга")
	portFlag := shareCmd.Int("port", 0, "Порт для запуска сервера (0 = случайный свободный)")
	_ = shareCmd.Parse(os.Args[2:])

	cwd, _ := os.Getwd()

	err := P2PShare(cwd, *pinFlag, *portFlag)
	if err != nil {
		fmt.Printf("%s[-] Ошибка P2P раздачи: %v%s\n", ColorRed, err, ColorReset)
	}
}

func runPullFrom() {
	if len(os.Args) < 3 {
		fmt.Println("Использование: agentsync pull-from <host:port> --pin <PIN>")
		return
	}

	target := os.Args[2]

	pullCmd := flag.NewFlagSet("pull-from", flag.ExitOnError)
	pinFlag := pullCmd.String("pin", "", "PIN-код авторизации (обязательно)")
	_ = pullCmd.Parse(os.Args[3:])

	if *pinFlag == "" {
		fmt.Println("Ошибка: Необходимо передать --pin <PIN-код>")
		return
	}

	var host string
	port := 8080

	if strings.Contains(target, ":") {
		parts := strings.SplitN(target, ":", 2)
		host = parts[0]
		fmt.Sscanf(parts[1], "%d", &port)
	} else {
		host = target
	}

	cwd, _ := os.Getwd()

	err := P2PPull(host, port, *pinFlag, cwd)
	if err != nil {
		fmt.Printf("%s[-] Ошибка P2P импорта: %v%s\n", ColorRed, err, ColorReset)
	} else {
		runDeploy("")
	}
}

func isTarget(agent AgentType, targets []AgentType) bool {
	for _, t := range targets {
		if t == agent || t == "all" {
			return true
		}
	}
	return false
}

func runDiscover() {
	fmt.Printf("%s%s[+] Запуск автообнаружения AI-агентов...%s\n\n", ColorBlue, ColorBold, ColorReset)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("%s[-] Ошибка получения домашней директории: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	claudeCode := AgentInfo{
		Name: "Claude Code (CLI)",
		ConfigPaths: []string{
			filepath.Join(homeDir, ".claude.json"),
			filepath.Join(homeDir, ".claude", "settings.json"),
		},
		RulesPaths: []string{
			filepath.Join(homeDir, ".claude", "CLAUDE.md"),
		},
	}
	checkAgentExistence(&claudeCode)

	claudeDesktop := AgentInfo{
		Name: "Claude Desktop",
		ConfigPaths: []string{
			ResolveClaudeDesktopPath(),
		},
	}
	checkAgentExistence(&claudeDesktop)

	antigravity := AgentInfo{
		Name: "Antigravity IDE & CLI",
		ConfigPaths: []string{
			filepath.Join(homeDir, ".gemini", "config", "mcp_config.json"),
			filepath.Join(homeDir, ".gemini", "antigravity", "mcp_config.json"),
		},
		RulesPaths: []string{
			filepath.Join(homeDir, ".gemini", "GEMINI.md"),
			filepath.Join(homeDir, ".gemini", "AGENTS.md"),
		},
	}
	checkAgentExistence(&antigravity)

	printAgentReport(claudeCode)
	printAgentReport(claudeDesktop)
	printAgentReport(antigravity)

	analyzeGeminiFolderCollisions(homeDir)
}

func checkAgentExistence(info *AgentInfo) {
	detected := false
	for _, path := range info.ConfigPaths {
		if fileExists(path) {
			detected = true
		}
	}
	for _, path := range info.RulesPaths {
		if fileExists(path) {
			detected = true
		}
	}
	info.Detected = detected
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func printAgentReport(info AgentInfo) {
	if info.Detected {
		fmt.Printf("%s%s✔ %s%s — %sОБНАРУЖЕН%s\n", ColorGreen, ColorBold, info.Name, ColorReset, ColorGreen, ColorReset)
		for _, path := range info.ConfigPaths {
			if fileExists(path) {
				fmt.Printf("   Config: %s%s%s\n", ColorCyan, path, ColorReset)
			}
		}
		for _, path := range info.RulesPaths {
			if fileExists(path) {
				fmt.Printf("   Rules:  %s%s%s\n", ColorCyan, path, ColorReset)
			}
		}
	} else {
		fmt.Printf("%s%s✘ %s%s — %sНЕ НАЙДЕН%s\n", ColorRed, ColorBold, info.Name, ColorReset, ColorRed, ColorReset)
	}
	fmt.Println()
}

func analyzeGeminiFolderCollisions(homeDir string) {
	geminiDir := filepath.Join(homeDir, ".gemini")
	fmt.Printf("%s%s[+] Анализ коллизий в каталоге %s...%s\n", ColorBlue, ColorBold, geminiDir, ColorReset)

	fi, err := os.Stat(geminiDir)
	if os.IsNotExist(err) {
		fmt.Printf("   Каталог %s не существует. Коллизий нет.\n", geminiDir)
		return
	}

	if !fi.IsDir() {
		fmt.Printf("   %sПуть %s не является каталогом!%s\n", ColorRed, geminiDir, ColorReset)
		return
	}

	mcpConfig := filepath.Join(geminiDir, "config", "mcp_config.json")
	settingsJson := filepath.Join(geminiDir, "settings.json")
	geminiMd := filepath.Join(geminiDir, "GEMINI.md")

	hasMcpConfig := fileExists(mcpConfig)
	hasSettingsJson := fileExists(settingsJson)
	hasGeminiMd := fileExists(geminiMd)

	if hasMcpConfig && hasSettingsJson {
		fmt.Printf("   %s[!] ВНИМАНИЕ: Обнаружены оба файла mcp_config.json и settings.json!%s\n", ColorYellow, ColorReset)
		fmt.Println("       Это указывает на совместное использование каталога Antigravity и Gemini CLI.")
	} else {
		fmt.Println("   ✔ Конфликтов между mcp_config.json и settings.json не обнаружено.")
	}

	if hasGeminiMd {
		fmt.Printf("   %s[!] НАХОДКА: Обнаружен файл правил GEMINI.md.%s\n", ColorYellow, ColorReset)
		fmt.Printf("       Размер: %d байт\n", getFileSize(geminiMd))
	}
}

func getFileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fi.Size()
}

func runDoctorDiagnostics() {
	fmt.Printf("%s%s[+] Запуск полной диагностики системы...%s\n", ColorBlue, ColorBold, ColorReset)
	runDiscover()

	cwd, _ := os.Getwd()
	fmt.Printf("\n%s%s[+] Запуск аудита безопасности репозитория...%s\n", ColorBlue, ColorBold, ColorReset)
	warnings, err := AuditRepository(cwd)
	if err != nil {
		fmt.Printf("[-] Ошибка проведения аудита: %v\n", err)
		return
	}

	if len(warnings) > 0 {
		fmt.Printf("%s[-] Обнаружено потенциальных утечек секретов: %d%s\n", ColorRed, len(warnings), ColorReset)
		for _, w := range warnings {
			fmt.Println("  ", w)
		}
	} else {
		fmt.Printf("%s✔ Утечек секретов в репозитории не обнаружено.%s\n", ColorGreen, ColorReset)
	}
}

// AgentInfo содержит информацию об обнаруженном агенте
type AgentInfo struct {
	Name        string   `json:"name"`
	Detected    bool     `json:"detected"`
	ConfigPaths []string `json:"config_paths"`
	RulesPaths  []string `json:"rules_paths"`
	Version     string   `json:"version,omitempty"`
}
