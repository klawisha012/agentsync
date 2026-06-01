package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// CreateBackup делает резервную копию файла перед слиянием
func CreateBackup(filePath string, agent AgentType) error {
	// Если файл не существует, бэкапить нечего
	if !fileExists(filePath) {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("ошибка получения домашней папки: %w", err)
	}

	// Формируем уникальный таймстемп
	timestamp := time.Now().Format("2006-01-02_150405")
	backupDir := filepath.Join(home, ".agentsync", "backups", timestamp)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания папки бэкапов %s: %w", backupDir, err)
	}

	baseName := filepath.Base(filePath)
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s_%s", string(agent), baseName))

	// Копируем содержимое файла
	src, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}
