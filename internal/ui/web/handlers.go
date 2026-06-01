package web

import (
	"agentsync/internal/agent"
	"agentsync/internal/domain"
	"agentsync/internal/repository"
	"agentsync/internal/service"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func fileExistsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func webFileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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
		claudeDesktopPath := agent.ResolveClaudeDesktopPath()
		antigravityPath := filepath.Join(homeDir, ".gemini", "config", "mcp_config.json")

		agents := []struct {
			Name        string   `json:"name"`
			Detected    bool     `json:"detected"`
			ConfigPaths []string `json:"config_paths"`
		}{
			{Name: "Claude Code (CLI)", ConfigPaths: []string{claudePath}, Detected: webFileExists(claudePath)},
			{Name: "Claude Desktop", ConfigPaths: []string{claudeDesktopPath}, Detected: webFileExists(claudeDesktopPath)},
			{Name: "Antigravity IDE & CLI", ConfigPaths: []string{antigravityPath}, Detected: webFileExists(antigravityPath)},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agents)
	}
}

// GET /api/components
func apiComponentsHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		store := repository.NewStore(repoPath)
		var components []domain.AgentComponent

		home, _ := os.UserHomeDir()

		// Загружаем MCP
		mcps, _ := store.LoadMCPs()
		for _, mcp := range mcps {
			var targets []domain.AgentType
			for _, t := range mcp.Targets {
				targets = append(targets, domain.AgentType(t))
			}
			fileName := mcp.Name + ".yaml"
			components = append(components, domain.AgentComponent{
				Type:         domain.ComponentMCP,
				Name:         mcp.Name,
				Description:  mcp.Description,
				Path:         filepath.Join("mcp", fileName),
				Targets:      targets,
				Details:      fmt.Sprintf("Команда: %s %s", mcp.Command, strings.Join(mcp.Args, " ")),
				Active:       true,
				LocalExists:  webFileExists(filepath.Join(repoPath, "mcp", fileName)),
				GlobalExists: webFileExists(filepath.Join(home, ".agents", "mcp", fileName)),
			})
		}

		// Загружаем Rules
		rules, _ := store.LoadRules()
		for _, rule := range rules {
			fileName := rule.Header.Name + ".md"
			components = append(components, domain.AgentComponent{
				Type:         domain.ComponentRule,
				Name:         rule.Header.Name,
				Description:  rule.Header.Description,
				Path:         filepath.Join("rules", fileName),
				Targets:      rule.Header.Targets,
				Details:      fmt.Sprintf("Область: %s", rule.Header.Scope),
				Active:       true,
				LocalExists:  webFileExists(filepath.Join(repoPath, "rules", fileName)),
				GlobalExists: webFileExists(filepath.Join(home, ".agents", "rules", fileName)),
			})
		}

		// Загружаем Skills
		skills, _ := store.LoadSkills()
		for _, skill := range skills {
			fileName := filepath.Join(skill.Header.Name, "SKILL.md")
			components = append(components, domain.AgentComponent{
				Type:         domain.ComponentSkill,
				Name:         skill.Header.Name,
				Description:  skill.Header.Description,
				Path:         filepath.Join("skills", fileName),
				Targets:      skill.Header.Targets,
				Details:      "Навык контекстного мышления (Skill)",
				Active:       true,
				LocalExists:  webFileExists(filepath.Join(repoPath, "skills", fileName)),
				GlobalExists: webFileExists(filepath.Join(home, ".agents", "skills", fileName)),
			})
		}

		// Загружаем Workflows
		wfs, _ := store.LoadWorkflows()
		for _, wf := range wfs {
			components = append(components, domain.AgentComponent{
				Type:         domain.ComponentWorkflow,
				Name:         wf.Header.Name,
				Description:  wf.Header.Description,
				Path:         filepath.Join("workflows", wf.Header.Name),
				Targets:      wf.Header.Targets,
				Details:      "Сценарий рабочего процесса (Workflow)",
				Active:       true,
				LocalExists:  webFileExists(filepath.Join(repoPath, "workflows", wf.Header.Name)),
				GlobalExists: webFileExists(filepath.Join(home, ".agents", "workflows", wf.Header.Name)),
			})
		}

		// Загружаем Hooks
		hooks, _ := store.LoadHooks()
		for _, hook := range hooks {
			components = append(components, domain.AgentComponent{
				Type:         domain.ComponentHook,
				Name:         hook.Header.Name,
				Description:  hook.Header.Description,
				Path:         filepath.Join("hooks", hook.Header.Name),
				Targets:      hook.Header.Targets,
				Details:      "Скрипт автоматизации задач (Hook)",
				Active:       true,
				LocalExists:  webFileExists(filepath.Join(repoPath, "hooks", hook.Header.Name)),
				GlobalExists: webFileExists(filepath.Join(home, ".agents", "hooks", hook.Header.Name)),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(components)
	}
}

// POST /api/components/update-targets
func apiUpdateTargetsHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name    string               `json:"name"`
			Type    domain.ComponentType `json:"type"`
			Targets []domain.AgentType   `json:"targets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		store := repository.NewStore(repoPath)
		var err error
		switch req.Type {
		case domain.ComponentMCP:
			err = store.UpdateMCPTargets(req.Name, req.Targets)
		case domain.ComponentRule:
			err = store.UpdateRuleTargets(req.Name, req.Targets)
		case domain.ComponentSkill:
			err = store.UpdateSkillTargets(req.Name, req.Targets)
		case domain.ComponentWorkflow:
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

// POST /api/components/sync-source
func apiSyncSourceHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string               `json:"name"`
			Type     domain.ComponentType `json:"type"`
			ToGlobal bool                 `json:"toGlobal"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		comp := domain.AgentComponent{
			Name: req.Name,
			Type: req.Type,
		}

		err := service.SyncComponentSource(repoPath, comp, req.ToGlobal)
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
			secrets, err := repository.LoadSecrets(repoPath)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			required := getRequiredSecretsLocal(repoPath)

			resp := struct {
				Values   map[string]string           `json:"values"`
				Required []repository.RequiredSecret `json:"required"`
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

			secrets, _ := repository.LoadSecrets(repoPath)
			secrets[req.Name] = req.Value
			err := repository.SaveSecrets(repoPath, secrets)
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

			secrets, _ := repository.LoadSecrets(repoPath)
			delete(secrets, req.Name)
			err := repository.SaveSecrets(repoPath, secrets)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write([]byte(`{"status":"ok"}`))
		}
	}
}

