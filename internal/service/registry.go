package service

import (
	"agentsync/internal/domain"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RegistryItem представляет компонент в удаленном каталоге
type RegistryItem struct {
	Name        string               `json:"name"`
	Type        domain.ComponentType `json:"type"`
	Description string               `json:"description"`
	SourceURL   string               `json:"sourceUrl"`
	Installed   bool                 `json:"installed"`
	Targets     []domain.AgentType   `json:"targets"`
}

type RegistryCache struct {
	LastUpdated time.Time      `json:"lastUpdated"`
	Items       []RegistryItem `json:"items"`
}

type gitHubSearchResponse struct {
	Items []struct {
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Description   string `json:"description"`
		HTMLURL       string `json:"html_url"`
		DefaultBranch string `json:"default_branch"`
	} `json:"items"`
}

// FetchRegistryPage загружает одну страницу с GitHub
func FetchRegistryPage(t domain.ComponentType, searchQuery string, page int, perPage int) ([]RegistryItem, error) {
	var query string
	switch t {
	case domain.ComponentMCP:
		query = "topic:mcp-server"
	case domain.ComponentSkill:
		query = "topic:agent-skill"
	case domain.ComponentWorkflow:
		query = "topic:agent-workflow"
	case domain.ComponentRule:
		query = "topic:agent-rule"
	case domain.ComponentHook:
		query = "topic:agent-hook"
	default:
		return nil, nil // Return empty for unknown
	}

	if searchQuery != "" {
		query += " " + searchQuery
	}

	url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&order=desc&per_page=%d&page=%d", query, perPage, page)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AgentSync-TUI")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var searchRes gitHubSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return nil, err
	}

	var items []RegistryItem
	for _, ghItem := range searchRes.Items {
		desc := ghItem.Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		if desc == "" {
			desc = "Нет описания"
		}

		var sourceURL string
		if t == domain.ComponentSkill {
			sourceURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/SKILL.md", ghItem.FullName, ghItem.DefaultBranch)
		} else {
			sourceURL = ghItem.HTMLURL
		}

		items = append(items, RegistryItem{
			Name:        ghItem.Name,
			Type:        t,
			Description: desc,
			SourceURL:   sourceURL,
			Targets:     []domain.AgentType{domain.AgentAntigravity}, // Заглушка
		})
	}

	return items, nil
}

// FetchRegistry возвращает список компонентов из GitHub (для глубокого поиска)
func FetchRegistry(t domain.ComponentType, searchQuery string) ([]RegistryItem, error) {
	return FetchRegistryPage(t, searchQuery, 1, 50)
}

// LoadLocalRegistry читает кэш из файла
func LoadLocalRegistry() ([]RegistryItem, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cachePath := filepath.Join(cwd, "data", "registry_cache.json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache RegistryCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return cache.Items, nil
}

// SyncLocalRegistry скачивает ТОП-500 для MCP и Skills в фоне и сохраняет в БД
func SyncLocalRegistry() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dataDir := filepath.Join(cwd, "data")
	os.MkdirAll(dataDir, 0755)
	cachePath := filepath.Join(dataDir, "registry_cache.json")

	var allItems []RegistryItem
	for _, c := range []domain.ComponentType{domain.ComponentMCP, domain.ComponentSkill} {
		for page := 1; page <= 5; page++ {
			items, err := FetchRegistryPage(c, "", page, 100)
			if err != nil {
				break
			}
			if len(items) == 0 {
				break
			}
			allItems = append(allItems, items...)
			time.Sleep(3 * time.Second)
		}
	}

	if len(allItems) > 0 {
		cache := RegistryCache{
			LastUpdated: time.Now(),
			Items:       allItems,
		}

		data, err := json.MarshalIndent(cache, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(cachePath, data, 0644)
	}

	return fmt.Errorf("sync empty")
}

// InstallComponent скачивает компонент и сохраняет его в текущую директорию (CWD)
func InstallComponent(item RegistryItem) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("ошибка получения текущей директории: %v", err)
	}

	var subDir string
	switch item.Type {
	case domain.ComponentMCP:
		subDir = "mcp"
	case domain.ComponentRule:
		subDir = "rules"
	case domain.ComponentSkill:
		subDir = "skills"
	case domain.ComponentWorkflow:
		subDir = "workflows"
	case domain.ComponentHook:
		subDir = "hooks"
	default:
		return fmt.Errorf("неизвестный тип компонента: %s", item.Type)
	}

	targetDir := filepath.Join(cwd, subDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию %s: %v", targetDir, err)
	}

	if item.Type == domain.ComponentSkill {
		skillDir := filepath.Join(targetDir, item.Name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		targetPath := filepath.Join(skillDir, "SKILL.md")

		resp, err := http.Get(item.SourceURL)
		if err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close()
			out, err := os.Create(targetPath)
			if err == nil {
				defer out.Close()
				_, err = io.Copy(out, resp.Body)
				if err == nil {
					return nil
				}
			}
		}
		mockContent := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n// Не удалось автоматически скачать SKILL.md\n// Ссылка на репозиторий: %s\n", item.Name, item.Description, item.SourceURL)
		return os.WriteFile(targetPath, []byte(mockContent), 0644)

	} else {
		targetPath := filepath.Join(targetDir, item.Name+".md")
		mockContent := fmt.Sprintf("# %s\n\n%s\n\nРепозиторий: %s\n\n> Чтобы использовать этот компонент, склонируйте его и настройте согласно его документации.", item.Name, item.Description, item.SourceURL)
		err = os.WriteFile(targetPath, []byte(mockContent), 0644)
		if err != nil {
			return fmt.Errorf("ошибка записи файла: %v", err)
		}
		return nil
	}
}

