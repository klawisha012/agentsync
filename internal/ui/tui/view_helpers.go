package tui

import (
	"agentsync/internal/domain"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m tuiModel) View() string {
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("⚡ AGENTSYNC TUI MANAGER v0.1.0 ⚡"))
	sb.WriteString("\n\n")

	// Рендерим табы
	tabs := []string{"[1] Dashboard", "[2] Agent Manager", "[3] Discover registry", "[4] AirDrop sync"}
	for i, t := range tabs {
		if tabId(i) == m.activeTab {
			sb.WriteString(styleActiveTab.Render(t))
		} else {
			sb.WriteString(styleTab.Render(t))
		}
	}
	sb.WriteString("\n\n")

	switch m.activeTab {
	case tabDashboard:
		sb.WriteString(m.viewDashboard())
	case tabMCP:
		sb.WriteString(m.viewAgentManager())
	case tabInstall:
		sb.WriteString(m.viewRegistry())
	case tabAirDrop:
		sb.WriteString(m.viewAirDrop())
	}

	// Статус бар внизу
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false, false, false).BorderForeground(lipgloss.Color("#3A3F58")).Width(85).Render(
		fmt.Sprintf("Лог [%s]: %s\nГорячие клавиши: [Tab/Shift+Tab] Вкладки | [d] Деплой Smart Merge | [q] Выход", m.lastActionTime, m.logMsg),
	))

	return sb.String()
}

