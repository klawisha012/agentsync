# План Реализации Рефакторинга и Интеграции Репозиториев AgentSync

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Инициализировать локальный Git и удаленный публичный GitHub-репозиторий, а затем реорганизовать кодовую базу AgentSync в модульную сервисно-слоистую структуру (internal/) с полной развязкой домена, логики деплоя и интерфейсов представления.

**Architecture:** Переход от монолитного package main к разделенной слоистой архитектуре: Domain -> Repository -> Agent Adapters -> Business Services -> UI Controllers (TUI/Web). Связывание через явные интерфейсы (AgentAdapter, Logger).

**Tech Stack:** Go (1.25+), Bubbletea (Lipgloss), SolidJS, GitHub API (GitHub MCP).

---

## План Задач

### Task 1: Инициализация Git и Создание Репозитория на GitHub

**Files:**
- Create: `.gitignore`
- Create: `README.md`

- [ ] **Step 1: Создание корневого `.gitignore`**
  Записать файл в корне проекта:
  ```ini
  # --- Специфичные для Go файлы и бинарники ---
  *.exe
  *.exe~
  *.dll
  *.so
  *.dylib
  agentsync
  agentsync.exe
  agentsync.exe~

  # --- Безопасность и Секреты ---
  secrets.env
  *.env
  *.pem
  *.key

  # --- Инструменты ИИ и базы знаний (AI & Agents) ---
  .superpowers/
  graphify-out/
  docs/
  !docs/superpowers/specs/
  !docs/superpowers/plans/

  # --- Фронтенд (SolidJS/Vite) ---
  frontend/node_modules/
  frontend/dist/
  frontend/dist-ssr/
  frontend/.cache/
  frontend/*.local

  # --- Рантаймы и Портативные пакеты ---
  data/runtimes/
  data/packages/
  node_modules/

  # --- Системные файлы и IDE ---
  .idea/
  .vscode/
  *.suo
  *.ntvs*
  *.njsproj
  *.sln
  *.sw?
  .DS_Store
  Thumbs.db
  ```

- [ ] **Step 2: Инициализировать локальный репозиторий Git**
  Запустить:
  `git init`

- [ ] **Step 3: Создать удаленный репозиторий на GitHub**
  Вызвать инструмент `github.create_repository` через MCP-сервер `github` с параметрами:
  - `name`: "agentsync"
  - `description`: "Canonical configuration & rules manager for AI agents (Claude Code, Claude Desktop, Antigravity)"
  - `private`: false

- [ ] **Step 4: Связать локальный гит с удаленным, сделать первый коммит и отправить**
  Выполнить последовательно в терминале:
  `git remote add origin https://github.com/zward/agentsync.git` (или URL, полученный из Step 3)
  `git branch -M main`
  `git add .`
  `git commit -m "initial: canonical agentsync structure with specs and ignore patterns"`
  `git push -u origin main`

---

### Task 2: Создание Директорий и Разделение Доменных Моделей (Domain)

**Files:**
- Create: `internal/domain/types.go`
- Create: `internal/domain/manifest.go`
- Create: `internal/domain/mcp.go`
- Create: `internal/domain/bundle.go`
- Modify: `types.go` (удалить после миграции)

- [ ] **Step 1: Создание `internal/domain/types.go`**
  Выделить базовые константы и перечисления:
  ```go
  package domain

  type AgentType string
  type ComponentType string

  const (
  	AgentClaudeCode    AgentType = "claude-code"
  	AgentClaudeDesktop AgentType = "claude-desktop"
  	AgentAntigravity   AgentType = "antigravity"
  )

  const (
  	ComponentMCP      ComponentType = "MCP"
  	ComponentRule     ComponentType = "Rule"
  	ComponentSkill    ComponentType = "Skill"
  	ComponentWorkflow ComponentType = "Workflow"
  	ComponentHook     ComponentType = "Hook"
  )

  type AgentComponent struct {
  	Type        ComponentType `json:"type"`
  	Name        string        `json:"name"`
  	Description string        `json:"description"`
  	Path        string        `json:"path"`
  	Targets     []AgentType   `json:"targets"`
  	Details     string        `json:"details,omitempty"`
  	Active      bool          `json:"active"`
  }
  ```

