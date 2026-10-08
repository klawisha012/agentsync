package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestReleaseOrdering(t *testing.T) {
	tests := []struct {
		name    string
		local   string
		remote  string
		newer   bool
		known   bool
		changed bool
		localH  string
		remoteH string
	}{
		{name: "same numbers", local: "1.2.0", remote: "1.2.0", known: true},
		{name: "local patch", local: "1.2.1", remote: "1.2.0", newer: true, known: true, changed: true, localH: "aa", remoteH: "bb"},
		{name: "missing patch is equal", local: "1.2", remote: "1.2.0", known: false, changed: true},
		{name: "numeric minor", local: "1.10.0", remote: "1.9.0", newer: true, known: true, changed: true, localH: "aa", remoteH: "bb"},
		{name: "v prefix", local: "v1.3.0", remote: "1.2.9", newer: true, known: true, changed: true, localH: "aa", remoteH: "bb"},
		{name: "dev is unordered", local: "dev", remote: "1.0.0", known: true, changed: true, localH: "aa", remoteH: "bb"},
		{name: "date stamp", local: "20261008", remote: "20261007", newer: true, known: true, changed: true, localH: "aa", remoteH: "bb"},
		{name: "same hash", local: "dev", remote: "1.4.0", localH: "abc", remoteH: "ABC", known: true},
		{name: "same version without hash", local: "1.4.0", remote: "1.4.0", known: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newerRelease(tt.local, tt.remote); got != tt.newer {
				t.Fatalf("newer %v", got)
			}
			known, changed := releaseChanged(tt.local, tt.remote, tt.localH, tt.remoteH)
			if known != tt.known || changed != tt.changed {
				t.Fatalf("changed known %v %v", known, changed)
			}
		})
	}
}

func TestUpdateReplacesPublishedBinary(t *testing.T) {
	oldBody := fakeCLI(t, 'a')
	newBody := fakeCLI(t, 'b')
	exe := withExecutable(t, oldBody)
	srv := releaseServer(t, "1.2.3", map[string][]byte{mustReleaseName(t): newBody}, true)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)

	var out, err bytes.Buffer
	if code := Run([]string{"update"}, nil, &out, &err); code != 0 {
		t.Fatalf("update %d %s", code, err.String())
	}
	if out.String() != "AgentSync обновлён до 1.2.3.\n" {
		t.Fatalf("stdout %q", out.String())
	}
	if got := readFile(t, exe); got != string(newBody) {
		t.Fatal("binary was not replaced")
	}
	if _, statErr := os.Stat(exe + ".old"); !os.IsNotExist(statErr) {
		t.Fatal("old binary left behind")
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(exe), "update.lock")); !os.IsNotExist(statErr) {
		t.Fatal("lock left behind")
	}
}

func TestUpdateCheckDoesNotReplace(t *testing.T) {
	oldBody := fakeCLI(t, 'a')
	exe := withExecutable(t, oldBody)
	before := fileStamp(t, exe)
	srv := releaseServer(t, "1.2.3", map[string][]byte{mustReleaseName(t): fakeCLI(t, 'b')}, true)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)

	var out, err bytes.Buffer
	if code := Run([]string{"update", "--check"}, nil, &out, &err); code != 0 {
		t.Fatalf("check %d %s", code, err.String())
	}
	if out.String() != "Доступна версия 1.2.3.\n" {
		t.Fatalf("stdout %q", out.String())
	}
	if got := readFile(t, exe); got != string(oldBody) {
		t.Fatal("check replaced the binary")
	}
	if fileStamp(t, exe) != before {
		t.Fatal("check rewrote the binary")
	}
}

