package agent

import "agentsync/internal/domain"

// AgentAdapter определяет интерфейс для взаимодействия с конкретным ИИ-агентом
type AgentAdapter interface {
	// Name возвращает тип агента
	Name() domain.AgentType
	
	// Detect проверяет, установлен ли агент в системе
	Detect() bool
	
	// DeployMCP интегрирует конфигурацию MCP-сервера для агента
	DeployMCP(mcp domain.MCPConfig, secrets map[string]string) error
	
	// DeployRule записывает правила (Rules) для агента
	DeployRule(rule domain.Rule) error
	
	// ClearMCPs очищает старые канонические MCP-серверы
	ClearMCPs() error
}