func (m tuiModel) viewDashboard() string {
	var sb strings.Builder
	sb.WriteString(styleCyan.Render("=== АКТИВНЫЕ ИИ-АГЕНТЫ В СИСТЕМЕ ===\n\n"))

	for _, a := range m.agents {
		status := styleRed.Render("✘ Не найден")
		if a.Detected {
			status = styleGreen.Render("✔ Обнаружен")
		}
		sb.WriteString(fmt.Sprintf("- %s: %s\n", a.Name, status))
		for _, path := range a.ConfigPaths {
			sb.WriteString(styleGray.Render(fmt.Sprintf("  Путь: %s\n", path)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m tuiModel) viewAgentManager() string {
	var sb strings.Builder

	// Переключатель подкатегорий
	sb.WriteString("Подкатегории: ")
	for _, cat := range m.categories {
		if cat == m.activeCategory {
			sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#BD93F9")).Render(fmt.Sprintf("[%s] ", string(cat))))
		} else {
			sb.WriteString(fmt.Sprintf("%s ", string(cat)))
		}
	}
	sb.WriteString("\n\n")

	// Логика отображения для Секретов
	if m.activeCategory == domain.ComponentSecret {
		if m.editingSecret || m.creatingSecret {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Bold(true).Render("=== РЕДАКТИРОВАНИЕ ЛОКАЛЬНОГО СЕКРЕТА ===\n\n"))
			if m.creatingSecret {
				sb.WriteString(fmt.Sprintf("Имя:   %s\n", m.secretNameInput.View()))
			} else {
				visible := m.getVisibleComponents()
				if m.cursor >= 0 && m.cursor < len(visible) {
					sb.WriteString(fmt.Sprintf("Имя:   %s (только чтение)\n", visible[m.cursor].Name))
				}
			}
			sb.WriteString(fmt.Sprintf("Токен: %s\n\n", m.secretValueInput.View()))
			sb.WriteString(styleGray.Render("Нажмите Enter для сохранения | Esc для отмены\n"))
			return sb.String()
		}

		sb.WriteString(styleCyan.Render("=== ЛОКАЛЬНЫЕ СЕКРЕТЫ (secrets.env) ===\n"))
		sb.WriteString(styleGray.Render("Безопасно подставляются в MCP окружение. Никогда не коммитятся в Git.\n\n"))
		visible := m.getVisibleComponents()
		if len(visible) == 0 {
			sb.WriteString("Секреты не заданы. Нажмите [n] для добавления.\n")
		} else {
			for i, comp := range visible {
				marker := "  "
				if i == m.cursor {
					marker = styleCyan.Render("> ")
				}

				maskedVal := "••••••••••••••••"
				if m.showSecretValue {
					maskedVal = comp.Details
				}

				sb.WriteString(fmt.Sprintf("%s%s = %s\n", marker, comp.Name, styleGreen.Render(maskedVal)))
			}
		}
		sb.WriteString("\n" + styleGray.Render("[n] Новый секрет | [e] Изменить | [Delete] Удалить | [v] Показать/скрыть значения") + "\n")
		return sb.String()
	}

	// Обычный рендеринг компонентов
	visible := m.getVisibleComponents()
	if len(visible) == 0 {
		return "Нет канонических компонентов в данной категории.\n"
	}

	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0FF")).Render(fmt.Sprintf("Доступно компонентов: %d (скролл стрелками)\n\n", len(visible))))

	// Левая колонка (Компоненты)
	var leftRows []string
	endIdx := m.scrollOffset + maxVisibleCards
	if endIdx > len(visible) {
		endIdx = len(visible)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		comp := visible[i]
		isActive := i == m.cursor && m.focus == focusComponents
		cardStyle := styleCard
		if isActive {
			cardStyle = styleCardActive
		}

		cardContent := fmt.Sprintf(
			"%s\n%s\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF")).Render("⚡ "+comp.Name),
			lipgloss.NewStyle().Width(35).Render(comp.Description),
			styleGray.Render(comp.Path),
		)
		leftRows = append(leftRows, cardStyle.Render(cardContent))
	}
	leftPanel := lipgloss.JoinVertical(lipgloss.Left, leftRows...)

	// Правая колонка
	var rightPanel string
	if m.cursor >= 0 && m.cursor < len(visible) {
		comp := visible[m.cursor]

		home, _ := os.UserHomeDir()
		var subDir string
		var fileName string
		switch comp.Type {
		case domain.ComponentMCP:
			subDir = "mcp"
			fileName = comp.Name + ".yaml"
		case domain.ComponentRule:
			subDir = "rules"
			fileName = comp.Name + ".md"
		case domain.ComponentSkill:
			subDir = "skills"
			fileName = filepath.Join(comp.Name, "SKILL.md")
		case domain.ComponentWorkflow:
			subDir = "workflows"
			fileName = comp.Name
		case domain.ComponentHook:
			subDir = "hooks"
			fileName = comp.Name
		}

		localExists := fileExistsTui(filepath.Join(m.repoPath, subDir, fileName))
		globalExists := fileExistsTui(filepath.Join(home, ".agents", subDir, fileName))

		localMark := styleRed.Render("[ ] Локально (CWD)")
		if localExists {
			localMark = styleGreen.Render("[✔] Локально (CWD)")
		}
		globalMark := styleRed.Render("[ ] Глобально (~/.agents)")
		if globalExists {
			globalMark = styleGreen.Render("[✔] Глобально (~/.agents)")
		}

		if m.focus == focusSources {
			if m.rightCursor == 0 {
				localMark = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF")).Render("> ") + localMark
			} else {
				globalMark = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF")).Render("> ") + globalMark
			}
		}

		sourceBlock := fmt.Sprintf(
			"Источники (пробел для копирования):\n%s\n%s",
			localMark,
			globalMark,
		)

		var targetRows []string
		agentsList := []domain.AgentType{domain.AgentClaudeCode, domain.AgentClaudeDesktop, domain.AgentAntigravity}

		for idx, a := range agentsList {
			checked := "[ ]"
			for _, t := range comp.Targets {
				if t == a {
					checked = "[✔]"
					break
				}
			}

			row := fmt.Sprintf("%s %s", checked, string(a))
			if checked == "[✔]" {
				row = styleGreen.Render(row)
			} else {
				row = styleRed.Render(row)
			}

			if m.focus == focusTargets && idx == m.rightCursor {
				row = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF")).Render("> ") + row
			}
			targetRows = append(targetRows, row)
		}

		targetBlock := fmt.Sprintf(
			"Направления деплоя (пробел для переключения):\n%s",
			strings.Join(targetRows, "\n"),
		)

		detailsStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3A3F58")).Width(38).Padding(0, 1)
		rightPanel = detailsStyle.Render(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			styleCyan.Render("ДЕТАЛИ КОМПОНЕНТА"),
			sourceBlock,
			targetBlock,
		))
	}

	mainBlock := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "   ", rightPanel)
	return mainBlock + "\n\n" + styleGray.Render("Навигация: стрелки [Вверх/Вниз] - скролл | [Left/Right] - перемещение фокуса панелей | Клавиши [ и ] - смена категорий") + "\n"
}