func TestUpdateSameHashSkipsDownload(t *testing.T) {
	body := fakeCLI(t, 'a')
	exe := withExecutable(t, body)
	var binaryHits atomic.Int32
	name := mustReleaseName(t)
	sum := sha256.Sum256(body)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cli/"+name {
			binaryHits.Add(1)
		}
		payload, err := json.Marshal(map[string]any{
			"version": "9.9.9",
			"sha256":  map[string]string{name: hex.EncodeToString(sum[:])},
			"note":    "later",
		})
		if err != nil {
			t.Errorf("json %v", err)
		}
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)

	var out, err bytes.Buffer
	if code := Run([]string{"update"}, nil, &out, &err); code != 0 {
		t.Fatalf("update %d %s", code, err.String())
	}
	if out.String() != "Уже стоит последняя версия.\n" {
		t.Fatalf("stdout %q", out.String())
	}
	if binaryHits.Load() != 0 {
		t.Fatalf("binary downloads %d", binaryHits.Load())
	}
	if got := readFile(t, exe); got != string(body) {
		t.Fatal("binary changed")
	}
}

func TestUpdateOldServerComparesFile(t *testing.T) {
	body := fakeCLI(t, 'a')
	exe := withExecutable(t, body)
	same := releaseServer(t, "", map[string][]byte{mustReleaseName(t): body}, false)
	t.Setenv("AGENTSYNC_SERVER", same.URL)
	var out, err bytes.Buffer
	before := fileStamp(t, exe)
	if code := Run([]string{"update"}, nil, &out, &err); code == 0 {
		t.Fatalf("same succeeded without a manifest: %s", out.String())
	}
	if !bytes.Contains(err.Bytes(), []byte("Не удалось проверить обновление.")) {
		t.Fatalf("same stderr %q", err.String())
	}
	if got := readFile(t, exe); got != string(body) || fileStamp(t, exe) != before {
		t.Fatal("missing manifest changed the binary")
	}

	next := fakeCLI(t, 'c')
	other := releaseServer(t, "", map[string][]byte{mustReleaseName(t): next}, false)
	t.Setenv("AGENTSYNC_SERVER", other.URL)
	out.Reset()
	err.Reset()
	if code := Run([]string{"update"}, nil, &out, &err); code == 0 {
		t.Fatalf("other succeeded without a manifest: %s", out.String())
	}
	if got := readFile(t, exe); got != string(body) {
		t.Fatal("missing manifest replaced the binary")
	}
}

func TestUpdateRejectsPage(t *testing.T) {
	body := fakeCLI(t, 'a')
	exe := withExecutable(t, body)
	page := bytes.Repeat([]byte("x"), 80)
	copy(page, []byte("<html>"))
	srv := releaseServer(t, "", map[string][]byte{mustReleaseName(t): page}, true)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)

	var out, err bytes.Buffer
	if code := Run([]string{"update"}, nil, &out, &err); code != 1 {
		t.Fatalf("code %d %q", code, err.String())
	}
	if !bytes.Contains(err.Bytes(), []byte("не похож на программу")) {
		t.Fatalf("stderr %q", err.String())
	}
	if got := readFile(t, exe); got != string(body) {
		t.Fatal("bad download replaced the binary")
	}
}

func TestUpdateRejectsForeignRedirect(t *testing.T) {
	exe := withExecutable(t, fakeCLI(t, 'a'))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.com/cli/version", http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)

	var out, err bytes.Buffer
	if code := Run([]string{"update"}, nil, &out, &err); code != 1 {
		t.Fatalf("code %d", code)
	}
	if !bytes.Contains(err.Bytes(), []byte("Не удалось проверить обновление.")) {
		t.Fatalf("stderr %q", err.String())
	}
	if got := readFile(t, exe); got != string(fakeCLI(t, 'a')) {
		t.Fatal("binary changed")
	}
}

func TestUpdateKeepsExplicitDowngrade(t *testing.T) {
	setVersion(t, "2.0.0")
	exe := withExecutable(t, fakeCLI(t, 'a'))
	next := fakeCLI(t, 'b')
	srv := releaseServer(t, "1.0.0", map[string][]byte{mustReleaseName(t): next}, true)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)

	var out, err bytes.Buffer
	if code := Run([]string{"update"}, nil, &out, &err); code != 0 {
		t.Fatalf("update %d %s", code, err.String())
	}
	if got := readFile(t, exe); got != string(next) {
		t.Fatal("explicit update kept the newer local build")
	}
}