- [ ] **Step 2: Создание `internal/domain/manifest.go`**
  ```go
  package domain

  type Manifest struct {
  	ActiveAgents     []AgentType            `yaml:"active_agents" json:"active_agents"`
  	ActiveBundle     string                 `yaml:"active_bundle" json:"active_bundle"`
  	InstalledPlugins map[AgentType][]string `yaml:"installed_plugins,omitempty" json:"installed_plugins,omitempty"`
  }
  ```

- [ ] **Step 3: Создание `internal/domain/mcp.go`**
  ```go
  package domain

  type MCPConfig struct {
  	Name        string            `yaml:"name" json:"name"`
  	Description string            `yaml:"description" json:"description"`
  	Command     string            `yaml:"command" json:"command"`
  	Args        []string          `yaml:"args" json:"args"`
  	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
  	Runtime     string            `yaml:"runtime,omitempty" json:"runtime,omitempty"`
  	Package     string            `yaml:"package,omitempty" json:"package,omitempty"`
  	ServerURL   string            `yaml:"server_url,omitempty" json:"server_url,omitempty"`
  	Targets     []string          `yaml:"targets" json:"targets"`
  }
  ```

- [ ] **Step 4: Создание `internal/domain/bundle.go`**
  ```go
  package domain

  type BundleItem struct {
  	Type ComponentType `json:"type"`
  	Name string        `json:"name"`
  }

  type ConfigBundle struct {
  	ID          string       `json:"id"`
  	Name        string       `json:"name"`
  	Description string       `json:"description"`
  	Components  []BundleItem `json:"components"`
  }
  ```

- [ ] **Step 5: Коммит доменного слоя**
  `git add internal/domain/*.go`
  `git commit -m "feat: setup domain layer models"`

---

### Task 3: Портирование Слоя Хранилища (Repository)

**Files:**
- Create: `internal/repository/store.go`
- Create: `internal/repository/secrets.go`
- Create: `internal/repository/backup.go`

- [ ] **Step 1: Создание `internal/repository/store.go`**
  Перенести методы загрузки и сохранения файлов из `store.go`, адаптировав их под пакет `domain`:
  ```go
  package repository

  import (
  	"agentsync/internal/domain"
  	"os"
  	"path/filepath"
  	"gopkg.in/yaml.v3"
  )

  type Store struct {
  	repoPath string
  }

  func NewStore(repoPath string) *Store {
  	return &Store{repoPath: repoPath}
  }

  func (s *Store) LoadManifest() (*domain.Manifest, error) {
  	path := filepath.Join(s.repoPath, "manifest.yaml")
  	data, err := os.ReadFile(path)
  	if err != nil {
  		return nil, err
  	}
  	var m domain.Manifest
  	if err := yaml.Unmarshal(data, &m); err != nil {
  		return nil, err
  	}
  	return &m, nil
  }
  // Добавить методы SaveManifest, LoadMCPs, LoadRules, LoadBundles, SaveBundle, DeleteBundle
  ```

- [ ] **Step 2: Создание `internal/repository/secrets.go`**
  Перенести функции загрузки/сохранения локальных секретов (`secrets.go` -> `internal/repository/secrets.go`).

- [ ] **Step 3: Создание `internal/repository/backup.go`**
  Перенести функции создания бэкапов (`backup.go` -> `internal/repository/backup.go`).

- [ ] **Step 4: Написание модульного теста хранилища**
  Создать `internal/repository/store_test.go` для проверки чтения манифеста.

- [ ] **Step 5: Проверка и коммит слоя Repository**
  `go test ./internal/repository/...`
  `git add internal/repository/*.go`
  `git commit -m "feat: implement repository layer and tests"`

---

### Task 4: Разработка Интерфейса и Адаптеров Агентов (Agent Adapters)

**Files:**
- Create: `internal/agent/adapter.go`
- Create: `internal/agent/claude_desktop.go`
- Create: `internal/agent/claude_code.go`
- Create: `internal/agent/antigravity.go`

- [ ] **Step 1: Создание `internal/agent/adapter.go`**
  Определить общий интерфейс адаптеров ИИ-агентов (согласно Разделу 3 спецификации).

- [ ] **Step 2: Перенос адаптера Claude Desktop в `internal/agent/claude_desktop.go`**
  Портировать функции `NewClaudeDesktopAdapter()` и его методы деплоя MCP и правил.

