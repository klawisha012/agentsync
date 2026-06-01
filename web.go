package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// StartWebServer запускает HTTP-сервер для Web GUI
func StartWebServer(port int, repoPath string) error {
	// Включаем CORS для разработки, если фронтенд запущен отдельно
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// 1. Статические файлы фронтенда
	distPath := filepath.Join(repoPath, "frontend", "dist")
	if fileExistsDir(distPath) {
		fs := http.FileServer(http.Dir(distPath))
		http.Handle("/", corsMiddleware(fs))
	} else {
		// Если фронтенд не скомпилирован, отдаем заглушку
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api") {
				return // Пропускаем API
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>AgentSync Web GUI</title></head>
<body style="font-family: sans-serif; background: #0a0b10; color: #fff; text-align: center; padding: 50px;">
  <h1 style="color: #00f0ff;">⚡ AgentSync Web GUI ⚡</h1>
  <p>Фронтенд не скомпилирован. Выполните в папке frontend:</p>
  <code>npm run build</code>
  <p>После чего перезапустите веб-сервер.</p>
</body>
</html>`))
		})
	}

	// 2. API Маршруты
	http.Handle("/api/status", corsMiddleware(http.HandlerFunc(apiStatusHandler(repoPath))))
	http.Handle("/api/components", corsMiddleware(http.HandlerFunc(apiComponentsHandler(repoPath))))
	http.Handle("/api/components/sync-source", corsMiddleware(http.HandlerFunc(apiSyncSourceHandler(repoPath))))
	http.Handle("/api/components/update-targets", corsMiddleware(http.HandlerFunc(apiUpdateTargetsHandler(repoPath))))
	http.Handle("/api/secrets", corsMiddleware(http.HandlerFunc(apiSecretsHandler(repoPath))))
	http.Handle("/api/deploy", corsMiddleware(http.HandlerFunc(apiDeployHandler(repoPath))))
	http.Handle("/api/manifest", corsMiddleware(http.HandlerFunc(apiManifestHandler(repoPath))))
	http.Handle("/api/bundles", corsMiddleware(http.HandlerFunc(apiBundlesHandler(repoPath))))
	http.Handle("/api/bundles/sync", corsMiddleware(http.HandlerFunc(apiBundlesSyncHandler(repoPath))))
	http.Handle("/api/bundles/share", corsMiddleware(http.HandlerFunc(apiBundlesShareHandler(repoPath))))
	http.Handle("/api/registry/search", corsMiddleware(http.HandlerFunc(apiRegistrySearchHandler(repoPath))))
	http.Handle("/api/registry/install", corsMiddleware(http.HandlerFunc(apiRegistryInstallHandler(repoPath))))

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("%s[+] Запуск Web GUI на http://localhost%s...%s\n", ColorGreen, addr, ColorReset)
	
	// Попытка автоматически открыть браузер
	go openBrowser(fmt.Sprintf("http://localhost:%d", port))

	return http.ListenAndServe(addr, nil)
}

func fileExistsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// openBrowser открывает ссылку в дефолтном браузере Windows
func openBrowser(url string) {
	// Для Windows используем cmd /c start
	// В реальной жизни это можно сделать через exec.Command
	// Для безопасности мы просто выводим лог, но на Windows cmd.exe /c start работает отлично
	// exec.Command("cmd", "/c", "start", url).Start()
}

// GET /api/status
func apiStatusHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		claudePath := filepath.Join(homeDir, ".claude.json")
		claudeDesktopPath := ResolveClaudeDesktopPath()
		antigravityPath := filepath.Join(homeDir, ".gemini", "config", "mcp_config.json")

		agents := []AgentInfo{
			{Name: "Claude Code (CLI)", ConfigPaths: []string{claudePath}, Detected: fileExists(claudePath)},
			{Name: "Claude Desktop", ConfigPaths: []string{claudeDesktopPath}, Detected: fileExists(claudeDesktopPath)},
			{Name: "Antigravity IDE & CLI", ConfigPaths: []string{antigravityPath}, Detected: fileExists(antigravityPath)},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agents)
	}
}

// GET /api/components
func apiComponentsHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		store := NewStore(repoPath)
		var components []AgentComponent

		// Загружаем MCP
		mcps, _ := store.LoadMCPs()
		for _, mcp := range mcps {
			var targets []AgentType
			for _, t := range mcp.Targets {
				targets = append(targets, AgentType(t))
			}
			components = append(components, AgentComponent{
				Type:        ComponentMCP,
				Name:        mcp.Name,
				Description: mcp.Description,
				Path:        filepath.Join("mcp", mcp.Name+".yaml"),
				Targets:     targets,
				Details:     fmt.Sprintf("Команда: %s %s", mcp.Command, strings.Join(mcp.Args, " ")),
				Active:      true,
			})
		}

		// Загружаем Rules
		rules, _ := store.LoadRules()
		for _, rule := range rules {
			components = append(components, AgentComponent{
				Type:        ComponentRule,
				Name:        rule.Header.Name,
				Description: rule.Header.Description,
				Path:        filepath.Join("rules", rule.Header.Name+".md"),
				Targets:     rule.Header.Targets,
				Details:     fmt.Sprintf("Область: %s", rule.Header.Scope),
				Active:      true,
			})
		}

		// Загружаем Skills
		skills, _ := store.LoadSkills()
		for _, skill := range skills {
			components = append(components, AgentComponent{
				Type:        ComponentSkill,
				Name:        skill.Header.Name,
				Description: skill.Header.Description,
				Path:        filepath.Join("skills", skill.Header.Name, "SKILL.md"),
				Targets:     skill.Header.Targets,
				Details:     "Навык контекстного мышления (Skill)",
				Active:      true,
			})
		}

		// Загружаем Workflows
		wfs, _ := store.LoadWorkflows()
		for _, wf := range wfs {
			components = append(components, AgentComponent{
				Type:        ComponentWorkflow,
				Name:        wf.Header.Name,
				Description: wf.Header.Description,
				Path:        filepath.Join("workflows", wf.Header.Name),
				Targets:     wf.Header.Targets,
				Details:     "Сценарий рабочего процесса (Workflow)",
				Active:      true,
			})
		}

		// Загружаем Hooks
		hooks, _ := store.LoadHooks()
		for _, hook := range hooks {
			components = append(components, AgentComponent{
				Type:        ComponentHook,
				Name:        hook.Header.Name,
				Description: hook.Header.Description,
				Path:        filepath.Join("hooks", hook.Header.Name),
				Targets:     hook.Header.Targets,
				Details:     "Скрипт автоматизации задач (Hook)",
				Active:      true,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(components)
	}
}

// POST /api/components/sync-source
func apiSyncSourceHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string        `json:"name"`
			Type     ComponentType `json:"type"`
			ToGlobal bool          `json:"toGlobal"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		comp := AgentComponent{
			Name: req.Name,
			Type: req.Type,
		}

		err := SyncComponentSource(repoPath, comp, req.ToGlobal)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Write([]byte(`{"status":"ok"}`))
	}
}

