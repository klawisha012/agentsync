package repository

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RequiredSecret представляет секрет, необходимый для работы конкретных MCP серверов
type RequiredSecret struct {
	Name   string   `json:"name"`
	UsedBy []string `json:"used_by"`
}

// Паттерны для поиска утечек секретов в канонических файлах (Фаза 2: gitleaks-аудит)
var secretLeakPatterns = map[string]*regexp.Regexp{
	"GitHub Personal Access Token": regexp.MustCompile(`(?i)ghp_[a-zA-Z0-9]{36,255}`),
	"OpenAI API Key":               regexp.MustCompile(`(?i)sk-(?:proj-)?[a-zA-Z0-9]{48,255}`),
	"Generic Private Key":          regexp.MustCompile(`(?i)-----BEGIN[ A-Z0-9_-]+PRIVATE KEY-----`),
	"Slack Webhook URL":            regexp.MustCompile(`https://hooks\.slack\.com/services/[T|B][A-Z0-9_]+/[A-Z0-9_]+/[A-Z0-9_]+`),
}

// LoadSecrets загружает секреты из ~/.agentsync/secrets.env или локального secrets.env
func LoadSecrets(repoPath string) (map[string]string, error) {
	secrets := make(map[string]string)

	home, err := os.UserHomeDir()
	if err != nil {
		return secrets, err
	}

	// Сначала проверяем глобальный путь ~/.agentsync/secrets.env
	secretsPath := filepath.Join(home, ".agentsync", "secrets.env")
	if !repoFileExists(secretsPath) {
		// Затем проверяем локальный secrets.env в репозитории
		secretsPath = filepath.Join(repoPath, "data", "secrets.env")
		if !repoFileExists(secretsPath) {
			// Если нигде нет, возвращаем пустую мапу без ошибки
			return secrets, nil
		}
	}

	file, err := os.Open(secretsPath)
	if err != nil {
		return secrets, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Убираем окружающие кавычки, если есть
		val = strings.Trim(val, "\"'")

		secrets[key] = val
	}

	return secrets, scanner.Err()
}

// AuditRepository сканирует файлы репозитория на утечки реальных секретов
func AuditRepository(repoPath string) ([]string, error) {
	var warnings []string

	// Проходим по папкам mcp/ и rules/
	dirs := []string{
		filepath.Join(repoPath, "mcp"),
		filepath.Join(repoPath, "rules"),
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("ошибка чтения %s при аудите: %w", entry.Name(), err)
			}

			content := string(data)
			relPath, _ := filepath.Rel(repoPath, path)

			// Сканируем контент по паттернам уязвимостей
			for name, re := range secretLeakPatterns {
				matches := re.FindAllString(content, -1)
				for _, match := range matches {
					// Проверяем, не является ли это плейсхолдером (например, GITHUB_TOKEN)
					if strings.Contains(match, "secrets.") {
						continue
					}
					// Вырезаем середину секрета для безопасного вывода
					masked := match
					if len(match) > 12 {
						masked = match[:6] + "..." + match[len(match)-6:]
					}
					// Возвращаем чистые строки варнингов без ANSI цветов
					warnings = append(warnings, fmt.Sprintf("%s: Найден сырой секрет %s в файле %s", name, masked, relPath))
				}
			}
		}
	}

	return warnings, nil
}

// SaveSecrets записывает секреты в secrets.env
func SaveSecrets(repoPath string, secrets map[string]string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	secretsPath := filepath.Join(home, ".agentsync", "secrets.env")
	if !repoFileExists(secretsPath) {
		secretsPath = filepath.Join(repoPath, "data", "secrets.env")
	}

	// Убедимся, что директория существует
	if err := os.MkdirAll(filepath.Dir(secretsPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(secretsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	for k, v := range secrets {
		if k == "" {
			continue
		}
		_, err = fmt.Fprintf(file, "%s=%s\n", k, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func repoFileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