func (m tuiModel) viewRegistry() string {
	var sb strings.Builder

	sb.WriteString("Поиск в удаленных каталогах (Smithery/GitHub): ")
	if m.searchMode {
		sb.WriteString(m.searchInput.View())
	} else {
		if m.searchInput.Value() != "" {
			sb.WriteString(lipgloss.NewStyle().Underline(true).Render(m.searchInput.Value()))
		} else {
			sb.WriteString(styleGray.Render("Нажмите [/] для поиска в реестре"))
		}
	}
	sb.WriteString("\n\n")

	sb.WriteString("Реестр: ")
	for _, cat := range []domain.ComponentType{domain.ComponentMCP, domain.ComponentSkill} {
		if cat == m.activeInstallCategory {
			sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF")).Render(fmt.Sprintf("[%s] ", string(cat))))
		} else {
			sb.WriteString(fmt.Sprintf("%s ", string(cat)))
		}
	}
	sb.WriteString("\n\n")

	if !m.registryLoaded {
		return "Поиск на серверах Smithery и GitHub (сканирование репозиториев)... [Пожалуйста, подождите]\n"
	}

	visible := m.getVisibleRegistryItems()
	if len(visible) == 0 {
		return "По вашему запросу ничего не найдено в реестре.\n"
	}

	for i, item := range visible {
		marker := "  "
		if i == m.cursor {
			marker = styleCyan.Render("> ")
		}

		status := styleGray.Render("[Доступно для скачивания]")
		if item.Installed {
			status = styleGreen.Render("[Установлено локально]")
		}

		sb.WriteString(fmt.Sprintf("%s%s - %s\n", marker, styleCyan.Render(item.Name), status))
		sb.WriteString(fmt.Sprintf("  %s\n", item.Description))
		sb.WriteString(styleGray.Render(fmt.Sprintf("  Источник: %s\n\n", item.SourceURL)))
	}

	return sb.String()
}

func (m tuiModel) viewAirDrop() string {
	var sb strings.Builder
	sb.WriteString(styleCyan.Render("=== P2P AIRDROP: СИНХРОНИЗАЦИЯ КОНФИГУРАЦИЙ ===\n"))
	sb.WriteString(styleGray.Render("Позволяет мгновенно обмениваться правилами и MCP с компьютерами коллег в локальной сети.\n\n"))

	if m.scanning {
		radarFrame := "/-\\|"[m.radarAngle/90%4 : m.radarAngle/90%4+1]
		sb.WriteString(fmt.Sprintf("Радар сканирования локальной сети %s [mDNS поиск активирован]...\n\n", styleCyan.Render(radarFrame)))
		return sb.String()
	}

	sb.WriteString("Сканирование завершено. Найденные AgentSync ноды:\n")
	if len(m.p2pNodes) == 0 {
		sb.WriteString("   Ноды не обнаружены. Попробуйте еще раз.\n")
	} else {
		for i, node := range m.p2pNodes {
			marker := "  "
			if i == m.cursor {
				marker = styleCyan.Render("> ")
			}
			sb.WriteString(fmt.Sprintf("%s%s\n", marker, node))
		}
	}

	sb.WriteString("\n" + styleGray.Render("Горячие клавиши: [s] Запустить сканирование эфира | [d] Запустить HTTP AirDrop раздачу") + "\n")
	return sb.String()
}

func updateTuiSecrets(components []domain.AgentComponent, secrets map[string]string) []domain.AgentComponent {
	var filtered []domain.AgentComponent
	for _, c := range components {
		if c.Type != domain.ComponentSecret {
			filtered = append(filtered, c)
		}
	}

	var secretKeys []string
	for k := range secrets {
		secretKeys = append(secretKeys, k)
	}
	sort.Strings(secretKeys)

	for _, k := range secretKeys {
		filtered = append(filtered, domain.AgentComponent{
			Type:        domain.ComponentSecret,
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

func fileExistsTui(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