func TestAutoUpdateReplacesInstallCopy(t *testing.T) {
	body := fakeCLI(t, 'a')
	next := fakeCLI(t, 'b')
	exe := withInstallExecutable(t, body)
	var hits atomic.Int32
	name := mustReleaseName(t)
	sum := sha256.Sum256(next)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path == "/cli/version" {
			payload, err := json.Marshal(map[string]any{
				"version": "1.2.3",
				"sha256":  map[string]string{name: hex.EncodeToString(sum[:])},
			})
			if err != nil {
				t.Errorf("json %v", err)
			}
			_, _ = w.Write(payload)
			return
		}
		_, _ = w.Write(next)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)

	var out, err bytes.Buffer
	if code := Run([]string{"push"}, nil, &out, &err); code != 2 {
		t.Fatalf("push %d %s", code, err.String())
	}
	if !bytes.Contains(err.Bytes(), []byte("AgentSync обновлён до 1.2.3.")) {
		t.Fatalf("stderr %q", err.String())
	}
	if !bytes.Contains(err.Bytes(), []byte("Назовите ИИ-агента.")) {
		t.Fatalf("command stderr %q", err.String())
	}
	if got := readFile(t, exe); got != string(next) {
		t.Fatal("auto update did not replace the install copy")
	}
	seen := hits.Load()
	err.Reset()
	if code := Run([]string{"push"}, nil, &out, &err); code != 2 {
		t.Fatalf("second %d", code)
	}
	if hits.Load() != seen {
		t.Fatalf("second check hit the server: %d then %d", seen, hits.Load())
	}
	if bytes.Contains(err.Bytes(), []byte("обновлён")) {
		t.Fatalf("repeated notice %q", err.String())
	}
}

func TestAutoUpdateSkipsNewerLocal(t *testing.T) {
	setVersion(t, "2.0.0")
	exe := withInstallExecutable(t, fakeCLI(t, 'a'))
	var hits atomic.Int32
	name := mustReleaseName(t)
	next := fakeCLI(t, 'b')
	sum := sha256.Sum256(next)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		payload, err := json.Marshal(map[string]any{
			"version": "1.0.0",
			"sha256":  map[string]string{name: hex.EncodeToString(sum[:])},
		})
		if err != nil {
			t.Errorf("json %v", err)
		}
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)

	var err bytes.Buffer
	maybeAutoUpdate(&err)
	if err.Len() != 0 {
		t.Fatalf("stderr %q", err.String())
	}
	if got := readFile(t, exe); bytes.Equal([]byte(got), next) {
		t.Fatal("auto update downgraded a newer build")
	}
	if hits.Load() == 0 {
		t.Fatal("version was not checked")
	}
}

func TestAutoUpdateDisabled(t *testing.T) {
	exe := withInstallExecutable(t, fakeCLI(t, 'a'))
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "no", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_AUTO_UPDATE", "0")

	var out, err bytes.Buffer
	if code := Run([]string{"push"}, nil, &out, &err); code != 2 {
		t.Fatalf("push %d", code)
	}
	if hits.Load() != 0 {
		t.Fatalf("hits %d", hits.Load())
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(exe), "update-check")); !os.IsNotExist(statErr) {
		t.Fatal("disabled check wrote a stamp")
	}
}

func TestVersionCommand(t *testing.T) {
	setVersion(t, "1.2.3")
	var out, err bytes.Buffer
	if code := Run([]string{"version"}, nil, &out, &err); code != 0 {
		t.Fatalf("version %d %s", code, err.String())
	}
	if out.String() != "agentsync 1.2.3\n" || err.Len() != 0 {
		t.Fatalf("stdout %q stderr %q", out.String(), err.String())
	}
	out.Reset()
	if code := Run([]string{"version", "extra"}, nil, &out, &err); code != 2 {
		t.Fatalf("extra %d", code)
	}
	if code := Run([]string{"update", "--force"}, nil, &out, &err); code != 2 {
		t.Fatalf("force %d", code)
	}
	if !bytes.Contains(err.Bytes(), []byte("Неизвестный аргумент")) {
		t.Fatalf("stderr %q", err.String())
	}
}

