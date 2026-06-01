package service_test

import (
	"agentsync/internal/domain"
	"agentsync/internal/service"
	"os"
	"path/filepath"
	"testing"
)

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
	itemRule := service.RegistryItem{
		Name:        "test-rule-install",
		Type:        domain.ComponentRule,
		Description: "A rule installed during test",
		SourceURL:   "https://github.com/example/rule",
	}

	if err := service.InstallComponent(itemRule); err != nil {
		t.Fatalf("failed to install rule: %v", err)
	}

	ruleFile := filepath.Join("rules", "test-rule-install.md")
	if _, err := os.Stat(ruleFile); os.IsNotExist(err) {
		t.Errorf("expected installed rule file to exist at %s, but it doesn't", ruleFile)
	}

	// Тестируем установку ComponentSkill
	itemSkill := service.RegistryItem{
		Name:        "test-skill-install",
		Type:        domain.ComponentSkill,
		Description: "A skill installed during test",
		SourceURL:   "https://raw.githubusercontent.com/example/skill/main/SKILL.md",
	}

	if err := service.InstallComponent(itemSkill); err != nil {
		t.Fatalf("failed to install skill: %v", err)
	}

	skillFile := filepath.Join("skills", "test-skill-install", "SKILL.md")
	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		t.Errorf("expected installed skill file to exist at %s, but it doesn't", skillFile)
	}
}
