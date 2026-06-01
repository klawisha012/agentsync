package repository

import (
	"agentsync/internal/domain"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Store управляет каноническим хранилищем (репозиторием)
type Store struct {
	RepoPath      string
	DisableGlobal bool
}

func NewStore(repoPath string) *Store {
	return &Store{RepoPath: repoPath}
}

// getSearchDirs возвращает пути для поиска компонентов (локальный репозиторий + глобальная папка ~/.agents)
func (s *Store) getSearchDirs(subDir string) []string {
	dirs := []string{filepath.Join(s.RepoPath, "data", subDir)}
	if !s.DisableGlobal {
		home, err := os.UserHomeDir()
		if err == nil {
			dirs = append(dirs, filepath.Join(home, ".agents", subDir))
		}
	}
	return dirs
}

// LoadManifest загружает глобальный manifest.yaml
func (s *Store) LoadManifest() (domain.Manifest, error) {
	manifestPath := filepath.Join(s.RepoPath, "data", "manifest.yaml")
	var manifest domain.Manifest

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Возвращаем дефолтный манифест
			return domain.Manifest{
				ActiveAgents: []domain.AgentType{domain.AgentClaudeCode, domain.AgentClaudeDesktop, domain.AgentAntigravity},
			}, nil
		}
		return manifest, err
	}

	err = yaml.Unmarshal(data, &manifest)
	return manifest, err
}

// SaveManifest сохраняет manifest.yaml
func (s *Store) SaveManifest(manifest domain.Manifest) error {
	manifestPath := filepath.Join(s.RepoPath, "data", "manifest.yaml")
	data, err := yaml.Marshal(&manifest)
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, data, 0644)
}

// LoadMCPs загружает все канонические MCP-серверы из mcp/*.yaml
func (s *Store) LoadMCPs() ([]domain.MCPConfig, error) {
	var configs []domain.MCPConfig
	seen := make(map[string]bool)

	dirs := s.getSearchDirs("mcp")
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var mcp domain.MCPConfig
			if err := yaml.Unmarshal(data, &mcp); err != nil {
				continue
			}

			// Если имя не задано, берем имя файла без расширения
			if mcp.Name == "" {
				mcp.Name = strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".yaml"), ".yml")
			}

			if !seen[mcp.Name] {
				configs = append(configs, mcp)
				seen[mcp.Name] = true
			}
		}
	}

	return configs, nil
}

// LoadRules загружает все канонические правила из rules/*.md
func (s *Store) LoadRules() ([]domain.Rule, error) {
	var rules []domain.Rule
	seen := make(map[string]bool)

	dirs := s.getSearchDirs("rules")
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			rule, err := parseRuleMarkdown(string(data), entry.Name())
			if err != nil {
				continue
			}

			if !seen[rule.Header.Name] {
				rules = append(rules, rule)
				seen[rule.Header.Name] = true
			}
		}
	}

	return rules, nil
}

// parseRuleMarkdown разбирает Markdown с YAML frontmatter
func parseRuleMarkdown(content string, fileName string) (domain.Rule, error) {
	var rule domain.Rule
	normalized := strings.ReplaceAll(content, "\r\n", "\n")

	if !strings.HasPrefix(normalized, "---\n") {
		// Нет фронтматтера, используем имя файла как имя правила
		rule.Header = domain.RuleFrontmatter{
			Name:    strings.TrimSuffix(fileName, ".md"),
			Targets: []domain.AgentType{domain.AgentClaudeCode, domain.AgentClaudeDesktop, domain.AgentAntigravity},
		}
		rule.Content = content
		return rule, nil
	}

	parts := strings.SplitN(normalized, "---\n", 3)
	if len(parts) < 3 {
		return rule, fmt.Errorf("неверный формат frontmatter в файле %s", fileName)
	}

	var header domain.RuleFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &header); err != nil {
		return rule, fmt.Errorf("ошибка парсинга frontmatter в %s: %w", fileName, err)
	}

	if header.Name == "" {
		header.Name = strings.TrimSuffix(fileName, ".md")
	}

	rule.Header = header
	rule.Content = strings.TrimSpace(parts[2])
	return rule, nil
}

