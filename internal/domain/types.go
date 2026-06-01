package domain

// AgentType представляет поддерживаемых AI-агентов
type AgentType string

const (
	AgentClaudeCode    AgentType = "claude-code"
	AgentClaudeDesktop AgentType = "claude-desktop"
	AgentAntigravity   AgentType = "antigravity"
	AgentGeminiCLI     AgentType = "gemini-cli"
	AgentCodexCLI      AgentType = "codex-cli"
)

// ComponentType представляет тип компонента
type ComponentType string

const (
	ComponentMCP      ComponentType = "MCP"
	ComponentRule     ComponentType = "Rule"
	ComponentSkill    ComponentType = "Skill"
	ComponentWorkflow ComponentType = "Workflow"
	ComponentHook     ComponentType = "Hook"
	ComponentSecret   ComponentType = "Secrets"
)

// AgentComponent представляет один компонент
type AgentComponent struct {
	Type         ComponentType `json:"type"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Path         string        `json:"path"`        // Относительный путь к файлу (например, "skills/brainstorming/SKILL.md")
	Targets      []AgentType   `json:"targets"`     // Список целевых агентов
	Details      string        `json:"details,omitempty"` // Дополнительная техническая информация (команда, аргументы и т.д.)
	Active       bool          `json:"active"`      // Статус активности
	LocalExists  bool          `json:"local_exists"`
	GlobalExists bool          `json:"global_exists"`
}
