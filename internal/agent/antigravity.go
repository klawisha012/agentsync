package agent

import (
	"agentsync/internal/domain"
	"os"
	"path/filepath"
)

type AntigravityAdapter struct {
	homeDir string
}

func NewAntigravityAdapter() *AntigravityAdapter {
	home, _ := os.UserHomeDir()
	return &AntigravityAdapter{homeDir: home}
}

func (a *AntigravityAdapter) Name() domain.AgentType {
	return domain.AgentAntigravity
}

func (a *AntigravityAdapter) Detect() bool {
	configPath := filepath.Join(a.homeDir, ".gemini", "config", "mcp_config.json")
	_, err := os.Stat(configPath)
	return err == nil
}

func (a *AntigravityAdapter) DeployMCP(mcp domain.MCPConfig, secrets map[string]string) error {
	configPath := filepath.Join(a.homeDir, ".gemini", "config", "mcp_config.json")
	return deployMCPToPath(configPath, a.Name(), mcp, secrets)
}

func (a *AntigravityAdapter) ClearMCPs() error {
	configPath := filepath.Join(a.homeDir, ".gemini", "config", "mcp_config.json")
	return clearMCPsInPath(configPath, a.Name())
}

func (a *AntigravityAdapter) DeployRule(rule domain.Rule) error {
	rulesPath := filepath.Join(a.homeDir, ".gemini", "GEMINI.md")
	return deployRuleToPath(rulesPath, a.Name(), rule)
}