// LoadSkills загружает все канонические навыки из skills/*/SKILL.md или skills/*.md
func (s *Store) LoadSkills() ([]domain.Rule, error) {
	var skills []domain.Rule
	seen := make(map[string]bool)

	dirs := s.getSearchDirs("skills")
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				if strings.HasSuffix(entry.Name(), ".md") {
					path := filepath.Join(dir, entry.Name())
					data, err := os.ReadFile(path)
					if err == nil {
						skill, err := parseRuleMarkdown(string(data), entry.Name())
						if err == nil && !seen[skill.Header.Name] {
							skills = append(skills, skill)
							seen[skill.Header.Name] = true
						}
					}
				}
				continue
			}

			skillName := entry.Name()
			subDir := filepath.Join(dir, skillName)
			subEntries, err := os.ReadDir(subDir)
			if err != nil {
				continue
			}

			for _, subEntry := range subEntries {
				if !subEntry.IsDir() && (subEntry.Name() == "SKILL.md" || strings.HasSuffix(subEntry.Name(), ".md")) {
					path := filepath.Join(subDir, subEntry.Name())
					data, err := os.ReadFile(path)
					if err != nil {
						continue
					}
					skill, err := parseRuleMarkdown(string(data), skillName)
					if err == nil && !seen[skill.Header.Name] {
						skills = append(skills, skill)
						seen[skill.Header.Name] = true
					}
				}
			}
		}
	}

	return skills, nil
}

// LoadWorkflows загружает все канонические рабочие процессы из workflows/*
func (s *Store) LoadWorkflows() ([]domain.Rule, error) {
	var workflows []domain.Rule
	seen := make(map[string]bool)

	dirs := s.getSearchDirs("workflows")
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var rule domain.Rule
			isYaml := strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml")
			
			if isYaml {
				var fm domain.RuleFrontmatter
				if err := yaml.Unmarshal(data, &fm); err == nil {
					if fm.Name == "" {
						fm.Name = strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".yaml"), ".yml")
					}
					rule.Header = fm
					rule.Content = string(data)
					if !seen[rule.Header.Name] {
						workflows = append(workflows, rule)
						seen[rule.Header.Name] = true
					}
					continue
				}
			}

			wf, err := parseRuleMarkdown(string(data), entry.Name())
			if err == nil && !seen[wf.Header.Name] {
				workflows = append(workflows, wf)
				seen[wf.Header.Name] = true
			}
		}
	}

	return workflows, nil
}

// LoadHooks загружает все хуки из hooks/*
func (s *Store) LoadHooks() ([]domain.Rule, error) {
	var hooks []domain.Rule
	seen := make(map[string]bool)

	dirs := s.getSearchDirs("hooks")
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if !seen[entry.Name()] {
				rule := domain.Rule{
					Header: domain.RuleFrontmatter{
						Name:        entry.Name(),
						Description: "Локальный триггер автоматизации (Hook)",
						Targets:     []domain.AgentType{domain.AgentClaudeCode, domain.AgentAntigravity},
					},
					Content: "Локальный хук репозитория",
				}
				hooks = append(hooks, rule)
				seen[entry.Name()] = true
			}
		}
	}

	return hooks, nil
}

// ResolveComponentPath находит физический путь к файлу компонента (локальный или глобальный)
func (s *Store) ResolveComponentPath(subDir string, fileName string) string {
	local := filepath.Join(s.RepoPath, "data", subDir, fileName)
	if fileExists(local) {
		return local
	}
	home, err := os.UserHomeDir()
	if err == nil {
		global := filepath.Join(home, ".agents", subDir, fileName)
		if fileExists(global) {
			return global
		}
	}
	return local // По умолчанию локальный путь
}

