package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAllComponents(t *testing.T) {
	// Создаем временную директорию репозитория
	tmpDir, err := os.MkdirTemp("", "agentsync-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Создаем MCP
	mcpDir := filepath.Join(tmpDir, "mcp")
	_ = os.MkdirAll(mcpDir, 0755)
	_ = os.WriteFile(filepath.Join(mcpDir, "test-mcp.yaml"), []byte("name: test-mcp\ncommand: npx\nargs: []\ntargets: [claude-code]"), 0644)

	// 2. Создаем Skills
	skillsDir := filepath.Join(tmpDir, "skills", "test-skill")
	_ = os.MkdirAll(skillsDir, 0755)
	_ = os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("---\nname: test-skill\ndescription: A test skill\ntargets: [antigravity]\n---\nContent of skill"), 0644)

	// 3. Создаем Workflows
	workflowsDir := filepath.Join(tmpDir, "workflows")
	_ = os.MkdirAll(workflowsDir, 0755)
	_ = os.WriteFile(filepath.Join(workflowsDir, "test-wf.yaml"), []byte("name: test-wf\ndescription: A test workflow\ntargets: [claude-code]"), 0644)

	// 4. Создаем Hooks
	hooksDir := filepath.Join(tmpDir, "hooks")
	_ = os.MkdirAll(hooksDir, 0755)
	_ = os.WriteFile(filepath.Join(hooksDir, "pre-commit.sh"), []byte("#!/bin/sh\necho test"), 0755)

	store := NewStore(tmpDir)
	store.DisableGlobal = true

	// Проверяем LoadSkills
	skills, err := store.LoadSkills()
	if err != nil {
		t.Errorf("LoadSkills failed: %v", err)
	}
	if len(skills) != 1 || skills[0].Header.Name != "test-skill" {
		t.Errorf("expected 1 skill with name 'test-skill', got: %+v", skills)
	}

	// Проверяем LoadWorkflows
	wfs, err := store.LoadWorkflows()
	if err != nil {
		t.Errorf("LoadWorkflows failed: %v", err)
	}
	if len(wfs) != 1 || wfs[0].Header.Name != "test-wf" {
		t.Errorf("expected 1 workflow with name 'test-wf', got: %+v", wfs)
	}

	// Проверяем LoadHooks
	hooks, err := store.LoadHooks()
	if err != nil {
		t.Errorf("LoadHooks failed: %v", err)
	}
	if len(hooks) != 1 || hooks[0].Header.Name != "pre-commit.sh" {
		t.Errorf("expected 1 hook with name 'pre-commit.sh', got: %+v", hooks)
	}
}

func TestInstallComponent(t *testing.T) {
	// Создаем временную директорию
	tmpDir, err := os.MkdirTemp("", "agentsync-install-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Сохраняем текущую директорию и меняем её на временную
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp: %v", err)
	}
	defer os.Chdir(oldWd)

	// Тестируем установку ComponentRule
	itemRule := RegistryItem{
		Name:        "test-rule-install",
		Type:        ComponentRule,
		Description: "A rule installed during test",
		SourceURL:   "https://github.com/example/rule",
	}

	if err := InstallComponent(itemRule); err != nil {
		t.Fatalf("failed to install rule: %v", err)
	}

	ruleFile := filepath.Join("rules", "test-rule-install.md")
	if _, err := os.Stat(ruleFile); os.IsNotExist(err) {
		t.Errorf("expected installed rule file to exist at %s, but it doesn't", ruleFile)
	}

	// Тестируем установку ComponentSkill
	itemSkill := RegistryItem{
		Name:        "test-skill-install",
		Type:        ComponentSkill,
		Description: "A skill installed during test",
		SourceURL:   "https://raw.githubusercontent.com/example/skill/main/SKILL.md",
	}

	if err := InstallComponent(itemSkill); err != nil {
		t.Fatalf("failed to install skill: %v", err)
	}

	skillFile := filepath.Join("skills", "test-skill-install", "SKILL.md")
	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		t.Errorf("expected installed skill file to exist at %s, but it doesn't", skillFile)
	}
}

func TestUpdateTargetsAndSaveSecrets(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentsync-update-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mcpDir := filepath.Join(tmpDir, "mcp")
	_ = os.MkdirAll(mcpDir, 0755)
	
	mcpFile := filepath.Join(mcpDir, "test-mcp.yaml")
	_ = os.WriteFile(mcpFile, []byte("name: test-mcp\ncommand: npx\nargs: []\ntargets: [claude-code]"), 0644)

	store := NewStore(tmpDir)
	err = store.UpdateMCPTargets("test-mcp", []AgentType{AgentClaudeDesktop, AgentAntigravity})
	if err != nil {
		t.Fatalf("failed to update mcp targets: %v", err)
	}

	// Проверяем, что обновилось в файле
	mcps, err := store.LoadMCPs()
	if err != nil {
		t.Fatalf("failed to load mcps: %v", err)
	}
	if len(mcps) != 1 {
		t.Fatalf("expected 1 mcp, got %d", len(mcps))
	}
	
	targets := mcps[0].Targets
	if len(targets) != 2 || targets[0] != AgentClaudeDesktop || targets[1] != AgentAntigravity {
		t.Errorf("unexpected updated targets: %v", targets)
	}

	// Проверяем SaveSecrets
	secrets := map[string]string{
		"OPENAI_API_KEY": "sk-12345",
		"GITHUB_TOKEN":   "ghp_67890",
	}
	err = SaveSecrets(tmpDir, secrets)
	if err != nil {
		t.Fatalf("failed to save secrets: %v", err)
	}

	loaded, err := LoadSecrets(tmpDir)
	if err != nil {
		t.Fatalf("failed to load secrets: %v", err)
	}

	if loaded["OPENAI_API_KEY"] != "sk-12345" || loaded["GITHUB_TOKEN"] != "ghp_67890" {
		t.Errorf("unexpected loaded secrets: %v", loaded)
	}
}

func TestAPIHandlers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentsync-api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Тестируем GET /api/status
	req, err := http.NewRequest("GET", "/api/status", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(apiStatusHandler(tmpDir))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Проверяем, что в ответе валидный JSON
	var agents []AgentInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &agents); err != nil {
		t.Errorf("handler returned invalid JSON: %v", err)
	}

	// Тестируем GET /api/components
	reqComp, err := http.NewRequest("GET", "/api/components", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rrComp := httptest.NewRecorder()
	handlerComp := http.HandlerFunc(apiComponentsHandler(tmpDir))
	handlerComp.ServeHTTP(rrComp, reqComp)

	if status := rrComp.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var components []AgentComponent
	if err := json.Unmarshal(rrComp.Body.Bytes(), &components); err != nil {
		t.Errorf("handler returned invalid JSON for components: %v", err)
	}
}
