package main

// AgentType представляет поддерживаемых AI-агентов
type AgentType string

const (
	AgentClaudeCode    AgentType = "claude-code"
	AgentClaudeDesktop AgentType = "claude-desktop"
	AgentAntigravity   AgentType = "antigravity"
	AgentGeminiCLI     AgentType = "gemini-cli"
	AgentCodexCLI      AgentType = "codex-cli"
)

// Manifest описывает глобальный manifest.yaml в корне репозитория
type Manifest struct {
	ActiveAgents     []AgentType            `yaml:"active_agents" json:"active_agents"`
	ActiveBundle     string                 `yaml:"active_bundle,omitempty" json:"active_bundle"`
	SecretsPath      string                 `yaml:"secrets_path,omitempty" json:"secrets_path"`
	Overrides        map[string]string      `yaml:"overrides,omitempty" json:"overrides"`
	InstalledPlugins map[AgentType][]string `yaml:"installed_plugins,omitempty" json:"installed_plugins"`
}

// MCPConfig представляет каноническое описание MCP-сервера в YAML
type MCPConfig struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Runtime     string            `yaml:"runtime,omitempty"`
	Package     string            `yaml:"package,omitempty"`
	Command     string            `yaml:"command,omitempty"`
	Args        []string          `yaml:"args,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	ServerURL   string            `yaml:"server_url,omitempty" json:"serverUrl,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Targets     []AgentType       `yaml:"targets"`
	Secret      bool              `yaml:"secret,omitempty"`
}

// RuleFrontmatter представляет фронтматтер канонического правила в Markdown
type RuleFrontmatter struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Targets     []AgentType `yaml:"targets"`
	Scope       string      `yaml:"scope,omitempty"` // global или project
}

// Rule представляет каноническое правило
type Rule struct {
	Header  RuleFrontmatter
	Content string // Само тело Markdown-правила без фронтматтера
}

// Adapter определяет интерфейс для взаимодействия с конкретным агентом
type Adapter interface {
	Name() AgentType
	Detect() bool
	DeployMCP(mcp MCPConfig, secrets map[string]string) error
	DeployRule(rule Rule) error
	ClearMCPs() error
}

type ComponentType string

const (
	ComponentMCP      ComponentType = "MCP"
	ComponentRule     ComponentType = "Rule"
	ComponentSkill    ComponentType = "Skill"
	ComponentWorkflow ComponentType = "Workflow"
	ComponentHook     ComponentType = "Hook"
	ComponentSecret   ComponentType = "Secrets"
)

type AgentComponent struct {
	Type        ComponentType
	Name        string
	Description string
	Path        string      // Относительный путь к файлу (например, "skills/brainstorming/SKILL.md")
	Targets     []AgentType // Список целевых агентов
	Details     string      // Дополнительная техническая информация (команда, аргументы и т.д.)
	Active      bool        // Статус активности
}

