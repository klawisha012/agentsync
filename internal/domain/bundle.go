package domain

// BundleItem описывает один компонент внутри бандла
type BundleItem struct {
	Type ComponentType `yaml:"type" json:"type"`
	Name string        `yaml:"name" json:"name"`
}

// ConfigBundle представляет бандл конфигурации
type ConfigBundle struct {
	ID          string       `yaml:"id" json:"id"`
	Name        string       `yaml:"name" json:"name"`
	Description string       `yaml:"description,omitempty" json:"description"`
	Components  []BundleItem `yaml:"components" json:"components"`
}
