package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Стили оформления Lip Gloss (WOW-эффект)
var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00F0FF")).
			Background(lipgloss.Color("#1A1A2E")).
			Padding(1, 3).
			MarginBottom(1)

	styleTab = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#3A3F58")).
			Padding(0, 2)

	styleActiveTab = styleTab.Copy().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#00F0FF")).
			Bold(true).
			Foreground(lipgloss.Color("#00F0FF"))

	styleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3A3F58")).
			Padding(1, 2) // Убрали Margin(0, 1) для стабильности рамок в Windows Terminal

	styleCardActive = styleCard.Copy().
			BorderForeground(lipgloss.Color("#00F0FF"))

	styleGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF87"))
	styleRed   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF416C"))
	styleCyan  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0FF"))
	styleGray  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7E849E"))
)

type tabId int

const (
	tabDashboard tabId = iota
	tabMCP
	tabInstall
	tabAirDrop
)

const maxVisibleCards = 2 // Лимит видимых карточек для скроллинга

type focusState int

const (
	focusComponents focusState = iota
	focusSources
	focusTargets
)

type tuiModel struct {
	activeTab      tabId
	cursor         int
	scrollOffset   int // Индекс первого видимого элемента для скролла
	agents         []AgentInfo
	components     []AgentComponent // универсальный список вместо mcps
	activeCategory ComponentType
	categories     []ComponentType
	registryItems  []RegistryItem
	registryLoaded bool
	activeInstallCategory ComponentType
	searchInput    textinput.Model
	searchMode     bool
	p2pNodes       []string
	scanning       bool
	logMsg         string
	radarAngle     int
	lastActionTime string

	// Новые поля для фокуса и секретов
	focus            focusState
	rightCursor      int // Курсор внутри правых столбиков
	secretNameInput  textinput.Model
	secretValueInput textinput.Model
	editingSecret    bool
	creatingSecret   bool
	showSecretValue  bool
}

