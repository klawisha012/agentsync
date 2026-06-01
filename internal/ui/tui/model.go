package tui

import (
	"agentsync/internal/agent"
	"agentsync/internal/domain"
	"agentsync/internal/repository"
	"agentsync/internal/service"
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

type tabId int

const (
	tabDashboard tabId = iota
	tabMCP
	tabInstall
	tabAirDrop
)

const maxVisibleCards = 2

type focusState int

const (
	focusComponents focusState = iota
	focusSources
	focusTargets
)

type tuiModel struct {
	repoPath              string
	activeTab             tabId
	cursor                int
	scrollOffset          int
	agents                []agentInfoTui
	components            []domain.AgentComponent
	activeCategory        domain.ComponentType
	categories            []domain.ComponentType
	registryItems         []service.RegistryItem
	registryLoaded        bool
	activeInstallCategory domain.ComponentType
	searchInput           textinput.Model
	searchMode            bool
	p2pNodes              []string
	scanning              bool
	logMsg                string
	radarAngle            int
	lastActionTime        string

	focus            focusState
	rightCursor      int
	secretNameInput  textinput.Model
	secretValueInput textinput.Model
	editingSecret    bool
	creatingSecret   bool
	showSecretValue  bool
}

type agentInfoTui struct {
	Name        string   `json:"name"`
	Detected    bool     `json:"detected"`
	ConfigPaths []string `json:"config_paths"`
}

func StartTUI(repoPath string) error {
	p := tea.NewProgram(initialModel(repoPath), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func initialModel(repoPath string) tuiModel {
	home, _ := os.UserHomeDir()

	claudePath := filepath.Join(home, ".claude.json")
	claudeDesktopPath := agent.ResolveClaudeDesktopPath()
	antigravityPath := filepath.Join(home, ".gemini", "config", "mcp_config.json")

	agents := []agentInfoTui{
		{Name: "Claude Code (CLI)", ConfigPaths: []string{claudePath}, Detected: fileExistsTui(claudePath)},
		{Name: "Claude Desktop", ConfigPaths: []string{claudeDesktopPath}, Detected: fileExistsTui(claudeDesktopPath)},
		{Name: "Antigravity IDE & CLI", ConfigPaths: []string{antigravityPath}, Detected: fileExistsTui(antigravityPath)},
	}

	_ = service.NewDeployService(repoPath, nil).SmartImportMCPs()
	store := repository.NewStore(repoPath)

	var components []domain.AgentComponent

	// Загружаем MCP
	mcps, _ := store.LoadMCPs()
	for _, mcp := range mcps {
		var targets []domain.AgentType
		for _, t := range mcp.Targets {
			targets = append(targets, domain.AgentType(t))
		}
		details := fmt.Sprintf("Команда: %s %s", mcp.Command, strings.Join(mcp.Args, " "))
		components = append(components, domain.AgentComponent{
			Type:        domain.ComponentMCP,
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
		components = append(components, domain.AgentComponent{
			Type:        domain.ComponentRule,
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
		components = append(components, domain.AgentComponent{
			Type:        domain.ComponentSkill,
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
		components = append(components, domain.AgentComponent{
			Type:        domain.ComponentWorkflow,
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
		components = append(components, domain.AgentComponent{
			Type:        domain.ComponentHook,
			Name:        hook.Header.Name,
			Description: hook.Header.Description,
			Path:        filepath.Join("hooks", hook.Header.Name),
			Targets:     hook.Header.Targets,
			Details:     "Скрипт автоматизации коммитов/задач (Hook)",
			Active:      true,
		})
	}

	// Загружаем Secrets
	secrets, _ := repository.LoadSecrets(repoPath)
	var secretKeys []string
	for k := range secrets {
		secretKeys = append(secretKeys, k)
	}
	sort.Strings(secretKeys)
	for _, k := range secretKeys {
		components = append(components, domain.AgentComponent{
			Type:        domain.ComponentSecret,
			Name:        k,
			Description: "Конфиденциальный токен авторизации API",
			Details:     secrets[k],
			Active:      true,
		})
	}

	sort.Slice(components, func(i, j int) bool {
		if components[i].Type != components[j].Type {
			return components[i].Type < components[j].Type
		}
		return components[i].Name < components[j].Name
	})

	var regItems []service.RegistryItem
	var loaded bool
	cached, err := service.LoadLocalRegistry()
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
	secValInput.EchoMode = textinput.EchoPassword
	secValInput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9"))

	return tuiModel{
		repoPath:              repoPath,
		activeTab:             tabDashboard,
		cursor:                0,
		scrollOffset:          0,
		agents:                agents,
		components:            components,
		activeCategory:        domain.ComponentMCP,
		categories:            []domain.ComponentType{domain.ComponentMCP, domain.ComponentRule, domain.ComponentSkill, domain.ComponentWorkflow, domain.ComponentHook, domain.ComponentSecret},
		registryItems:         regItems,
		registryLoaded:        loaded,
		activeInstallCategory: domain.ComponentMCP,
		searchInput:           ti,
		searchMode:            false,
		scanning:              false,
		p2pNodes:              []string{},
		logMsg:                "AgentSync готов к работе.",
		radarAngle:            0,
		lastActionTime:        time.Now().Format("15:04:05"),
		focus:                 focusComponents,
		secretNameInput:       secNameInput,
		secretValueInput:      secValInput,
	}
}

func (m tuiModel) getVisibleComponents() []domain.AgentComponent {
	var filtered []domain.AgentComponent
	for _, c := range m.components {
		if c.Type == m.activeCategory {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (m tuiModel) getVisibleRegistryItems() []service.RegistryItem {
	var filtered []service.RegistryItem
	q := strings.ToLower(m.searchInput.Value())
	for _, c := range m.registryItems {
		if c.Type == m.activeInstallCategory {
			if q == "" || strings.Contains(strings.ToLower(c.Name), q) || strings.Contains(strings.ToLower(c.Description), q) {
				filtered = append(filtered, c)
			}
		}
	}
	return filtered
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(syncRegistryCmd(), textinput.Blink)
}

type registryLoadedMsg []service.RegistryItem
type registrySyncFinishedMsg []service.RegistryItem

func syncRegistryCmd() tea.Cmd {
	return func() tea.Msg {
		_ = service.SyncLocalRegistry()
		items, err := service.LoadLocalRegistry()
		if err == nil {
			return registrySyncFinishedMsg(items)
		}
		return nil
	}
}

func fetchRegistryCmd(query string) tea.Cmd {
	return func() tea.Msg {
		var regItems []service.RegistryItem
		for _, c := range []domain.ComponentType{domain.ComponentMCP, domain.ComponentSkill} {
			items, _ := service.FetchRegistry(c, query)
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

func installCmd(item service.RegistryItem) tea.Cmd {
	return func() tea.Msg {
		err := service.InstallComponent(item)
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

	case registrySyncFinishedMsg:
		m.registryItems = msg
		m.registryLoaded = true
		return m, nil

	case registryLoadedMsg:
		m.registryItems = msg
		m.registryLoaded = true
		return m, nil

	case installResultMsg:
		if msg.err != nil {
			m.logMsg = fmt.Sprintf("Ошибка установки %s: %v", msg.name, msg.err)
		} else {
			m.logMsg = fmt.Sprintf("Успешно установлен компонент %s!", msg.name)
		}
		m.lastActionTime = time.Now().Format("15:04:05")
		return m, nil

	case tea.KeyMsg:
		if m.activeTab == tabMCP && m.activeCategory == domain.ComponentSecret {
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
					secrets, _ := repository.LoadSecrets(m.repoPath)
					
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
						err := repository.SaveSecrets(m.repoPath, secrets)
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
				m.registryLoaded = false
				return m, fetchRegistryCmd(m.searchInput.Value())
			}
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			m.activeTab = (m.activeTab + 1) % 4
			m.cursor = 0
			m.scrollOffset = 0
			m.focus = focusComponents
			m.logMsg = "Вкладка переключена."
			return m, nil

		case "shift+tab":
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

		case "right":
			if m.activeTab == tabMCP {
				if m.activeCategory == domain.ComponentSecret {
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

		case "left":
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

		case "]":
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

		case "[":
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

		case "d":
			m.logMsg = "Слияние и деплой запущены..."
			m.lastActionTime = time.Now().Format("15:04:05")
			go func() {
				deployService := service.NewDeployService(m.repoPath, nil)
				_ = deployService.Deploy("")
			}()
			m.logMsg = "Деплой успешно применен ко всем агентам!"
			return m, nil

		case "/", "s":
			if m.activeTab == tabInstall {
				m.searchMode = true
				m.searchInput.Focus()
				return m, textinput.Blink
			}
			if m.activeTab == tabAirDrop && msg.String() == "s" {
				m.scanning = true
				m.logMsg = "mDNS: Сканирование локального эфира..."
				m.p2pNodes = []string{}
				m.lastActionTime = time.Now().Format("15:04:05")
				return m, tea.Batch(
					tickCmd(),
					func() tea.Msg {
						nodes, _ := service.DiscoverServices(time.Second * 3)
						return nodes
					},
				)
			}
			return m, nil

		case "e":
			if m.activeTab == tabMCP && m.activeCategory == domain.ComponentSecret {
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

		case "n":
			if m.activeTab == tabMCP && m.activeCategory == domain.ComponentSecret {
				m.creatingSecret = true
				m.secretNameInput.Focus()
				m.secretNameInput.SetValue("")
				m.secretValueInput.SetValue("")
				m.logMsg = "Создание нового секрета. Введите имя, нажмите Tab, введите значение и нажмите Enter..."
				return m, nil
			}
			return m, nil

		case "delete":
			if m.activeTab == tabMCP && m.activeCategory == domain.ComponentSecret {
				visible := m.getVisibleComponents()
				if m.cursor >= 0 && m.cursor < len(visible) {
					targetName := visible[m.cursor].Name
					secrets, _ := repository.LoadSecrets(m.repoPath)
					delete(secrets, targetName)
					err := repository.SaveSecrets(m.repoPath, secrets)
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

		case "v":
			if m.activeTab == tabMCP && m.activeCategory == domain.ComponentSecret {
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

		case "esc":
			if m.activeTab == tabMCP && m.focus != focusComponents {
				m.focus = focusComponents
				m.logMsg = "Фокус возвращен к списку компонентов."
				return m, nil
			}
			return m, nil

		case " ":
			if m.activeTab == tabMCP {
				visible := m.getVisibleComponents()
				if m.cursor < 0 || m.cursor >= len(visible) {
					return m, nil
				}
				comp := visible[m.cursor]

				if m.focus == focusSources {
					err := service.SyncComponentSource(m.repoPath, comp, m.rightCursor == 1)
					if err != nil {
						m.logMsg = fmt.Sprintf("Ошибка синхронизации: %v", err)
					} else {
						m.logMsg = fmt.Sprintf("Успешно: скопировано для %s", comp.Name)
					}
					m.lastActionTime = time.Now().Format("15:04:05")
					return m, nil
				}

				if m.focus == focusTargets {
					agentsList := []domain.AgentType{domain.AgentClaudeCode, domain.AgentClaudeDesktop, domain.AgentAntigravity}
					targetAgent := agentsList[m.rightCursor]

					var newTargets []domain.AgentType
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

					store := repository.NewStore(m.repoPath)
					var err error
					switch comp.Type {
					case domain.ComponentMCP:
						err = store.UpdateMCPTargets(comp.Name, newTargets)
					case domain.ComponentRule:
						err = store.UpdateRuleTargets(comp.Name, newTargets)
					case domain.ComponentSkill:
						err = store.UpdateSkillTargets(comp.Name, newTargets)
					case domain.ComponentWorkflow:
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
				return m, nil
			}
		}
	}
	return m, nil
}
