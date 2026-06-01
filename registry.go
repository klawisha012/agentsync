package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// RegistryItem представляет компонент в удаленном каталоге
type RegistryItem struct {
	Name        string        `json:"name"`
	Type        ComponentType `json:"type"`
	Description string        `json:"description"`
	SourceURL   string        `json:"sourceUrl"`
	Installed   bool          `json:"installed"`
	Targets     []AgentType   `json:"targets"`
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
func FetchRegistryPage(t ComponentType, searchQuery string, page int, perPage int) ([]RegistryItem, error) {
	var query string
	switch t {
	case ComponentMCP:
		query = "topic:mcp-server"
	case ComponentSkill:
		query = "topic:agent-skill"
	case ComponentWorkflow:
		query = "topic:agent-workflow"
	case ComponentRule:
		query = "topic:agent-rule"
	case ComponentHook:
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
		if t == ComponentSkill {
			sourceURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/SKILL.md", ghItem.FullName, ghItem.DefaultBranch)
		} else {
			sourceURL = ghItem.HTMLURL
		}

		items = append(items, RegistryItem{
			Name:        ghItem.Name,
			Type:        t,
			Description: desc,
			SourceURL:   sourceURL,
			Targets:     []AgentType{AgentAntigravity}, // Заглушка
		})
	}

	return items, nil
}

// FetchRegistry возвращает список компонентов из GitHub (для глубокого поиска)
func FetchRegistry(t ComponentType, searchQuery string) ([]RegistryItem, error) {
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
	for _, c := range []ComponentType{ComponentMCP, ComponentSkill} {
		for page := 1; page <= 5; page++ {
			items, err := FetchRegistryPage(c, "", page, 100)
			if err != nil {
				break
			}
			if len(items) == 0 {
				break
			}
			allItems = append(allItems, items...)
			// Ждем, чтобы не поймать Rate Limit (10 запросов в минуту для неавторизованных)
			// Мы делаем суммарно 10 запросов (5 страниц * 2 типа), поэтому 3 секунды задержки нормально.
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
	case ComponentMCP:
		subDir = "mcp"
	case ComponentRule:
		subDir = "rules"
	case ComponentSkill:
		subDir = "skills"
	case ComponentWorkflow:
		subDir = "workflows"
	case ComponentHook:
		subDir = "hooks"
	default:
		return fmt.Errorf("неизвестный тип компонента: %s", item.Type)
	}

	targetDir := filepath.Join(cwd, subDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию %s: %v", targetDir, err)
	}

	if item.Type == ComponentSkill {
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
