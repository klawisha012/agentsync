package web

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

// StartWebServer запускает HTTP-сервер для Web GUI
func StartWebServer(port int, repoPath string) error {
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
	http.Handle("/api/components/update-targets", corsMiddleware(http.HandlerFunc(apiUpdateTargetsHandler(repoPath))))
	http.Handle("/api/components/sync-source", corsMiddleware(http.HandlerFunc(apiSyncSourceHandler(repoPath))))
	http.Handle("/api/secrets", corsMiddleware(http.HandlerFunc(apiSecretsHandler(repoPath))))
	http.Handle("/api/deploy", corsMiddleware(http.HandlerFunc(apiDeployHandler(repoPath))))
	http.Handle("/api/manifest", corsMiddleware(http.HandlerFunc(apiManifestHandler(repoPath))))
	http.Handle("/api/bundles", corsMiddleware(http.HandlerFunc(apiBundlesHandler(repoPath))))
	http.Handle("/api/bundles/sync", corsMiddleware(http.HandlerFunc(apiBundlesSyncHandler(repoPath))))
	http.Handle("/api/bundles/share", corsMiddleware(http.HandlerFunc(apiBundlesShareHandler(repoPath))))
	http.Handle("/api/registry/search", corsMiddleware(http.HandlerFunc(apiRegistrySearchHandler(repoPath))))
	http.Handle("/api/registry/install", corsMiddleware(http.HandlerFunc(apiRegistryInstallHandler(repoPath))))

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("[+] Запуск Web GUI на http://localhost%s...\n", addr)
	
	return http.ListenAndServe(addr, nil)
}