- [ ] **Step 3: Перенос адаптера Claude Code в `internal/agent/claude_code.go`**
  Портировать функции `NewClaudeCodeAdapter()`.

- [ ] **Step 4: Перенос адаптера Antigravity в `internal/agent/antigravity.go`**
  Портировать функции `NewAntigravityAdapter()`.

- [ ] **Step 5: Проверка компиляции адаптеров и коммит**
  `go build ./internal/agent/...`
  `git add internal/agent/*.go`
  `git commit -m "feat: implement agent integration adapters"`

---

### Task 5: Создание Бизнес-Сервисов (Service Layer)

**Files:**
- Create: `internal/service/deploy.go`
- Create: `internal/service/sync.go`
- Create: `internal/service/registry.go`
- Create: `internal/service/runtime.go`

- [ ] **Step 1: Создание сервиса логирования деплоя и `internal/service/deploy.go`**
  Реализовать конвейер `DeployService` (подстановка секретов, фильтрация совместимости, запуск адаптеров) независимо от вывода в консоль.

- [ ] **Step 2: Портирование P2P синхронизации в `internal/service/sync.go`**
  Перенести функции MDNS поиска, шаринга и пулла из `p2p.go`.

- [ ] **Step 3: Портирование Smithery реестра в `internal/service/registry.go`**
  Портировать функции работы с внешними MCP-хабами и реестрами.

- [ ] **Step 4: Портирование рантаймов в `internal/service/runtime.go`**
  Портировать менеджер портативных сред `RuntimeManager` (Node.js/Python).

- [ ] **Step 5: Написание Unit-теста деплоя и коммит**
  Создать тест в `/internal/service/deploy_test.go`, проверяющий слияние через Mock-логгер.
  `go test ./internal/service/...`
  `git add internal/service/*.go`
  `git commit -m "feat: setup business services layer and deploy tests"`

---

### Task 6: Перенос Пользовательских Интерфейсов (UI Layer)

**Files:**
- Create: `internal/ui/web/server.go`
- Create: `internal/ui/web/handlers.go`
- Create: `internal/ui/tui/model.go`
- Create: `internal/ui/tui/styles.go`

- [ ] **Step 1: Создание Web-сервера в `internal/ui/web/`**
  Портировать функции `StartWebServer()` и эндпоинты API из `web.go`, ссылаясь на сервисный и репозиторный слои в `internal/`.

- [ ] **Step 2: Создание TUI Bubbletea модели в `internal/ui/tui/`**
  Перенести массивный Lipgloss/Bubbletea код из `tui.go` в `/internal/ui/tui/model.go` и `styles.go`.

- [ ] **Step 3: Коммит слоев UI**
  `git add internal/ui/...`
  `git commit -m "feat: migrate Web and TUI layers into internal packages"`

---

### Task 7: Финализация CLI Загрузчика и Удаление Монолитного Кода

**Files:**
- Modify: `cmd/agentsync/main.go`
- Delete: `main.go`
- Delete: `adapters.go`
- Delete: `store.go`
- Delete: `web.go`
- Delete: `tui.go`
- Delete: `p2p.go`
- Delete: `registry.go`
- Delete: `runtime.go`
- Delete: `backup.go`
- Delete: `secrets.go`

- [ ] **Step 1: Обновление главного CLI-файла в `cmd/agentsync/main.go`**
  Записать чистый CLI-парсер, связывающий все слои:
  ```go
  package main

  import (
  	"agentsync/internal/ui/tui"
  	"agentsync/internal/ui/web"
  	"os"
  	"fmt"
  )

  func main() {
  	// CLI маршрутизация (init, deploy, share, web, tui...)
  }
  ```

- [ ] **Step 2: Удаление старых плоских файлов Go в корне репозитория**
  Удалить `main.go`, `adapters.go`, `store.go`, `web.go`, `tui.go` и другие перенесенные файлы, чтобы очистить корень проекта.

- [ ] **Step 3: Запуск итоговых автотестов приложения**
  Выполнить:
  `go test ./...`
  Ожидается: PASS.

- [ ] **Step 4: Финальный коммит структуры**
  `git add .`
  `git commit -m "refactor: complete lean modular transition, clean root directory"`
  `git push origin main`
