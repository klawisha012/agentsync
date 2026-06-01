package service

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RuntimeManager управляет изолированными рантаймами (Node.js/Python)
type RuntimeManager struct {
	basePath string
	logger   Logger
}

func NewRuntimeManager(logger Logger) (*RuntimeManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	base := filepath.Join(home, ".agentsync", "runtime")
	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, err
	}
	return &RuntimeManager{basePath: base, logger: logger}, nil
}

func (m *RuntimeManager) log(msg string, level string) {
	if m.logger != nil {
		m.logger.Log(msg, level)
	}
}

// ResolveRuntimePath возвращает путь к исполняемому файлу рантайма, при необходимости скачивая его
func (m *RuntimeManager) ResolveRuntimePath(runtimeName string) (string, error) {
	if runtimeName == "" {
		// По умолчанию используем системный PATH
		return "", nil
	}

	parts := strings.SplitN(runtimeName, "@", 2)
	lang := parts[0]
	version := "20"
	if len(parts) > 1 {
		version = parts[1]
	}

	if lang == "nodejs" || lang == "node" {
		return m.ensureNodeJS(version)
	}

	// Для Python в MVP возвращаем системный python (graceful fallback)
	pythonPath, err := exec.LookPath("python")
	if err == nil {
		return pythonPath, nil
	}
	return "python", nil
}

// ensureNodeJS гарантирует наличие портативного Node.js нужной версии
func (m *RuntimeManager) ensureNodeJS(version string) (string, error) {
	nodeDir := filepath.Join(m.basePath, "node", "v"+version)
	nodeExe := filepath.Join(nodeDir, "node.exe") // На Windows

	if serviceFileExists(nodeExe) {
		return nodeExe, nil
	}

	// Сначала проверяем, есть ли системный Node.js той же мажорной версии (ожидаем ускорение)
	sysNode, err := exec.LookPath("node")
	if err == nil {
		// Проверяем версию
		cmd := exec.Command(sysNode, "-v")
		output, errOut := cmd.Output()
		if errOut == nil && strings.HasPrefix(strings.TrimSpace(string(output)), "v"+version) {
			m.log(fmt.Sprintf("Обнаружен подходящий системный Node.js (%s)", strings.TrimSpace(string(output))), "info")
			return sysNode, nil
		}
	}

	// Если системного нет, скачиваем портативный Node.js
	m.log(fmt.Sprintf("Скачивание портативного Node.js v%s...", version), "info")
	url := fmt.Sprintf("https://nodejs.org/dist/v%s.11.0/node-v%s.11.0-win-x64.zip", version, version)
	
	tmpZip := filepath.Join(os.TempDir(), fmt.Sprintf("node-v%s.zip", version))
	if err := downloadFile(url, tmpZip); err != nil {
		return "", fmt.Errorf("ошибка скачивания Node.js: %w", err)
	}
	defer os.Remove(tmpZip)

	m.log(fmt.Sprintf("Распаковка в изолированное окружение %s...", nodeDir), "info")
	if err := unzipNode(tmpZip, filepath.Dir(nodeDir), "node-v"+version+".11.0-win-x64", "v"+version); err != nil {
		return "", fmt.Errorf("ошибка распаковки Node.js: %w", err)
	}

	if serviceFileExists(nodeExe) {
		m.log("Портативный Node.js успешно установлен!", "info")
		return nodeExe, nil
	}

	return "", fmt.Errorf("не удалось верифицировать установку Node.js по пути %s", nodeExe)
}

// InstallMCPPackage устанавливает npm-пакет локально в изолированное окружение рантайма
func (m *RuntimeManager) InstallMCPPackage(runtimeExe string, packageName string) (string, error) {
	if packageName == "" {
		return "", nil
	}

	runtimeDir := filepath.Dir(runtimeExe)
	npmCli := filepath.Join(runtimeDir, "node_modules", "npm", "bin", "npm-cli.js")
	
	// Если мы используем системный Node.js, то устанавливаем во внутреннюю изолированную директорию
	var targetInstallDir string
	if strings.Contains(runtimeExe, "runtime") {
		targetInstallDir = runtimeDir
	} else {
		// Системный node -> пишем в наш изолированный кэш
		targetInstallDir = filepath.Join(m.basePath, "node", "global")
		_ = os.MkdirAll(targetInstallDir, 0755)
	}

	packageCheckPath := filepath.Join(targetInstallDir, "node_modules", packageName)
	if serviceFileExists(filepath.Join(packageCheckPath, "package.json")) {
		return packageCheckPath, nil
	}

	m.log(fmt.Sprintf("Установка npm пакета %s в изолированное окружение...", packageName), "info")

	var cmd *exec.Cmd
	if strings.Contains(runtimeExe, "runtime") && serviceFileExists(npmCli) {
		// Используем портативный npm
		cmd = exec.Command(runtimeExe, npmCli, "install", "--prefix", targetInstallDir, packageName)
	} else {
		// Используем системный npm / npx
		npmPath, err := exec.LookPath("npm")
		if err == nil {
			cmd = exec.Command(npmPath, "install", "--prefix", targetInstallDir, packageName)
		} else {
			// fallback на npx
			npxPath, errNpx := exec.LookPath("npx")
			if errNpx == nil {
				cmd = exec.Command(npxPath, "--prefix", targetInstallDir, "install", packageName)
			} else {
				return "", fmt.Errorf("системный npm/npx не найден. Установите Node.js")
			}
		}
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ошибка установки npm-пакета %s: %w", packageName, err)
	}

	m.log(fmt.Sprintf("Пакет %s успешно установлен!", packageName), "info")
	return packageCheckPath, nil
}

func downloadFile(url string, filepath string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("сервер вернул статус: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}

func unzipNode(src string, dest string, expectedSubDir string, finalDirName string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		relPath := f.Name
		if strings.HasPrefix(relPath, expectedSubDir) {
			relPath = finalDirName + relPath[len(expectedSubDir):]
		}

		fpath := filepath.Join(dest, relPath)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func serviceFileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