func TestUpdateHelp(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"help", "update"}, nil, &out, &err); code != 0 {
		t.Fatalf("help %d", code)
	}
	if !bytes.Contains(out.Bytes(), []byte("agentsync update [--check]")) {
		t.Fatalf("stdout %q", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("AGENTSYNC_AUTO_UPDATE")) {
		t.Fatal("help missed the switch")
	}
}

func fakeCLI(t *testing.T, fill byte) []byte {
	t.Helper()
	body := bytes.Repeat([]byte{fill}, 80)
	switch runtime.GOOS {
	case "windows":
		copy(body, []byte("MZ"))
	case "linux":
		copy(body, []byte{0x7f, 'E', 'L', 'F'})
	case "darwin":
		copy(body, []byte{0xfe, 0xed, 0xfa, 0xcf})
	default:
		t.Fatal("unsupported os")
	}
	return body
}

func mustReleaseName(t *testing.T) string {
	t.Helper()
	name, ok := releaseFileName()
	if !ok {
		t.Fatal("unsupported arch")
	}
	return name
}

func withExecutable(t *testing.T, body []byte) string {
	t.Helper()
	return installExecutable(t, t.TempDir(), body)
}

func withInstallExecutable(t *testing.T, body []byte) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	return installExecutable(t, filepath.Join(home, ".agentsync"), body)
}

func installExecutable(t *testing.T, dir string, body []byte) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "agentsync.exe")
	if err := os.WriteFile(exe, body, 0o755); err != nil {
		t.Fatal(err)
	}
	prev := executablePath
	executablePath = func() (string, error) { return exe, nil }
	t.Cleanup(func() { executablePath = prev })
	return exe
}

func setVersion(t *testing.T, value string) {
	t.Helper()
	prev := Version
	Version = value
	t.Cleanup(func() { Version = prev })
}

func releaseServer(t *testing.T, version string, files map[string][]byte, withManifest bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cli/version" {
			if !withManifest {
				http.NotFound(w, r)
				return
			}
			sums := map[string]string{}
			for name, body := range files {
				sum := sha256.Sum256(body)
				sums[name] = hex.EncodeToString(sum[:])
			}
			payload, err := json.Marshal(map[string]any{
				"version": version,
				"sha256":  sums,
				"note":    "later",
			})
			if err != nil {
				t.Errorf("json %v", err)
			}
			_, _ = w.Write(payload)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/cli/")
		body, ok := files[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func fileStamp(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.ModTime().UnixNano()
}

func TestUpdateKeepsAPIPrefix(t *testing.T) {
	body := fakeCLI(t, 'a')
	exe := withExecutable(t, body)
	name := mustReleaseName(t)
	sum := sha256.Sum256(body)
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		if r.URL.Path != "/api/cli/version" {
			http.NotFound(w, r)
			return
		}
		payload, err := json.Marshal(map[string]any{
			"version": "1.2.3",
			"sha256":  map[string]string{name: hex.EncodeToString(sum[:])},
		})
		if err != nil {
			t.Errorf("json %v", err)
		}
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AGENTSYNC_SERVER", srv.URL+"/api")

	var out, err bytes.Buffer
	if code := Run([]string{"update"}, nil, &out, &err); code != 0 {
		t.Fatalf("update %d %s path %s", code, err.String(), got)
	}
	if got != "/api/cli/version" {
		t.Fatalf("path %s", got)
	}
	if out.String() != "Уже стоит последняя версия.\n" {
		t.Fatalf("stdout %q", out.String())
	}
	if readFile(t, exe) != string(body) {
		t.Fatal("binary changed")
	}
}

func TestUpdateDue(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "agentsync.exe")
	if err := os.WriteFile(exe, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0)
	if !updateDue(exe, now) {
		t.Fatal("missing stamp should be due")
	}
	stampUpdate(exe, now.Add(time.Hour))
	if updateDue(exe, now) {
		t.Fatal("future stamp was due")
	}
	if !updateDue(exe, now.Add(2*time.Hour)) {
		t.Fatal("expired stamp was not due")
	}
}