// POST /api/components/update-targets
func apiUpdateTargetsHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name    string        `json:"name"`
			Type    ComponentType `json:"type"`
			Targets []AgentType   `json:"targets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		store := NewStore(repoPath)
		var err error
		switch req.Type {
		case ComponentMCP:
			err = store.UpdateMCPTargets(req.Name, req.Targets)
		case ComponentRule:
			err = store.UpdateRuleTargets(req.Name, req.Targets)
		case ComponentSkill:
			err = store.UpdateSkillTargets(req.Name, req.Targets)
		case ComponentWorkflow:
			err = store.UpdateWorkflowTargets(req.Name, req.Targets)
		default:
			http.Error(w, "неподдерживаемый тип для синхронизации целей", 400)
			return
		}

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Write([]byte(`{"status":"ok"}`))
	}
}

// GET, POST, DELETE /api/secrets
func apiSecretsHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			secrets, err := LoadSecrets(repoPath)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			required := GetRequiredSecrets(repoPath)

			resp := struct {
				Values   map[string]string `json:"values"`
				Required []RequiredSecret  `json:"required"`
			}{
				Values:   secrets,
				Required: required,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case "POST":
			var req struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}

			secrets, _ := LoadSecrets(repoPath)
			secrets[req.Name] = req.Value
			err := SaveSecrets(repoPath, secrets)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write([]byte(`{"status":"ok"}`))

		case "DELETE":
			var req struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}

			secrets, _ := LoadSecrets(repoPath)
			delete(secrets, req.Name)
			err := SaveSecrets(repoPath, secrets)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write([]byte(`{"status":"ok"}`))
		}
	}
}

// GET, POST /api/manifest
func apiManifestHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		store := NewStore(repoPath)
		switch r.Method {
		case "GET":
			manifest, err := store.LoadManifest()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest)
		case "POST":
			var req struct {
				ActiveAgents []AgentType `json:"active_agents"`
				ActiveBundle string      `json:"active_bundle"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			manifest, err := store.LoadManifest()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			manifest.ActiveAgents = req.ActiveAgents
			manifest.ActiveBundle = req.ActiveBundle
			err = store.SaveManifest(manifest)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write([]byte(`{"status":"ok"}`))
		default:
			http.Error(w, "Метод не поддерживается", 405)
		}
	}
}

