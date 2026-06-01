package repository_test

import (
	"agentsync/internal/domain"
	"agentsync/internal/repository"
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
	mcpDir := filepath.Join(tmpDir, "data", "mcp")
	_ = os.MkdirAll(mcpDir, 0755)
	_ = os.WriteFile(filepath.Join(mcpDir, "test-mcp.yaml"), []byte("name: test-mcp\ncommand: npx\nargs: []\ntargets: [claude-code]"), 0644)

	// 2. Создаем Skills
	skillsDir := filepath.Join(tmpDir, "data", "skills", "test-skill")
	_ = os.MkdirAll(skillsDir, 0755)
	_ = os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("---\nname: test-skill\ndescription: A test skill\ntargets: [antigravity]\n---\nContent of skill"), 0644)

	// 3. Создаем Workflows
	workflowsDir := filepath.Join(tmpDir, "data", "workflows")
	_ = os.MkdirAll(workflowsDir, 0755)
	_ = os.WriteFile(filepath.Join(workflowsDir, "test-wf.yaml"), []byte("name: test-wf\ndescription: A test workflow\ntargets: [claude-code]"), 0644)

	// 4. Создаем Hooks
	hooksDir := filepath.Join(tmpDir, "data", "hooks")
	_ = os.MkdirAll(hooksDir, 0755)
	_ = os.WriteFile(filepath.Join(hooksDir, "pre-commit.sh"), []byte("#!/bin/sh\necho test"), 0755)

	store := repository.NewStore(tmpDir, repository.WithDisableGlobal(true))

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

func TestUpdateTargetsAndSaveSecrets(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentsync-update-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mcpDir := filepath.Join(tmpDir, "data", "mcp")
	_ = os.MkdirAll(mcpDir, 0755)
	
	mcpFile := filepath.Join(mcpDir, "test-mcp.yaml")
	_ = os.WriteFile(mcpFile, []byte("name: test-mcp\ncommand: npx\nargs: []\ntargets: [claude-code]"), 0644)

	store := repository.NewStore(tmpDir, repository.WithDisableGlobal(true))
	err = store.UpdateMCPTargets("test-mcp", []domain.AgentType{domain.AgentClaudeDesktop, domain.AgentAntigravity})
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
	if len(targets) != 2 || targets[0] != domain.AgentClaudeDesktop || targets[1] != domain.AgentAntigravity {
		t.Errorf("unexpected updated targets: %v", targets)
	}

	// Проверяем SaveSecrets
	secrets := map[string]string{
		"OPENAI_API_KEY": "sk-12345",
		"GITHUB_TOKEN":   "ghp_67890",
	}
	err = repository.SaveSecrets(tmpDir, secrets)
	if err != nil {
		t.Fatalf("failed to save secrets: %v", err)
	}

	loaded, err := repository.LoadSecrets(tmpDir)
	if err != nil {
		t.Fatalf("failed to load secrets: %v", err)
	}

	if loaded["OPENAI_API_KEY"] != "sk-12345" || loaded["GITHUB_TOKEN"] != "ghp_67890" {
		t.Errorf("unexpected loaded secrets: %v", loaded)
	}
}