// UpdateMCPTargets обновляет цели для MCP-сервера
func (s *Store) UpdateMCPTargets(name string, targets []domain.AgentType) error {
	fileName := name + ".yaml"
	path := s.ResolveComponentPath("mcp", fileName)
	if !fileExists(path) {
		if fileExists(s.ResolveComponentPath("mcp", name+".yml")) {
			path = s.ResolveComponentPath("mcp", name+".yml")
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var mcp domain.MCPConfig
	if err := yaml.Unmarshal(data, &mcp); err != nil {
		return err
	}

	mcp.Targets = targets

	newData, err := yaml.Marshal(&mcp)
	if err != nil {
		return err
	}

	return os.WriteFile(path, newData, 0644)
}

// UpdateRuleTargets обновляет цели для правила
func (s *Store) UpdateRuleTargets(name string, targets []domain.AgentType) error {
	path := s.ResolveComponentPath("rules", name+".md")

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	normalized := strings.ReplaceAll(content, "\r\n", "\n")

	if !strings.HasPrefix(normalized, "---\n") {
		header := domain.RuleFrontmatter{
			Name:    name,
			Targets: targets,
		}
		headerData, err := yaml.Marshal(&header)
		if err != nil {
			return err
		}
		newContent := fmt.Sprintf("---\n%s---\n\n%s", string(headerData), content)
		return os.WriteFile(path, []byte(newContent), 0644)
	}

	parts := strings.SplitN(normalized, "---\n", 3)
	if len(parts) < 3 {
		return fmt.Errorf("неверный формат frontmatter")
	}

	var header domain.RuleFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &header); err != nil {
		return err
	}

	header.Targets = targets

	headerData, err := yaml.Marshal(&header)
	if err != nil {
		return err
	}

	newContent := fmt.Sprintf("---\n%s---\n%s", string(headerData), parts[2])
	return os.WriteFile(path, []byte(newContent), 0644)
}

// UpdateSkillTargets обновляет цели для навыка (Skill)
func (s *Store) UpdateSkillTargets(name string, targets []domain.AgentType) error {
	path := s.ResolveComponentPath("skills", filepath.Join(name, "SKILL.md"))
	if !fileExists(path) {
		path = s.ResolveComponentPath("skills", name+".md")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	normalized := strings.ReplaceAll(content, "\r\n", "\n")

	if !strings.HasPrefix(normalized, "---\n") {
		header := domain.RuleFrontmatter{
			Name:    name,
			Targets: targets,
		}
		headerData, err := yaml.Marshal(&header)
		if err != nil {
			return err
		}
		newContent := fmt.Sprintf("---\n%s---\n\n%s", string(headerData), content)
		return os.WriteFile(path, []byte(newContent), 0644)
	}

	parts := strings.SplitN(normalized, "---\n", 3)
	if len(parts) < 3 {
		return fmt.Errorf("неверный формат frontmatter")
	}

	var header domain.RuleFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &header); err != nil {
		return err
	}

	header.Targets = targets

	headerData, err := yaml.Marshal(&header)
	if err != nil {
		return err
	}

	newContent := fmt.Sprintf("---\n%s---\n%s", string(headerData), parts[2])
	return os.WriteFile(path, []byte(newContent), 0644)
}

// UpdateWorkflowTargets обновляет цели для воркфлоу
func (s *Store) UpdateWorkflowTargets(name string, targets []domain.AgentType) error {
	path := s.ResolveComponentPath("workflows", name)
	if !fileExists(path) {
		if fileExists(s.ResolveComponentPath("workflows", name+".yaml")) {
			path = s.ResolveComponentPath("workflows", name+".yaml")
		} else if fileExists(s.ResolveComponentPath("workflows", name+".yml")) {
			path = s.ResolveComponentPath("workflows", name+".yml")
		} else if fileExists(s.ResolveComponentPath("workflows", name+".md")) {
			path = s.ResolveComponentPath("workflows", name+".md")
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	isYaml := strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")
	if isYaml {
		var header domain.RuleFrontmatter
		if err := yaml.Unmarshal(data, &header); err != nil {
			return err
		}
		header.Targets = targets
		newData, err := yaml.Marshal(&header)
		if err != nil {
			return err
		}
		return os.WriteFile(path, newData, 0644)
	}

	content := string(data)
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		header := domain.RuleFrontmatter{
			Name:    name,
			Targets: targets,
		}
		headerData, err := yaml.Marshal(&header)
		if err != nil {
			return err
		}
		newContent := fmt.Sprintf("---\n%s---\n\n%s", string(headerData), content)
		return os.WriteFile(path, []byte(newContent), 0644)
	}

	parts := strings.SplitN(normalized, "---\n", 3)
	if len(parts) < 3 {
		return fmt.Errorf("неверный формат frontmatter")
	}

	var header domain.RuleFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &header); err != nil {
		return err
	}

	header.Targets = targets

	headerData, err := yaml.Marshal(&header)
	if err != nil {
		return err
	}

	newContent := fmt.Sprintf("---\n%s---\n%s", string(headerData), parts[2])
	return os.WriteFile(path, []byte(newContent), 0644)
}

// LoadBundles загружает все бандлы из папки configs/*.yaml
func (s *Store) LoadBundles() ([]domain.ConfigBundle, error) {
	configsDir := filepath.Join(s.RepoPath, "data", "configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(configsDir)
	if err != nil {
		return nil, err
	}

	var bundles []domain.ConfigBundle
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}

		path := filepath.Join(configsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var bundle domain.ConfigBundle
		if err := yaml.Unmarshal(data, &bundle); err != nil {
			continue
		}

		if bundle.ID == "" {
			bundle.ID = strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".yaml"), ".yml")
		}
		bundles = append(bundles, bundle)
	}

	return bundles, nil
}

// SaveBundle сохраняет бандл в configs/<id>.yaml
func (s *Store) SaveBundle(bundle domain.ConfigBundle) error {
	configsDir := filepath.Join(s.RepoPath, "data", "configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		return err
	}

	if bundle.ID == "" {
		bundle.ID = strings.ToLower(strings.ReplaceAll(bundle.Name, " ", "-"))
	}

	path := filepath.Join(configsDir, bundle.ID+".yaml")
	data, err := yaml.Marshal(&bundle)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// DeleteBundle удаляет бандл
func (s *Store) DeleteBundle(id string) error {
	path := filepath.Join(s.RepoPath, "data", "configs", id+".yaml")
	if !fileExists(path) {
		path = filepath.Join(s.RepoPath, "data", "configs", id+".yml")
	}
	return os.Remove(path)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
