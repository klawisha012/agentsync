# Спецификация дизайна: Agent Manager TUI

Этот документ описывает изменения для превращения вкладки `MCP Manager` в полноценный `Agent Manager` в консольном интерфейсе (TUI) AgentSync. Это позволит пользователям отслеживать и синхронизировать не только MCP-серверы, но и правила (Rules), навыки (Skills), рабочие процессы (Workflows) и хуки (Hooks).

## 1. Цели
* Переименовать вкладку в TUI во вторую позицию в `💼 AGENT MANAGER`.
* Обеспечить единый плосный список всех синхронизируемых компонентов (MCP, Rules, Skills, Workflows, Hooks).
* Разработать цветовую и иконочную кодировку для разных типов компонентов для создания премиального визуального стиля.
* Расширить загрузку ресурсов в `store.go`, чтобы динамически считывать Skills из `skills/`, Workflows из `workflows/` и Hooks из `hooks/`.
* Сохранить обратную совместимость и устойчивость к отсутствию папок в репозитории.

## 2. Предлагаемые изменения

### 2.1 Модели данных (`types.go`)
Мы добавляем структуры для универсального представления компонентов `AgentComponent` и типы компонентов:

```go
type ComponentType string

const (
	ComponentMCP      ComponentType = "MCP"
	ComponentRule     ComponentType = "Rule"
	ComponentSkill    ComponentType = "Skill"
	ComponentWorkflow ComponentType = "Workflow"
	ComponentHook     ComponentType = "Hook"
)

type AgentComponent struct {
	Type        ComponentType
	Name        string
	Description string
	Path        string      // Относительный путь к файлу
	Targets     []AgentType // Список целевых агентов
	Details     string      // Дополнительная техническая информация
}
```

### 2.2 Загрузка ресурсов (`store.go`)
Мы добавим новые методы в `Store` для считывания дополнительных папок:

* **LoadSkills()**: сканирует директорию `skills/` рекурсивно на файлы `SKILL.md` или `*.md`. Парсит фронтматтер, возвращает список компонентов со статусом `ComponentSkill`.
* **LoadWorkflows()**: сканирует `workflows/` на наличие `.yaml`, `.yml` или `.json`. Возвращает список `ComponentWorkflow`.
* **LoadHooks()**: сканирует `hooks/` на наличие исполняемых файлов или скриптов. Возвращает список `ComponentHook`.

Если папки `skills/`, `workflows/` или `hooks/` отсутствуют, методы будут возвращать пустые слайсы и логировать это без падения программы.

### 2.3 Интерфейс TUI (`tui.go`)
В `tui.go` мы:
* Переименуем вкладку `tabMCP` во внутреннем перечислении и в `tabNames` в `"  💼 AGENT MANAGER  "`.
* Добавим стили Lipgloss для подсветки активного элемента в зависимости от его типа:
  * MCP: бирюзовый (`#00F0FF`)
  * Rule: зеленый (`#00FF87`)
  * Skill: фиолетовый (`#BD93F9`)
  * Workflow: оранжевый (`#FFB86C`)
  * Hook: розово-красный (`#FF5555`)
* Перепишем логику отрисовки карточек во второй вкладке для поддержки `AgentComponent`.
* Сохраним скроллинг и автовычисление высоты.

## 3. План верификации

### Ручное тестирование
1. Запустить `agentsync init` для создания тестовой структуры (включая `rules/react-style.md` и `mcp/github.yaml`).
2. Вручную создать тестовые папки и файлы:
   * `skills/brainstorming/SKILL.md` (с тестовым текстом и фронтматтером)
   * `workflows/deploy-flow.yaml`
   * `hooks/pre-commit.sh`
3. Запустить `agentsync tui` и переключиться на вторую вкладку.
4. Проверить корректность скроллинга, наличие иконок, подсветку рамок карточек при выборе элементов разного типа.