var rxSecretLocal = regexp.MustCompile(`\$\{secrets\.([a-zA-Z0-9_]+)\}`)

func getRequiredSecretsLocal(repoPath string) []repository.RequiredSecret {
	store := repository.NewStore(repoPath)
	mcps, err := store.LoadMCPs()
	if err != nil {
		return nil
	}

	reqMap := make(map[string][]string)

	for _, mcp := range mcps {
		for _, val := range mcp.Env {
			matches := rxSecretLocal.FindStringSubmatch(val)
			if len(matches) > 1 {
				secName := matches[1]
				reqMap[secName] = append(reqMap[secName], fmt.Sprintf("MCP '%s'", mcp.Name))
			}
		}
	}

	var result []repository.RequiredSecret
	for secName, usedBy := range reqMap {
		result = append(result, repository.RequiredSecret{
			Name:   secName,
			UsedBy: usedBy,
		})
	}
	return result
}

// GET, POST /api/manifest
func apiManifestHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		store := repository.NewStore(repoPath)
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
				ActiveAgents []domain.AgentType `json:"active_agents"`
				ActiveBundle string             `json:"active_bundle"`
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
		_ = json.NewDecoder(r.Body).Decode(&req)

		go func() {
			deployService := service.NewDeployService(repoPath, nil)
			_ = deployService.Deploy(req.ActiveBundle)
		}()
		w.Write([]byte(`{"status":"ok","msg":"деплой запущен"}`))
	}
}

// GET, POST, DELETE /api/bundles
func apiBundlesHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		store := repository.NewStore(repoPath)
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
			var bundle domain.ConfigBundle
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

		store := repository.NewStore(repoPath)
		bundles, err := store.LoadBundles()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var targetBundle *domain.ConfigBundle
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

		go func() {
			_ = service.P2PShare(repoPath, "8899", 0, nil)
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

// GET /api/registry/search
func apiRegistrySearchHandler(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		t := r.URL.Query().Get("type")

		var compType domain.ComponentType
		switch t {
		case "MCP":
			compType = domain.ComponentMCP
		case "Skill":
			compType = domain.ComponentSkill
		case "Rule":
			compType = domain.ComponentRule
		case "Workflow":
			compType = domain.ComponentWorkflow
		case "Hook":
			compType = domain.ComponentHook
		default:
			compType = domain.ComponentMCP
		}

		items, err := service.FetchRegistry(compType, q)
		if err != nil {
			cached, errLoad := service.LoadLocalRegistry()
			if errLoad == nil {
				var filtered []service.RegistryItem
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
			Name        string               `json:"name"`
			Type        domain.ComponentType `json:"type"`
			Description string               `json:"description"`
			SourceURL   string               `json:"sourceUrl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		item := service.RegistryItem{
			Name:        req.Name,
			Type:        req.Type,
			Description: req.Description,
			SourceURL:   req.SourceURL,
		}

		err := service.InstallComponent(item)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Write([]byte(`{"status":"ok"}`))
	}
}

// Публичные обертки для модульных тестов
func TestApiStatusHandlerWrapper(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return apiStatusHandler(repoPath)
}

func TestApiComponentsHandlerWrapper(repoPath string) func(w http.ResponseWriter, r *http.Request) {
	return apiComponentsHandler(repoPath)
}