func (m tuiModel) getVisibleComponents() []AgentComponent {
	var filtered []AgentComponent
	for _, c := range m.components {
		if c.Type == m.activeCategory {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (m tuiModel) getVisibleRegistryItems() []RegistryItem {
	var filtered []RegistryItem
	q := strings.ToLower(m.searchInput.Value())
	for _, c := range m.registryItems {
		if c.Type == m.activeInstallCategory {
			// Локальный фильтр
			if q == "" || strings.Contains(strings.ToLower(c.Name), q) || strings.Contains(strings.ToLower(c.Description), q) {
				filtered = append(filtered, c)
			}
		}
	}
	return filtered
}

func initialModel() tuiModel {
	home, _ := os.UserHomeDir()

	claudePath := filepath.Join(home, ".claude.json")
	claudeDesktopPath := ResolveClaudeDesktopPath()
	antigravityPath := filepath.Join(home, ".gemini", "config", "mcp_config.json")

	agents := []AgentInfo{
		{Name: "Claude Code (CLI)", ConfigPaths: []string{claudePath}, Detected: fileExists(claudePath)},
		{Name: "Claude Desktop", ConfigPaths: []string{claudeDesktopPath}, Detected: fileExists(claudeDesktopPath)},
		{Name: "Antigravity IDE & CLI", ConfigPaths: []string{antigravityPath}, Detected: fileExists(antigravityPath)},
	}

	cwd, _ := os.Getwd()
	_ = SmartImportMCPs(cwd) // Авто-импорт перед запуском
	store := NewStore(cwd)

	var components []AgentComponent

	// Загружаем MCP
	mcps, _ := store.LoadMCPs()
	for _, mcp := range mcps {
		var targets []AgentType
		for _, t := range mcp.Targets {
			targets = append(targets, AgentType(t))
		}
		details := fmt.Sprintf("Команда: %s %s", mcp.Command, strings.Join(mcp.Args, " "))
		components = append(components, AgentComponent{
			Type:        ComponentMCP,
			Name:        mcp.Name,
			Description: mcp.Description,
			Path:        filepath.Join("mcp", mcp.Name+".yaml"),
			Targets:     targets,
			Details:     details,
			Active:      true,
		})
	}

	// Загружаем Rules
	rules, _ := store.LoadRules()
	for _, rule := range rules {
		components = append(components, AgentComponent{
			Type:        ComponentRule,
			Name:        rule.Header.Name,
			Description: rule.Header.Description,
			Path:        filepath.Join("rules", rule.Header.Name+".md"),
			Targets:     rule.Header.Targets,
			Details:     fmt.Sprintf("Область применения: %s", rule.Header.Scope),
			Active:      true,
		})
	}

	// Загружаем Skills
	skills, _ := store.LoadSkills()
	for _, skill := range skills {
		components = append(components, AgentComponent{
			Type:        ComponentSkill,
			Name:        skill.Header.Name,
			Description: skill.Header.Description,
			Path:        filepath.Join("skills", skill.Header.Name, "SKILL.md"),
			Targets:     skill.Header.Targets,
			Details:     "Навык контекстного мышления (Skill)",
			Active:      true,
		})
	}

	// Загружаем Workflows
	wfs, _ := store.LoadWorkflows()
	for _, wf := range wfs {
		components = append(components, AgentComponent{
			Type:        ComponentWorkflow,
			Name:        wf.Header.Name,
			Description: wf.Header.Description,
			Path:        filepath.Join("workflows", wf.Header.Name),
			Targets:     wf.Header.Targets,
			Details:     "Сценарий рабочего процесса (Workflow)",
			Active:      true,
		})
	}

	// Загружаем Hooks
	hooks, _ := store.LoadHooks()
	for _, hook := range hooks {
		components = append(components, AgentComponent{
			Type:        ComponentHook,
			Name:        hook.Header.Name,
			Description: hook.Header.Description,
			Path:        filepath.Join("hooks", hook.Header.Name),
			Targets:     hook.Header.Targets,
			Details:     "Скрипт автоматизации коммитов/задач (Hook)",
			Active:      true,
		})
	}

	// Загружаем Secrets
	secrets, _ := LoadSecrets(cwd)
	var secretKeys []string
	for k := range secrets {
		secretKeys = append(secretKeys, k)
	}
	sort.Strings(secretKeys)
	for _, k := range secretKeys {
		components = append(components, AgentComponent{
			Type:        ComponentSecret,
			Name:        k,
			Description: "Конфиденциальный токен авторизации API",
			Details:     secrets[k],
			Active:      true,
		})
	}

	// Сортировка компонентов по типу и имени для группировки
	sort.Slice(components, func(i, j int) bool {
		if components[i].Type != components[j].Type {
			return components[i].Type < components[j].Type
		}
		return components[i].Name < components[j].Name
	})

	// Пытаемся быстро загрузить локальный кэш
	var regItems []RegistryItem
	var loaded bool
	cached, err := LoadLocalRegistry()
	if err == nil && len(cached) > 0 {
		regItems = cached
		loaded = true
	}

	ti := textinput.New()
	ti.Placeholder = "Введите запрос и нажмите Enter..."
	ti.CharLimit = 50
	ti.Width = 40
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0FF"))

	secNameInput := textinput.New()
	secNameInput.Placeholder = "Имя секрета (например, GITHUB_TOKEN)"
	secNameInput.CharLimit = 50
	secNameInput.Width = 30
	secNameInput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9"))

	secValInput := textinput.New()
	secValInput.Placeholder = "Значение секрета"
	secValInput.CharLimit = 255
	secValInput.Width = 45
	secValInput.EchoMode = textinput.EchoPassword // По умолчанию маскируем
	secValInput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9"))
	
	return tuiModel{
		activeTab:      tabDashboard,
		cursor:         0,
		scrollOffset:   0,
		agents:         agents,
		components:     components,
		activeCategory: ComponentMCP,
		categories:     []ComponentType{ComponentMCP, ComponentRule, ComponentSkill, ComponentWorkflow, ComponentHook, ComponentSecret},
		registryItems:  regItems,
		registryLoaded: loaded,
		activeInstallCategory: ComponentMCP,
		searchInput:    ti,
		searchMode:     false,
		scanning:       false,
		p2pNodes:       []string{},
		logMsg:         "AgentSync готов к работе.",
		radarAngle:     0,
		lastActionTime: time.Now().Format("15:04:05"),
		focus:          focusComponents,
		secretNameInput:  secNameInput,
		secretValueInput: secValInput,
	}
}

func (m tuiModel) Init() tea.Cmd {
	// При старте запускаем фоновую синхронизацию каталога (чтобы обновить кэш)
	return tea.Batch(syncRegistryCmd(), textinput.Blink)
}

type registryLoadedMsg []RegistryItem
type registrySyncFinishedMsg []RegistryItem

func syncRegistryCmd() tea.Cmd {
	return func() tea.Msg {
		_ = SyncLocalRegistry()
		items, err := LoadLocalRegistry()
		if err == nil {
			return registrySyncFinishedMsg(items)
		}
		return nil
	}
}

func fetchRegistryCmd(query string) tea.Cmd {
	return func() tea.Msg {
		var regItems []RegistryItem
		for _, c := range []ComponentType{ComponentMCP, ComponentSkill} { // Пока грузим только реальные теги с GitHub
			items, _ := FetchRegistry(c, query)
			regItems = append(regItems, items...)
		}
		return registryLoadedMsg(regItems)
	}
}

type tickMsg time.Time

type installResultMsg struct {
	err  error
	name string
}

func installCmd(item RegistryItem) tea.Cmd {
	return func() tea.Msg {
		err := InstallComponent(item)
		return installResultMsg{err: err, name: item.Name}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.scanning {
			m.radarAngle = (m.radarAngle + 45) % 360
			return m, tickCmd()
		}
		return m, nil

	case tea.KeyMsg:
		// Обработка ввода для секретов
		if m.activeTab == tabMCP && m.activeCategory == ComponentSecret {
			if m.editingSecret || m.creatingSecret {
				var cmd tea.Cmd
				switch msg.String() {
				case "esc":
					m.editingSecret = false
					m.creatingSecret = false
					m.secretNameInput.Blur()
					m.secretValueInput.Blur()
					return m, nil
				case "enter":
					cwd, _ := os.Getwd()
					secrets, _ := LoadSecrets(cwd)
					
					var name, val string
					if m.creatingSecret {
						name = strings.TrimSpace(m.secretNameInput.Value())
						val = m.secretValueInput.Value()
					} else {
						visible := m.getVisibleComponents()
						if m.cursor >= 0 && m.cursor < len(visible) {
							name = visible[m.cursor].Name
							val = m.secretValueInput.Value()
						}
					}
					
					if name != "" {
						secrets[name] = val
						err := SaveSecrets(cwd, secrets)
						if err != nil {
							m.logMsg = fmt.Sprintf("Ошибка сохранения секрета: %v", err)
						} else {
							m.logMsg = fmt.Sprintf("Секрет %s сохранен!", name)
							m.components = updateTuiSecrets(m.components, secrets)
						}
					}
					
					m.editingSecret = false
					m.creatingSecret = false
					m.secretNameInput.Blur()
					m.secretValueInput.Blur()
					m.secretNameInput.SetValue("")
					m.secretValueInput.SetValue("")
					return m, nil
				case "tab":
					if m.creatingSecret {
						if m.secretNameInput.Focused() {
							m.secretNameInput.Blur()
							m.secretValueInput.Focus()
						} else {
							m.secretValueInput.Blur()
							m.secretNameInput.Focus()
						}
					}
					return m, nil
				}
				
				if m.creatingSecret && m.secretNameInput.Focused() {
					m.secretNameInput, cmd = m.secretNameInput.Update(msg)
					return m, cmd
				} else {
					m.secretValueInput, cmd = m.secretValueInput.Update(msg)
					return m, cmd
				}
			}
		}

		// Обработка режима поиска (глобальный перехват ввода)
		if m.activeTab == tabInstall && m.searchMode {
			var cmd tea.Cmd
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.searchInput.Blur()
				return m, nil
			case "enter":
				m.searchMode = false
				m.searchInput.Blur()
				m.registryLoaded = false // Покажем спиннер
				return m, fetchRegistryCmd(m.searchInput.Value())
			}
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab": // Вкладка вправо
			m.activeTab = (m.activeTab + 1) % 4
			m.cursor = 0
			m.scrollOffset = 0
			m.focus = focusComponents
			m.logMsg = "Вкладка переключена."
			return m, nil

		case "shift+tab": // Вкладка влево
			if m.activeTab == 0 {
				m.activeTab = 3
			} else {
				m.activeTab--
			}
			m.cursor = 0
			m.scrollOffset = 0
			m.focus = focusComponents
			m.logMsg = "Вкладка переключена."
			return m, nil

		case "right": // Фокус вправо в Agent Manager
			if m.activeTab == tabMCP {
				if m.activeCategory == ComponentSecret {
					return m, nil
				}
				if m.focus == focusComponents {
					m.focus = focusSources
					m.rightCursor = 0
				} else if m.focus == focusSources {
					m.focus = focusTargets
					m.rightCursor = 0
				}
				return m, nil
			}
			return m, nil

		case "left": // Фокус влево в Agent Manager
			if m.activeTab == tabMCP {
				if m.focus == focusTargets {
					m.focus = focusSources
					m.rightCursor = 0
				} else if m.focus == focusSources {
					m.focus = focusComponents
				}
				return m, nil
			}
			return m, nil

		case "]": // Категория вправо (вместо стрелки вправо)
			if m.activeTab == tabMCP {
				for i, c := range m.categories {
					if c == m.activeCategory {
						m.activeCategory = m.categories[(i+1)%len(m.categories)]
						m.cursor = 0
						m.scrollOffset = 0
						m.focus = focusComponents
						break
					}
				}
			} else if m.activeTab == tabInstall {
				for i, c := range m.categories {
					if c == m.activeInstallCategory {
						m.activeInstallCategory = m.categories[(i+1)%len(m.categories)]
						m.cursor = 0
						m.scrollOffset = 0
						break
					}
				}
			}
			return m, nil

		case "[": // Категория влево (вместо стрелки влево)
			if m.activeTab == tabMCP {
				for i, c := range m.categories {
					if c == m.activeCategory {
						if i == 0 {
							m.activeCategory = m.categories[len(m.categories)-1]
						} else {
							m.activeCategory = m.categories[i-1]
						}
						m.cursor = 0
						m.scrollOffset = 0
						m.focus = focusComponents
						break
					}
				}
			} else if m.activeTab == tabInstall {
				for i, c := range m.categories {
					if c == m.activeInstallCategory {
						if i == 0 {
							m.activeInstallCategory = m.categories[len(m.categories)-1]
						} else {
							m.activeInstallCategory = m.categories[i-1]
						}
						m.cursor = 0
						m.scrollOffset = 0
						break
					}
				}
			}
			return m, nil

		case "up":
			if m.activeTab == tabMCP && m.focus != focusComponents {
				if m.rightCursor > 0 {
					m.rightCursor--
				}
				return m, nil
			}
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scrollOffset {
					m.scrollOffset = m.cursor
				}
			}
			return m, nil

		case "down":
			if m.activeTab == tabMCP && m.focus != focusComponents {
				maxRight := 1
				if m.focus == focusTargets {
					maxRight = 2
				}
				if m.rightCursor < maxRight {
					m.rightCursor++
				}
				return m, nil
			}

			max := 0
			if m.activeTab == tabDashboard {
				max = len(m.agents) - 1
			} else if m.activeTab == tabMCP {
				max = len(m.getVisibleComponents()) - 1
			} else if m.activeTab == tabInstall {
				max = len(m.getVisibleRegistryItems()) - 1
			} else if m.activeTab == tabAirDrop {
				max = len(m.p2pNodes) - 1
			}
			
			if m.cursor < max {
				m.cursor++
				if m.cursor >= m.scrollOffset+maxVisibleCards {
					m.scrollOffset = m.cursor - maxVisibleCards + 1
				}
			}
			return m, nil

		case "enter":
			if m.activeTab == tabInstall {
				visible := m.getVisibleRegistryItems()
				if m.cursor >= 0 && m.cursor < len(visible) {
					item := visible[m.cursor]
					if !item.Installed {
						m.logMsg = fmt.Sprintf("Скачивание %s...", item.Name)
						m.lastActionTime = time.Now().Format("15:04:05")
						return m, installCmd(item)
					}
				}
			}
			return m, nil

		case "d": // Мгновенный деплой
			m.logMsg = "Слияние и деплой запущены..."
			m.lastActionTime = time.Now().Format("15:04:05")
			go func() {
				runDeploy("")
			}()
			m.logMsg = "Деплой успешно применен ко всем агентам!"
			return m, nil

		case "/", "s": // Вход в режим поиска
			if m.activeTab == tabInstall {
				m.searchMode = true
				m.searchInput.Focus()
				return m, textinput.Blink
			}
			// Для AirDrop `s` означает сканирование
			if m.activeTab == tabAirDrop && msg.String() == "s" {
				m.scanning = true
				m.logMsg = "mDNS: Сканирование локального эфира..."
				m.p2pNodes = []string{}
				m.lastActionTime = time.Now().Format("15:04:05")
				return m, tea.Batch(
					tickCmd(),
					func() tea.Msg {
						nodes, _ := DiscoverServices(time.Second * 3)
						return nodes
					},
				)
			}
			return m, nil

		case "e": // Редактировать выбранный секрет
			if m.activeTab == tabMCP && m.activeCategory == ComponentSecret {
				visible := m.getVisibleComponents()
				if m.cursor >= 0 && m.cursor < len(visible) {
					m.editingSecret = true
					m.secretValueInput.Focus()
					m.secretValueInput.SetValue(visible[m.cursor].Details)
					m.logMsg = fmt.Sprintf("Редактирование секрета %s. Введите значение и нажмите Enter...", visible[m.cursor].Name)
				}
				return m, nil
			}
			return m, nil

		case "n": // Создать новый секрет
			if m.activeTab == tabMCP && m.activeCategory == ComponentSecret {
				m.creatingSecret = true
				m.secretNameInput.Focus()
				m.secretNameInput.SetValue("")
				m.secretValueInput.SetValue("")
				m.logMsg = "Создание нового секрета. Введите имя, нажмите Tab, введите значение и нажмите Enter..."
				return m, nil
			}
			return m, nil

		case "delete": // Удалить секрет
			if m.activeTab == tabMCP && m.activeCategory == ComponentSecret {
				visible := m.getVisibleComponents()
				if m.cursor >= 0 && m.cursor < len(visible) {
					targetName := visible[m.cursor].Name
					cwd, _ := os.Getwd()
					secrets, _ := LoadSecrets(cwd)
					delete(secrets, targetName)
					err := SaveSecrets(cwd, secrets)
					if err != nil {
						m.logMsg = fmt.Sprintf("Ошибка удаления секрета: %v", err)
					} else {
						m.logMsg = fmt.Sprintf("Секрет %s успешно удален!", targetName)
						m.components = updateTuiSecrets(m.components, secrets)
						m.cursor = 0
						m.scrollOffset = 0
					}
				}
				return m, nil
			}
			return m, nil

		case "v": // Показать/скрыть значение секрета
			if m.activeTab == tabMCP && m.activeCategory == ComponentSecret {
				m.showSecretValue = !m.showSecretValue
				if m.showSecretValue {
					m.secretValueInput.EchoMode = textinput.EchoNormal
				} else {
					m.secretValueInput.EchoMode = textinput.EchoPassword
				}
				m.logMsg = "Режим отображения секрета изменен."
				return m, nil
			}
			return m, nil

		case "esc": // Выход из фокуса правой панели
			if m.activeTab == tabMCP && m.focus != focusComponents {
				m.focus = focusComponents
				m.logMsg = "Фокус возвращен к списку компонентов."
				return m, nil
			}
			return m, nil

		case " ": // Пробел: переключение галочек источников и целей в Agent Manager
			if m.activeTab == tabMCP {
				visible := m.getVisibleComponents()
				if m.cursor < 0 || m.cursor >= len(visible) {
					return m, nil
				}
				comp := visible[m.cursor]
				cwd, _ := os.Getwd()

				if m.focus == focusSources {
					// Копирование источника (синхронизация)
					home, _ := os.UserHomeDir()
					var subDir string
					var fileName string
					switch comp.Type {
					case ComponentMCP:
						subDir = "mcp"
						fileName = comp.Name + ".yaml"
					case ComponentRule:
						subDir = "rules"
						fileName = comp.Name + ".md"
					case ComponentSkill:
						subDir = "skills"
						fileName = filepath.Join(comp.Name, "SKILL.md")
					case ComponentWorkflow:
						subDir = "workflows"
						fileName = comp.Name
					case ComponentHook:
						subDir = "hooks"
						fileName = comp.Name
					}
					
					localExists := fileExists(filepath.Join(cwd, subDir, fileName))
					globalExists := fileExists(filepath.Join(home, ".agents", subDir, fileName))

					if m.rightCursor == 0 && !localExists && globalExists {
						err := SyncComponentSource(cwd, comp, false)
						if err != nil {
							m.logMsg = fmt.Sprintf("Ошибка копирования в CWD: %v", err)
						} else {
							m.logMsg = fmt.Sprintf("Успешно: %s скопирован локально (CWD)", comp.Name)
						}
					} else if m.rightCursor == 1 && localExists && !globalExists {
						err := SyncComponentSource(cwd, comp, true)
						if err != nil {
							m.logMsg = fmt.Sprintf("Ошибка копирования в Global: %v", err)
						} else {
							m.logMsg = fmt.Sprintf("Успешно: %s скопирован глобально (~/.agents)", comp.Name)
						}
					}
					m.lastActionTime = time.Now().Format("15:04:05")
					return m, nil
				}

				if m.focus == focusTargets {
					agentsList := []AgentType{AgentClaudeCode, AgentClaudeDesktop, AgentAntigravity}
					targetAgent := agentsList[m.rightCursor]

					var newTargets []AgentType
					found := false
					for _, t := range comp.Targets {
						if t == targetAgent {
							found = true
						} else {
							newTargets = append(newTargets, t)
						}
					}
					if !found {
						newTargets = append(newTargets, targetAgent)
					}

					store := NewStore(cwd)
					var err error
					switch comp.Type {
					case ComponentMCP:
						err = store.UpdateMCPTargets(comp.Name, newTargets)
					case ComponentRule:
						err = store.UpdateRuleTargets(comp.Name, newTargets)
					case ComponentSkill:
						err = store.UpdateSkillTargets(comp.Name, newTargets)
					case ComponentWorkflow:
						err = store.UpdateWorkflowTargets(comp.Name, newTargets)
					}

					if err != nil {
						m.logMsg = fmt.Sprintf("Ошибка обновления целей: %v", err)
					} else {
						m.logMsg = fmt.Sprintf("Обновлены цели для %s: %v", comp.Name, newTargets)
						for idx, c := range m.components {
							if c.Type == comp.Type && c.Name == comp.Name {
								m.components[idx].Targets = newTargets
								break
							}
						}
					}
					m.lastActionTime = time.Now().Format("15:04:05")
					return m, nil
				}
			}
			return m, nil
		}

	case []string: // Результат сканирования mDNS
		m.scanning = false
		m.p2pNodes = msg
		if len(m.p2pNodes) == 0 {
			m.logMsg = "mDNS: Сканирование завершено. Других нод не найдено."
		} else {
			m.logMsg = fmt.Sprintf("mDNS: Найдено активных нод: %d", len(m.p2pNodes))
		}
		return m, nil

	case registrySyncFinishedMsg:
		m.registryLoaded = true
		m.registryItems = msg
		m.logMsg = "Фоновая синхронизация глобального каталога завершена."
		m.lastActionTime = time.Now().Format("15:04:05")
		return m, nil

	case registryLoadedMsg:
		m.registryLoaded = true
		m.registryItems = msg
		m.logMsg = "Глобальный каталог загружен из GitHub."
		m.lastActionTime = time.Now().Format("15:04:05")
		return m, nil

	case installResultMsg:
		m.lastActionTime = time.Now().Format("15:04:05")
		if msg.err != nil {
			m.logMsg = fmt.Sprintf("Ошибка скачивания %s: %v", msg.name, msg.err)
		} else {
			m.logMsg = fmt.Sprintf("Успешно установлено: %s", msg.name)
			for i, r := range m.registryItems {
				if r.Name == msg.name {
					m.registryItems[i].Installed = true
					break
				}
			}
		}
		return m, nil
	}

	return m, nil
}

func (m tuiModel) View() string {
	var s strings.Builder

	// 1. Заголовок приложения
	s.WriteString(styleTitle.Render("⚡ AGENTSYNC TERMINAL UI ⚡"))
	s.WriteString("\n")

	// 2. Вкладки (Tabs)
	var tabs []string
	tabNames := []string{"  🎛 DASHBOARD  ", "  💼 AGENT MANAGER  ", "  ☁ INSTALL  ", "  📡 P2P AIRDROP  "}
	for i, name := range tabNames {
		if tabId(i) == m.activeTab {
			tabs = append(tabs, styleActiveTab.Render(name))
		} else {
			tabs = append(tabs, styleTab.Render(name))
		}
	}
	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
	s.WriteString("\n\n")

	// 3. Содержимое вкладки
	switch m.activeTab {
	case tabDashboard:
		s.WriteString("Статус AI-агентов на этом компьютере:\n\n")
		
		// КРИТИЧЕСКИЙ ФИКС: Вынесли "\n" за пределы .Render() для стабильности терминала
		if m.scrollOffset > 0 {
			s.WriteString(styleGray.Render("   ▲ Показать предыдущих агентов...") + "\n")
		} else {
			s.WriteString("\n")
		}

		end := m.scrollOffset + maxVisibleCards
		if end > len(m.agents) {
			end = len(m.agents)
		}

		for i := m.scrollOffset; i < end; i++ {
			agent := m.agents[i]
			isSelected := i == m.cursor
			var cardContent string
			statusStr := styleRed.Render("✘ НЕ НАЙДЕН")
			if agent.Detected {
				statusStr = styleGreen.Render("✔ ОБНАРУЖЕН")
			}

			cardContent = fmt.Sprintf("%s%s%s\nСтатус: %s\nПуть:   %s",
				lipgloss.NewStyle().Bold(true).Render(agent.Name),
				styleGray.Render(" (конфиг обнаружен)"),
				"\n",
				statusStr,
				styleGray.Render(agent.ConfigPaths[0]),
			)

			if isSelected {
				s.WriteString(styleCardActive.Render(cardContent))
			} else {
				s.WriteString(styleCard.Render(cardContent))
			}
			s.WriteString("\n")
		}

		// КРИТИЧЕСКИЙ ФИКС: Вынесли "\n" за пределы .Render()
		if m.scrollOffset+maxVisibleCards < len(m.agents) {
			s.WriteString(styleGray.Render("   ▼ Показать следующих агентов...") + "\n")
		}

	case tabMCP:
		s.WriteString("Управление компонентами агентов в каноническом репозитории:\n")
		s.WriteString(styleGray.Render("Используйте [ и ] для категорий | Стрелки Left/Right для фокуса | Space для переключения") + "\n\n")

		var subTabs []string
		for _, cat := range m.categories {
			if cat == m.activeCategory {
				subTabs = append(subTabs, styleActiveTab.Render(string(cat)))
			} else {
				subTabs = append(subTabs, styleTab.Render(string(cat)))
			}
		}
		s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, subTabs...) + "\n\n")

		visible := m.getVisibleComponents()

		// --- ЛЕВАЯ ПАНЕЛЬ: СПИСОК КОМПОНЕНТОВ ---
		var leftPanel strings.Builder
		if len(visible) == 0 {
			leftPanel.WriteString("\n   Нет установленных компонентов.\n")
		} else {
			if m.scrollOffset > 0 {
				leftPanel.WriteString(styleGray.Render("   ▲ Предыдущие...") + "\n")
			} else {
				leftPanel.WriteString("\n")
			}

			end := m.scrollOffset + maxVisibleCards
			if end > len(visible) {
				end = len(visible)
			}

			for i := m.scrollOffset; i < end; i++ {
				comp := visible[i]
				isSelected := i == m.cursor
				
				var icon string
				var typeColor string
				switch comp.Type {
				case ComponentMCP:
					icon = "🔌"
					typeColor = "#00F0FF"
				case ComponentRule:
					icon = "📜"
					typeColor = "#00FF87"
				case ComponentSkill:
					icon = "🧠"
					typeColor = "#BD93F9"
				case ComponentWorkflow:
					icon = "🔄"
					typeColor = "#FFB86C"
				case ComponentHook:
					icon = "🪝"
					typeColor = "#FF5555"
				case ComponentSecret:
					icon = "🔐"
					typeColor = "#FF79C6"
				}

				isSelectedList := isSelected && m.focus == focusComponents
				styleCompType := lipgloss.NewStyle().Foreground(lipgloss.Color(typeColor)).Bold(true)

				cardContent := fmt.Sprintf("%s %s\n[%s]", icon, comp.Name, styleCompType.Render(string(comp.Type)))
				
				var renderStyle lipgloss.Style
				if isSelectedList {
					renderStyle = styleCard.Copy().BorderForeground(lipgloss.Color(typeColor)).Width(26).Bold(true)
				} else {
					renderStyle = styleCard.Copy().BorderForeground(lipgloss.Color("#3A3F58")).Width(26)
				}
				
				leftPanel.WriteString(renderStyle.Render(cardContent) + "\n")
			}

			if m.scrollOffset+maxVisibleCards < len(visible) {
				leftPanel.WriteString(styleGray.Render("   ▼ Следующие...") + "\n")
			}
		}

		// --- ПРАВАЯ ПАНЕЛЬ: НАСТРОЙКИ / СЕКРЕТЫ ---
		var rightPanel strings.Builder
		if len(visible) > 0 && m.cursor >= 0 && m.cursor < len(visible) {
			comp := visible[m.cursor]

			if m.activeCategory == ComponentSecret {
				// Рендерим Менеджер Секретов
				rightPanel.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6")).Render("🔐 УПРАВЛЕНИЕ СЕКРЕТОМ") + "\n\n")
				rightPanel.WriteString(fmt.Sprintf("Секрет: %s\n\n", lipgloss.NewStyle().Bold(true).Render(comp.Name)))

				if m.editingSecret {
					rightPanel.WriteString(styleCyan.Render("Введите новое значение:") + "\n")
					rightPanel.WriteString(m.secretValueInput.View() + "\n\n")
					rightPanel.WriteString(styleGray.Render("Enter: сохранить | Esc: отмена") + "\n")
				} else if m.creatingSecret {
					rightPanel.WriteString(styleCyan.Render("Создание секрета:") + "\n")
					rightPanel.WriteString("Имя:\n" + m.secretNameInput.View() + "\n")
					rightPanel.WriteString("Значение:\n" + m.secretValueInput.View() + "\n\n")
					rightPanel.WriteString(styleGray.Render("Tab: переключение полей | Enter: сохранить | Esc: отмена") + "\n")
				} else {
					// Обычный просмотр
					valStr := "******"
					if m.showSecretValue {
						valStr = comp.Details
					}
					rightPanel.WriteString(fmt.Sprintf("Значение: %s\n\n", styleGreen.Render(valStr)))
					
					rightPanel.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#3A3F58")).Render("───────────────────────────────") + "\n")
					rightPanel.WriteString(fmt.Sprintf(" %s | %s\n %s | %s\n",
						styleCyan.Render("e: изменить значение"),
						styleCyan.Render("n: новый секрет"),
						styleRed.Render("delete: удалить"),
						styleCyan.Render("v: показать/скрыть"),
					))
				}
			} else {
				// Рендерим синхронизацию компонентов (Источники и Цели)
				rightPanel.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF")).Render("🛠 НАСТРОЙКИ СИНХРОНИЗАЦИИ") + "\n")
				rightPanel.WriteString(fmt.Sprintf("Компонент: %s\n\n", lipgloss.NewStyle().Bold(true).Render(comp.Name)))

				cwd, _ := os.Getwd()
				home, _ := os.UserHomeDir()
				
				// Вычисляем папки
				var subDir string
				var fileName string
				switch comp.Type {
				case ComponentMCP:
					subDir = "mcp"
					fileName = comp.Name + ".yaml"
				case ComponentRule:
					subDir = "rules"
					fileName = comp.Name + ".md"
				case ComponentSkill:
					subDir = "skills"
					fileName = filepath.Join(comp.Name, "SKILL.md")
				case ComponentWorkflow:
					subDir = "workflows"
					fileName = comp.Name
				case ComponentHook:
					subDir = "hooks"
					fileName = comp.Name
				}

				localPathExists := fileExists(filepath.Join(cwd, subDir, fileName))
				globalPathExists := fileExists(filepath.Join(home, ".agents", subDir, fileName))

				// --- СТОЛБИК 1: ИСТОЧНИКИ ---
				var srcCol strings.Builder
				srcCol.WriteString(lipgloss.NewStyle().Bold(true).Underline(true).Render("ИСТОЧНИКИ:") + "\n")
				
				localActive := "[ ]"
				if localPathExists {
					localActive = styleGreen.Render("[x]")
				}
				localCursor := " "
				if m.focus == focusSources && m.rightCursor == 0 {
					localCursor = styleCyan.Render("►")
				}
				srcCol.WriteString(fmt.Sprintf("%s %s Локальный (CWD)\n", localCursor, localActive))

				globalActive := "[ ]"
				if globalPathExists {
					globalActive = styleGreen.Render("[x]")
				}
				globalCursor := " "
				if m.focus == focusSources && m.rightCursor == 1 {
					globalCursor = styleCyan.Render("►")
				}
				srcCol.WriteString(fmt.Sprintf("%s %s Глобальный\n\n", globalCursor, globalActive))
				
				if m.focus == focusSources {
					srcCol.WriteString(styleGray.Render("Space: скопировать\n(в пустой источник)\nEsc: выйти"))
				}

				// --- СТОЛБИК 2: ЦЕЛИ ---
				var dstCol strings.Builder
				dstCol.WriteString(lipgloss.NewStyle().Bold(true).Underline(true).Render("ЦЕЛИ (TARGETS):") + "\n")

				agentsList := []AgentType{AgentClaudeCode, AgentClaudeDesktop, AgentAntigravity}
				for idx, agent := range agentsList {
					hasTarget := false
					for _, t := range comp.Targets {
						if t == agent {
							hasTarget = true
							break
						}
					}
					activeSymbol := "[ ]"
					if hasTarget {
						activeSymbol = styleGreen.Render("[x]")
					}
					targetCursor := " "
					if m.focus == focusTargets && m.rightCursor == idx {
						targetCursor = styleCyan.Render("►")
					}
					dstCol.WriteString(fmt.Sprintf("%s %s %s\n", targetCursor, activeSymbol, string(agent)))
				}
				dstCol.WriteString("\n")
				if m.focus == focusTargets {
					dstCol.WriteString(styleGray.Render("Space: вкл/выкл\nEsc: выйти"))
				}

				// Объединяем два столбика в горизонтальную сетку
				srcPanelStyle := lipgloss.NewStyle().Width(18)
				dstPanelStyle := lipgloss.NewStyle().Width(22)
				
				colsLayout := lipgloss.JoinHorizontal(lipgloss.Top,
					srcPanelStyle.Render(srcCol.String()),
					dstPanelStyle.Render(dstCol.String()),
				)
				rightPanel.WriteString(colsLayout)
			}
		} else {
			rightPanel.WriteString("\n Выберите компонент для просмотра параметров.")
		}

		// Объединяем Левую и Правую панели
		mainLayout := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(32).Render(leftPanel.String()),
			lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#3A3F58")).PaddingLeft(2).Width(42).Render(rightPanel.String()),
		)
		s.WriteString(mainLayout)
		s.WriteString("\n")

	case tabInstall:
		s.WriteString("Загрузка плагинов и компонентов из глобального каталога:\n\n")

		var subTabs []string
		for _, cat := range m.categories {
			if cat == m.activeInstallCategory {
				subTabs = append(subTabs, styleActiveTab.Render(string(cat)))
			} else {
				subTabs = append(subTabs, styleTab.Render(string(cat)))
			}
		}
		s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, subTabs...) + "\n\n")

		// Отрисовка строки поиска
		if m.searchMode {
			s.WriteString(styleCyan.Render("Поиск: ") + m.searchInput.View() + "\n\n")
		} else {
			s.WriteString(styleGray.Render("Нажмите '/' или 's' для поиска") + "\n\n")
		}

		if !m.registryLoaded {
			s.WriteString(styleCyan.Render("   ⏳ Поиск компонентов в GitHub (API)... Пожалуйста, подождите.\n"))
			break
		}

		visible := m.getVisibleRegistryItems()

		if m.scrollOffset > 0 {
			s.WriteString(styleGray.Render("   ▲ Показать предыдущие...") + "\n")
		} else {
			s.WriteString("\n")
		}

		end := m.scrollOffset + maxVisibleCards
		if end > len(visible) {
			end = len(visible)
		}

		for i := m.scrollOffset; i < end; i++ {
			item := visible[i]
			isSelected := i == m.cursor

			var targetsList []string
			for _, t := range item.Targets {
				targetsList = append(targetsList, string(t))
			}
			targetsStr := strings.Join(targetsList, ", ")

			var icon string
			var typeColor string
			switch item.Type {
			case ComponentMCP:
				icon = "🔌"
				typeColor = "#00F0FF"
			case ComponentRule:
				icon = "📜"
				typeColor = "#00FF87"
			case ComponentSkill:
				icon = "🧠"
				typeColor = "#BD93F9"
			case ComponentWorkflow:
				icon = "🔄"
				typeColor = "#FFB86C"
			case ComponentHook:
				icon = "🪝"
				typeColor = "#FF5555"
			}

			styleCompType := lipgloss.NewStyle().Foreground(lipgloss.Color(typeColor)).Bold(true)
			
			installStatus := styleGray.Render("Нажмите Enter для установки")
			if item.Installed {
				installStatus = styleGreen.Render("✔ УСТАНОВЛЕНО")
			}

			cardContent := fmt.Sprintf("%s %s [%s]\n%s\nИз:      %s\nАктивен: %s\n%s",
				icon,
				lipgloss.NewStyle().Bold(true).Render(item.Name),
				styleCompType.Render(string(item.Type)),
				item.Description,
				styleGray.Render(item.SourceURL),
				styleGreen.Render(targetsStr),
				installStatus,
			)

			if isSelected {
				styleActiveCard := styleCard.Copy().BorderForeground(lipgloss.Color(typeColor))
				s.WriteString(styleActiveCard.Render(cardContent))
			} else {
				s.WriteString(styleCard.Render(cardContent))
			}
			s.WriteString("\n")
		}

		if m.scrollOffset+maxVisibleCards < len(visible) {
			s.WriteString(styleGray.Render("   ▼ Показать следующие...") + "\n")
		}

	case tabAirDrop:
		s.WriteString("Быстрый шеринг конфигураций по локальной сети:\n\n")
		
		var statusStr string
		if m.scanning {
			radarChars := []string{"◐", "◓", "◑", "◒"}
			char := radarChars[(m.radarAngle/45)%4]
			statusStr = fmt.Sprintf("%s Сканирование Wi-Fi эфира...", styleCyan.Render(char))
		} else {
			statusStr = styleGray.Render("Нажмите 'S' для сканирования локальной сети.")
		}
		
		s.WriteString(fmt.Sprintf("   %s\n\n", statusStr))

		if len(m.p2pNodes) == 0 && !m.scanning {
			s.WriteString("   Ноды коллег не обнаружены. Убедитесь, что у друга запущен 'agentsync share'.\n\n")
		} else {
			// КРИТИЧЕСКИЙ ФИКС: Вынесли "\n" за пределы .Render()
			if m.scrollOffset > 0 {
				s.WriteString(styleGray.Render("   ▲ Показать предыдущие ноды...") + "\n")
			}

			end := m.scrollOffset + maxVisibleCards
			if end > len(m.p2pNodes) {
				end = len(m.p2pNodes)
			}

			for i := m.scrollOffset; i < end; i++ {
				node := m.p2pNodes[i]
				isSelected := i == m.cursor
				cardContent := fmt.Sprintf("💻 Нода друга: %s\nДоступно:      github, react-style\nДействие:      Нажмите Enter для импорта (AirDrop)", node)
				if isSelected {
					s.WriteString(styleCardActive.Render(cardContent))
				} else {
					s.WriteString(styleCard.Render(cardContent))
				}
				s.WriteString("\n")
			}

			// КРИТИЧЕСКИЙ ФИКС: Вынесли "\n" за пределы .Render()
			if m.scrollOffset+maxVisibleCards < len(m.p2pNodes) {
				s.WriteString(styleGray.Render("   ▼ Показать следующие ноды...") + "\n")
			}
		}
	}

	// 4. Панель логов и горячих клавиш (Footer)
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false, false, false).BorderForeground(lipgloss.Color("#3A3F58")).Width(65).Render(""))
	s.WriteString("\n")

	// Логи событий
	s.WriteString(fmt.Sprintf(" [%s] ℹ %s\n\n", styleCyan.Render(m.lastActionTime), m.logMsg))

	// Кнопки управления
	s.WriteString(fmt.Sprintf(" %s | %s | %s | %s\n",
		styleCyan.Render("← / → / Tab: вкладки"),
		styleCyan.Render("↑ / ↓: скролл элементов"),
		styleCyan.Render("D: deploy"),
		styleCyan.Render("Q: выход"),
	))

	return s.String()
}

