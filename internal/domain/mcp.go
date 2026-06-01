package domain

// MCPConfig представляет каноническое описание MCP-сервера в YAML
type MCPConfig struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Runtime     string            `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Package     string            `yaml:"package,omitempty" json:"package,omitempty"`
	Command     string            `yaml:"command,omitempty" json:"command,omitempty"`
	Args        []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	ServerURL   string            `yaml:"server_url,omitempty" json:"serverUrl,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Targets     []AgentType       `yaml:"targets" json:"targets"`
	Secret      bool              `yaml:"secret,omitempty" json:"secret,omitempty"`
}

// RuleFrontmatter представляет фронтматтер канонического правила в Markdown
type RuleFrontmatter struct {
	Name        string      `yaml:"name" json:"name"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Targets     []AgentType `yaml:"targets" json:"targets"`
	Scope       string      `yaml:"scope,omitempty" json:"scope,omitempty"` // global или project
}

// Rule представляет каноническое правило
type Rule struct {
	Header  RuleFrontmatter `json:"header"`
	Content string          `json:"content"` // Само тело Markdown-правила без фронтматтера
}