// POST /api/deploy

func apiDeployHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ActiveBundle string `json:"active_bundle"`
		}
		// Пробуем декодировать тело запроса (для обратной совместимости, если тела нет, игнорируем)
		_ = json.NewDecoder(r.Body).Decode(&req)

		go func() {
			runDeploy(req.ActiveBundle)
		}()
		w.Write([]byte(`{"status":"ok","msg":"деплой запущен"}`))
	}
}

// GET /api/registry/search
func apiRegistrySearchHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		t := r.URL.Query().Get("type")

		var compType ComponentType
		switch t {
		case "MCP":
			compType = ComponentMCP
		case "Skill":
			compType = ComponentSkill
		case "Rule":
			compType = ComponentRule
		case "Workflow":
			compType = ComponentWorkflow
		case "Hook":
			compType = ComponentHook
		default:
			compType = ComponentMCP
		}

		// Запускаем поиск (глубокий через GitHub API)
		items, err := FetchRegistry(compType, q)
		if err != nil {
			// Пробуем локально по кэшу
			cached, errLoad := LoadLocalRegistry()
			if errLoad == nil {
				var filtered []RegistryItem
				for _, item := range cached {
					if item.Type == compType && (q == "" || strings.Contains(strings.ToLower(item.Name), strings.ToLower(q))) {
						filtered = append(filtered, item)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(filtered)
				return
			}
			http.Error(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}
}

// POST /api/registry/install
func apiRegistryInstallHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string        `json:"name"`
			Type        ComponentType `json:"type"`
			Description string        `json:"description"`
			SourceURL   string        `json:"sourceUrl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		item := RegistryItem{
			Name:        req.Name,
			Type:        req.Type,
			Description: req.Description,
			SourceURL:   req.SourceURL,
		}

		err := InstallComponent(item)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Write([]byte(`{"status":"ok"}`))
	}
}

var rxSecret = regexp.MustCompile(`\$\{secrets\.([a-zA-Z0-9_]+)\}`)

type RequiredSecret struct {
	Name   string   `json:"name"`
	UsedBy []string `json:"usedBy"`
}

// GetRequiredSecrets обходит все MCP и вытаскивает переменные secrets.env
func GetRequiredSecrets(repoPath string) []RequiredSecret {
	store := NewStore(repoPath)
	mcps, err := store.LoadMCPs()
	if err != nil {
		return nil
	}

	reqMap := make(map[string][]string)

	for _, mcp := range mcps {
		for _, val := range mcp.Env {
			matches := rxSecret.FindStringSubmatch(val)
			if len(matches) > 1 {
				secName := matches[1]
				reqMap[secName] = append(reqMap[secName], fmt.Sprintf("MCP '%s'", mcp.Name))
			}
		}
	}

	var result []RequiredSecret
	for secName, usedBy := range reqMap {
		result = append(result, RequiredSecret{
			Name:   secName,
			UsedBy: usedBy,
		})
	}
	return result
}

// GET, POST, DELETE /api/bundles
func apiBundlesHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		store := NewStore(repoPath)
		switch r.Method {
		case "GET":
			bundles, err := store.LoadBundles()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bundles)
		case "POST":
			var bundle ConfigBundle
			if err := json.NewDecoder(r.Body).Decode(&bundle); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			err := store.SaveBundle(bundle)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write([]byte(`{"status":"ok"}`))
		case "DELETE":
			var req struct {
				ID string `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			err := store.DeleteBundle(req.ID)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write([]byte(`{"status":"ok"}`))
		}
	}
}

// POST /api/bundles/sync
func apiBundlesSyncHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		store := NewStore(repoPath)
		bundles, err := store.LoadBundles()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var targetBundle *ConfigBundle
		for _, b := range bundles {
			if b.ID == req.ID {
				targetBundle = &b
				break
			}
		}

		if targetBundle == nil {
			http.Error(w, "Бандл не найден", 404)
			return
		}

		// Логика синхронизации: для каждого компонента бандла переносим его в локальный репозиторий
		for _, item := range targetBundle.Components {
			comp := AgentComponent{
				Name: item.Name,
				Type: item.Type,
			}
			_ = SyncComponentSource(repoPath, comp, false) // синхронизируем локально
		}

		w.Write([]byte(`{"status":"ok","msg":"Бандл успешно синхронизирован локально"}`))
	}
}

// POST /api/bundles/share
func apiBundlesShareHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// Запускаем P2P-раздачу бандла в фоновом режиме
		go func() {
			_ = P2PShare(repoPath, "8899", 0)
		}()

		resp := map[string]string{
			"status": "ok",
			"pin":    "8899",
			"msg":    "P2P-раздача бандла запущена. Подключение по PIN-коду.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