// StartTUI запускает интерактивный терминальный интерфейс
func StartTUI() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func updateTuiSecrets(components []AgentComponent, secrets map[string]string) []AgentComponent {
	var filtered []AgentComponent
	for _, c := range components {
		if c.Type != ComponentSecret {
			filtered = append(filtered, c)
		}
	}
	var secretKeys []string
	for k := range secrets {
		secretKeys = append(secretKeys, k)
	}
	sort.Strings(secretKeys)
	for _, k := range secretKeys {
		filtered = append(filtered, AgentComponent{
			Type:        ComponentSecret,
			Name:        k,
			Description: "Конфиденциальный токен авторизации API",
			Details:     secrets[k],
			Active:      true,
		})
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Type != filtered[j].Type {
			return filtered[i].Type < filtered[j].Type
		}
		return filtered[i].Name < filtered[j].Name
	})
	return filtered
}

func SyncComponentSource(repoPath string, comp AgentComponent, toGlobal bool) error {
	var subDir string
	var fileName string
	switch comp.Type {
	case ComponentMCP:
		subDir = "mcp"
		fileName = comp.Name + ".yaml"
	case ComponentRule:
		subDir = "rules"
		fileName = comp.Name + ".md"
	case ComponentSkill:
		subDir = "skills"
		fileName = filepath.Join(comp.Name, "SKILL.md")
	case ComponentWorkflow:
		subDir = "workflows"
		fileName = comp.Name
	case ComponentHook:
		subDir = "hooks"
		fileName = comp.Name
	default:
		return fmt.Errorf("неподдерживаемый тип")
	}

	localPath := filepath.Join(repoPath, subDir, fileName)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	globalPath := filepath.Join(home, ".agents", subDir, fileName)

	var src, dst string
	if toGlobal {
		src = localPath
		dst = globalPath
	} else {
		src = globalPath
		dst = localPath
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}
