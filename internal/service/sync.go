package service

import (
	"archive/zip"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agentsync/internal/domain"

	"github.com/hashicorp/mdns"
)

// P2PShare запускает HTTP-сервер для раздачи и анонсирует его через mDNS
func P2PShare(repoPath string, pin string, port int, logger Logger) error {
	// 1. Создаем легковесный HTTP-сервер
	mux := http.NewServeMux()
	mux.HandleFunc("/pull", func(w http.ResponseWriter, r *http.Request) {
		// Проверка авторизационного пин-кода
		reqPin := r.URL.Query().Get("pin")
		if reqPin != pin {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, "Неверный PIN-код авторизации!")
			return
		}

		// Создаем zip-архив на лету в памяти
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=\"agentsync-config.zip\"")

		zipWriter := zip.NewWriter(w)
		defer zipWriter.Close()

		// Рекурсивно упаковываем репозиторий, исключая secrets.env и backups
		err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(repoPath, path)
			if err != nil {
				return err
			}

			// Исключаем secrets.env, бэкапы, Git-папку и сам бинарник
			if relPath == "secrets.env" || strings.HasPrefix(relPath, "backups") || strings.HasPrefix(relPath, ".git") || strings.HasSuffix(relPath, ".exe") {
				return nil
			}

			zipFile, err := zipWriter.Create(relPath)
			if err != nil {
				return err
			}

			fsFile, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fsFile.Close()

			_, err = io.Copy(zipFile, fsFile)
			return err
		})

		if err != nil {
			if logger != nil {
				logger.Log(fmt.Sprintf("Ошибка сборки zip: %v", err), "error")
			}
		}
	})

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("не удалось запустить сервер на порту %d: %w", port, err)
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port

	// 2. Анонсируем сервис через mDNS
	host, _ := os.Hostname()
	serviceName := fmt.Sprintf("agentsync-%s", host)
	
	if logger != nil {
		logger.Log(fmt.Sprintf("Сервер запущен на порту %d", actualPort), "info")
		logger.Log(fmt.Sprintf("Анонсирование P2P AirDrop по mDNS под именем %s.local...", serviceName), "info")
		logger.Log(fmt.Sprintf("Одноразовый PIN-код для друга: %s", pin), "info")
	}

	ips, _ := localIPs()
	if logger != nil {
		logger.Log("Доступные адреса в локальной сети:", "info")
		for _, ip := range ips {
			logger.Log(fmt.Sprintf("   http://%s:%d/pull?pin=%s", ip, actualPort, pin), "info")
		}
	}

	service, err := mdns.NewMDNSService(
		serviceName,
		"_agentsync._tcp",
		"local.",
		"",
		actualPort,
		nil,
		[]string{"AgentSync config sharing"},
	)
	if err != nil {
		return fmt.Errorf("ошибка создания mDNS сервиса: %w", err)
	}

	mdnsServer, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return fmt.Errorf("ошибка запуска mDNS сервера: %w", err)
	}
	defer mdnsServer.Shutdown()

	// Запускаем HTTP-сервер в блокирующем режиме
	return http.Serve(listener, mux)
}

// DiscoverServices сканирует сеть на наличие раздающих нод AgentSync
func DiscoverServices(timeout time.Duration) ([]string, error) {
	var nodes []string
	entriesCh := make(chan *mdns.ServiceEntry, 10)

	go func() {
		for entry := range entriesCh {
			nodes = append(nodes, fmt.Sprintf("%s (%s:%d)", entry.Name, entry.AddrV4.String(), entry.Port))
		}
	}()

	params := mdns.DefaultParams("_agentsync._tcp")
	params.Entries = entriesCh
	params.WantUnicastResponse = false
	params.Timeout = timeout

	err := mdns.Query(params)
	if err != nil {
		return nil, err
	}

	close(entriesCh)
	// Даем немного времени горутине на запись
	time.Sleep(100 * time.Millisecond)

	return nodes, nil
}

// P2PPull скачивает zip-архив конфигурации и разворачивает его локально
func P2PPull(targetHost string, targetPort int, pin string, repoPath string, logger Logger) error {
	if logger != nil {
		logger.Log(fmt.Sprintf("Подключение к P2P ноде %s:%d...", targetHost, targetPort), "info")
	}

	url := fmt.Sprintf("http://%s:%d/pull?pin=%s", targetHost, targetPort, pin)
	
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("отказано в доступе: неверный PIN-код")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("сервер вернул ошибку: %s", resp.Status)
	}

	// Сохраняем во временный zip-файл
	tmpFile, err := os.CreateTemp("", "agentsync-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err = io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}

	// Распаковываем
	_, _ = tmpFile.Seek(0, 0)
	zipReader, err := zip.NewReader(tmpFile, resp.ContentLength)
	if err != nil {
		fi, statErr := tmpFile.Stat()
		if statErr != nil {
			return statErr
		}
		zipReader, err = zip.NewReader(tmpFile, fi.Size())
		if err != nil {
			return err
		}
	}

	if logger != nil {
		logger.Log(fmt.Sprintf("Распаковка канонических файлов в репозиторий %s...", repoPath), "info")
	}

	for _, file := range zipReader.File {
		path := filepath.Join(repoPath, file.Name)

		// Убеждаемся, что пути безопасны (защита от Zip Slip)
		if !strings.HasPrefix(path, filepath.Clean(repoPath)) {
			continue
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(path), 0755)

		dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		srcFile, err := file.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		_, err = io.Copy(dstFile, srcFile)
		dstFile.Close()
		srcFile.Close()
		if err != nil {
			return err
		}
	}

	if logger != nil {
		logger.Log("Успешно импортировано конфигураций по P2P AirDrop!", "info")
	}
	return nil
}

// localIPs возвращает список IPv4 адресов локальных интерфейсов
func localIPs() ([]string, error) {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			ips = append(ips, ipnet.IP.String())
		}
	}

	return ips, nil
}

// SyncComponentSource копирует файлы компонентов между локальным CWD и глобальным ~/.agents
func SyncComponentSource(repoPath string, comp domain.AgentComponent, toGlobal bool) error {
	var subDir string
	var fileName string
	switch comp.Type {
	case domain.ComponentMCP:
		subDir = "mcp"
		fileName = comp.Name + ".yaml"
	case domain.ComponentRule:
		subDir = "rules"
		fileName = comp.Name + ".md"
	case domain.ComponentSkill:
		subDir = "skills"
		fileName = filepath.Join(comp.Name, "SKILL.md")
	case domain.ComponentWorkflow:
		subDir = "workflows"
		fileName = comp.Name
	case domain.ComponentHook:
		subDir = "hooks"
		fileName = comp.Name
	default:
		return fmt.Errorf("неподдерживаемый тип")
	}

	localPath := filepath.Join(repoPath, subDir, fileName)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	globalPath := filepath.Join(home, ".agents", subDir, fileName)

	var src, dst string
	if toGlobal {
		src = localPath
		dst = globalPath
	} else {
		src = globalPath
		dst = localPath
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}
