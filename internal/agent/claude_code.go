package agent

import (
	"agentsync/internal/domain"
	"os"
	"path/filepath"
)

type ClaudeCodeAdapter struct {
	homeDir string
}

func NewClaudeCodeAdapter() *ClaudeCodeAdapter {
	home, _ := os.UserHomeDir()
	return &ClaudeCodeAdapter{homeDir: home}
}

func (a *ClaudeCodeAdapter) Name() domain.AgentType {
	return domain.AgentClaudeCode
}

func (a *ClaudeCodeAdapter) Detect() bool {
	configPath := filepath.Join(a.homeDir, ".claude.json")
	_, err := os.Stat(configPath)
	return err == nil
}

func (a *ClaudeCodeAdapter) DeployMCP(mcp domain.MCPConfig, secrets map[string]string) error {
	configPath := filepath.Join(a.homeDir, ".claude.json")
	return deployMCPToPath(configPath, a.Name(), mcp, secrets)
}

func (a *ClaudeCodeAdapter) ClearMCPs() error {
	configPath := filepath.Join(a.homeDir, ".claude.json")
	return clearMCPsInPath(configPath, a.Name())
}

func (a *ClaudeCodeAdapter) DeployRule(rule domain.Rule) error {
	rulesPath := filepath.Join(a.homeDir, ".claude", "CLAUDE.md")
	return deployRuleToPath(rulesPath, a.Name(), rule)
}
