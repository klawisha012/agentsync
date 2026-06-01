package domain

// Manifest описывает глобальный manifest.yaml в корне репозитория
type Manifest struct {
	ActiveAgents     []AgentType            `yaml:"active_agents" json:"active_agents"`
	ActiveBundle     string                 `yaml:"active_bundle,omitempty" json:"active_bundle"`
	SecretsPath      string                 `yaml:"secrets_path,omitempty" json:"secrets_path"`
	Overrides        map[string]string      `yaml:"overrides,omitempty" json:"overrides"`
	InstalledPlugins map[AgentType][]string `yaml:"installed_plugins,omitempty" json:"installed_plugins"`
}