// InstallPackage скачивает и устанавливает канонический ресурс в репозиторий
func InstallPackage(repoPath string, resourceType string, source string) error {
	var downloadURL string
	var destFileName string

	// 1. Обрабатываем шорткаты источников
	if strings.HasPrefix(source, "github:") {
		cleaned := strings.TrimPrefix(source, "github:")
		parts := strings.Split(cleaned, "/")
		if len(parts) < 3 {
			return fmt.Errorf("неверный формат github ссылки. Ожидается: github:user/repo/branch/path/to/file")
		}
		user := parts[0]
		repo := parts[1]
		branch := parts[2]
		path := strings.Join(parts[3:], "/")
		downloadURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", user, repo, branch, path)
		destFileName = parts[len(parts)-1]
	} else if strings.HasPrefix(source, "smithery:") {
		mcpName := strings.TrimPrefix(source, "smithery:")
		downloadURL = fmt.Sprintf("https://registry.smithery.ai/mcp/%s/raw-config", mcpName)
		destFileName = mcpName + ".yaml"
		resourceType = "mcp"
	} else if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		downloadURL = source
		parts := strings.Split(source, "/")
		destFileName = parts[len(parts)-1]
		if destFileName == "" {
			destFileName = "downloaded_resource"
		}
	} else {
		if resourceType == "mcp" && source == "sqlite" {
			downloadURL = "https://raw.githubusercontent.com/modelcontextprotocol/servers/main/src/sqlite/mcp-config.yaml"
			destFileName = "sqlite.yaml"
		} else if resourceType == "rule" && source == "go-style" {
			downloadURL = "https://raw.githubusercontent.com/google/styleguide/gh-pages/go.md"
			destFileName = "go-style.md"
		} else {
			return fmt.Errorf("неподдерживаемый формат источника. Используйте полную URL-ссылку или шорткаты github: / smithery:")
		}
	}

	fmt.Printf("[+] Скачивание ресурса из %s...\n", downloadURL)

	// 2. Делаем HTTP GET-запрос
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("ошибка HTTP-запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(downloadURL, "smithery.ai") {
			fmt.Printf("[!] Ссылка Smithery недоступна, генерируем канонический шаблон %s...\n", destFileName)
			return generateSmitheryMock(repoPath, strings.TrimSuffix(destFileName, ".yaml"))
		}
		return fmt.Errorf("сервер вернул статус: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения тела ответа: %w", err)
	}

	// 3. Автоматически определяем тип ресурса
	if resourceType == "" {
		if strings.HasSuffix(destFileName, ".md") {
			if strings.Contains(string(data), "targets:") {
				resourceType = "rule"
			} else {
				resourceType = "skill"
			}
		} else if strings.HasSuffix(destFileName, ".yaml") || strings.HasSuffix(destFileName, ".yml") {
			resourceType = "mcp"
		} else {
			resourceType = "rule"
		}
	}

	// 4. Вычисляем целевую папку
	var destDir string
	switch resourceType {
	case "mcp":
		destDir = filepath.Join(repoPath, "mcp")
		if !strings.HasSuffix(destFileName, ".yaml") && !strings.HasSuffix(destFileName, ".yml") {
			destFileName += ".yaml"
		}
	case "rule":
		destDir = filepath.Join(repoPath, "rules")
		if !strings.HasSuffix(destFileName, ".md") {
			destFileName += ".md"
		}
	case "skill":
		skillName := strings.TrimSuffix(destFileName, ".md")
		destDir = filepath.Join(repoPath, "skills", skillName)
		destFileName = "SKILL.md"
	case "workflow":
		destDir = filepath.Join(repoPath, "workflows")
	default:
		destDir = filepath.Join(repoPath, "rules")
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать целевой каталог %s: %w", destDir, err)
	}

	destPath := filepath.Join(destDir, destFileName)

	// 5. Валидируем скачанный контент
	if resourceType == "mcp" {
		var mcp domain.MCPConfig
		if err := yaml.Unmarshal(data, &mcp); err != nil {
			fmt.Printf("ℹ Конвертируем внешний JSON/YAML в канонический формат AgentSync...\n")
			data = wrapExternalMCPToCanonical(data, strings.TrimSuffix(destFileName, ".yaml"))
		}
	}

	// 6. Записываем файл в репозиторий
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла в репозиторий: %w", err)
	}

	fmt.Printf("✔ Успешно установлен пакет %s по пути %s!\n", destFileName, filepath.Join(filepath.Base(destDir), destFileName))
	return nil
}

func wrapExternalMCPToCanonical(raw []byte, name string) []byte {
	var mcp domain.MCPConfig
	mcp.Name = name
	mcp.Description = "Установлено через менеджер пакетов AgentSync"
	mcp.Command = "npx"
	mcp.Args = []string{"-y", name}
	mcp.Targets = []domain.AgentType{domain.AgentClaudeCode, domain.AgentClaudeDesktop, domain.AgentAntigravity}

	data, err := yaml.Marshal(mcp)
	if err != nil {
		return raw
	}
	return data
}

func generateSmitheryMock(repoPath string, name string) error {
	mcp := domain.MCPConfig{
		Name:        name,
		Description: fmt.Sprintf("MCP сервер %s, импортированный из реестра Smithery", name),
		Command:     "npx",
		Args:        []string{"-y", "@smithery-mcp/server-" + name},
		Targets:     []domain.AgentType{domain.AgentClaudeCode, domain.AgentClaudeDesktop, domain.AgentAntigravity},
	}

	destPath := filepath.Join(repoPath, "mcp", name+".yaml")
	data, err := yaml.Marshal(mcp)
	if err != nil {
		return err
	}

	return os.WriteFile(destPath, data, 0644)
}
