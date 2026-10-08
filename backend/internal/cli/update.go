package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	updateInterval = 24 * time.Hour
	updateRetry    = time.Hour
	updateLockTTL  = 10 * time.Minute
	maxCLIBytes    = 64 << 20
	minCLIBytes    = 64
)

var (
	errUpdateCheck    = errors.New("Не удалось проверить обновление.")
	errUpdateDownload = errors.New("Не удалось скачать agentsync.")
	errUpdateBinary   = errors.New("Скачанный файл не похож на программу agentsync.")
	errUpdateReplace  = errors.New("Не удалось заменить agentsync.")
	errUpdateBusy     = errors.New("Обновление уже выполняется.")
	errUpdateArch     = errors.New("Эта архитектура не поддерживается.")
	errUpdateServer   = errors.New("Адрес AGENTSYNC_SERVER не подходит.")
	errUpdateFind     = errors.New("Не удалось найти файл agentsync.")
)

type releaseMode int

const (
	releaseApply releaseMode = iota
	releaseCheck
	releaseAuto
)

type releaseKind int

const (
	releaseCurrent releaseKind = iota
	releaseAvailable
	releaseReplaced
	releaseKeptNewer
)

type releaseOutcome struct {
	kind   releaseKind
	remote string
}

type releaseManifest struct {
	Version string            `json:"version"`
	SHA256  map[string]string `json:"sha256"`
}

var executablePath = os.Executable

func updateCommand(args []string, stdout io.Writer) error {
	mode := releaseApply
	for _, arg := range args {
		if arg == "--check" {
			mode = releaseCheck
			continue
		}
		return usageError{text: "Неизвестный аргумент «" + arg + "»."}
	}
	exe, err := currentExecutable()
	if err != nil {
		return errUpdateFind
	}
	unlock, ok := acquireUpdateLock(filepath.Dir(exe))
	if !ok {
		return errUpdateBusy
	}
	defer unlock()
	_ = os.Remove(exe + ".old")
	outcome, err := syncRelease(context.Background(), apiOrigin(), exe, mode)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, outcome.text())
	return nil
}

func maybeAutoUpdate(stderr io.Writer) {
	if autoUpdateOff() {
		return
	}
	exe, err := currentExecutable()
	if err != nil || !inInstallDir(exe) || !updateDue(exe, time.Now()) {
		return
	}
	unlock, ok := acquireUpdateLock(filepath.Dir(exe))
	if !ok {
		return
	}
	defer unlock()
	_ = os.Remove(exe + ".old")
	outcome, err := syncRelease(context.Background(), apiOrigin(), exe, releaseAuto)
	if err != nil {
		wait := updateRetry
		if errors.Is(err, errUpdateReplace) || errors.Is(err, errUpdateBinary) {
			fmt.Fprintln(stderr, "Доступна новая версия AgentSync. Запустите agentsync update.")
			wait = updateInterval
		}
		stampUpdate(exe, time.Now().Add(wait))
		return
	}
	stampUpdate(exe, time.Now().Add(updateInterval))
	if outcome.kind == releaseReplaced {
		fmt.Fprintln(stderr, autoUpdatedLine(outcome.remote))
	}
}

func (item releaseOutcome) text() string {
	switch item.kind {
	case releaseAvailable:
		if versionVisible(item.remote) {
			return "Доступна версия " + item.remote + "."
		}
		return "Доступна новая версия AgentSync."
	case releaseReplaced:
		if versionVisible(item.remote) {
			return "AgentSync обновлён до " + item.remote + "."
		}
		return "AgentSync обновлён."
	default:
		return "Уже стоит последняя версия."
	}
}

func autoUpdatedLine(remote string) string {
	if versionVisible(remote) {
		return "AgentSync обновлён до " + remote + ". Эта команда ещё выполняется предыдущей версией."
	}
	return "AgentSync обновлён. Эта команда ещё выполняется предыдущей версией."
}

func syncRelease(ctx context.Context, server, exe string, mode releaseMode) (releaseOutcome, error) {
	name, ok := releaseFileName()
	if !ok {
		return releaseOutcome{}, errUpdateArch
	}
	base, err := parseAPI(server)
	if err != nil {
		return releaseOutcome{}, err
	}
	client := releaseClient(base)
	manifest, found, err := getManifest(ctx, client, base)
	if err != nil {
		return releaseOutcome{}, err
	}
	localHash, _ := fileSHA256(exe)
	remoteHash := ""
	remoteVer := ""
	if found {
		remoteVer = strings.TrimSpace(manifest.Version)
		if manifest.SHA256 != nil {
			remoteHash = manifest.SHA256[name]
		}
	}
	known, changed := releaseChanged(Version, remoteVer, localHash, remoteHash)
	explicit := mode != releaseAuto
	if known && !changed {
		return releaseOutcome{kind: releaseCurrent, remote: remoteVer}, nil
	}
	if known && changed && !explicit && newerRelease(Version, remoteVer) {
		return releaseOutcome{kind: releaseKeptNewer, remote: remoteVer}, nil
	}
	if known && changed && mode == releaseCheck {
		return releaseOutcome{kind: releaseAvailable, remote: remoteVer}, nil
	}
	body, err := getBinary(ctx, client, base, name)
	if err != nil {
		return releaseOutcome{}, err
	}
	if !binaryMagic(runtime.GOOS, body) {
		return releaseOutcome{}, errUpdateBinary
	}
	sum := sha256.Sum256(body)
	if localHash != "" && strings.EqualFold(localHash, hex.EncodeToString(sum[:])) {
		return releaseOutcome{kind: releaseCurrent, remote: remoteVer}, nil
	}
	if mode == releaseCheck {
		return releaseOutcome{kind: releaseAvailable, remote: remoteVer}, nil
	}
	if !explicit && newerRelease(Version, remoteVer) {
		return releaseOutcome{kind: releaseKeptNewer, remote: remoteVer}, nil
	}
	if err := replaceExecutable(exe, body); err != nil {
		return releaseOutcome{}, err
	}
	return releaseOutcome{kind: releaseReplaced, remote: remoteVer}, nil
}

func releaseChanged(localVer, remoteVer, localHash, remoteHash string) (bool, bool) {
	if remoteHash != "" && localHash != "" {
		return true, !strings.EqualFold(localHash, remoteHash)
	}
	if remoteVer != "" && localVer != "" && localVer != "dev" && remoteVer == localVer {
		return true, false
	}
	return false, true
}

func newerRelease(local, remote string) bool {
	left, lok := parseRelease(local)
	right, rok := parseRelease(remote)
	if !lok || !rok {
		return false
	}
	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	for i := range n {
		lv, rv := 0, 0
		if i < len(left) {
			lv = left[i]
		}
		if i < len(right) {
			rv = right[i]
		}
		if lv != rv {
			return lv > rv
		}
	}
	return false
}

func parseRelease(value string) ([]int, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	if value == "" || value == "dev" {
		return nil, false
	}
	parts := strings.Split(value, ".")
	nums := make([]int, 0, len(parts))
	for _, part := range parts {
		if part == "" || len(part) > 9 {
			return nil, false
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return nil, false
		}
		nums = append(nums, number)
	}
	return nums, true
}

func versionVisible(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || value == "dev" || len(value) > 80 || strings.ContainsAny(value, "\r\n\t ") {
		return false
	}
	return true
}

func releaseFileName() (string, bool) {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
	default:
		return "", false
	}
	switch runtime.GOARCH {
	case "amd64", "arm64":
	default:
		return "", false
	}
	name := "agentsync-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name, true
}

func apiOrigin() string {
	server := strings.TrimSpace(os.Getenv("AGENTSYNC_SERVER"))
	if server == "" {
		server = "https://zwarder.ru/api"
	}
	return strings.TrimRight(server, "/")
}

func parseAPI(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, errUpdateServer
	}
	return parsed, nil
}

func releaseClient(base *url.URL) *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 || req.URL.Scheme != base.Scheme || !strings.EqualFold(req.URL.Host, base.Host) {
				return errUpdateDownload
			}
			return nil
		},
	}
}

func getManifest(ctx context.Context, client *http.Client, base *url.URL) (releaseManifest, bool, error) {
	checkCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	body, status, err := getPath(checkCtx, client, base, "version", 1<<20)
	if err != nil {
		return releaseManifest{}, false, errUpdateCheck
	}
	// Старый сервер не публикует описание версии и отвечает 404. Тогда сравниваем сам файл.
	if status == http.StatusNotFound {
		return releaseManifest{}, false, nil
	}
	if status != http.StatusOK {
		return releaseManifest{}, false, errUpdateCheck
	}
	var manifest releaseManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return releaseManifest{}, false, nil
	}
	return manifest, true, nil
}

func getBinary(ctx context.Context, client *http.Client, base *url.URL, name string) ([]byte, error) {
	loadCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	body, status, err := getPath(loadCtx, client, base, name, maxCLIBytes)
	if err != nil || status != http.StatusOK {
		return nil, errUpdateDownload
	}
	return body, nil
}

func getPath(ctx context.Context, client *http.Client, base *url.URL, name string, limit int64) ([]byte, int, error) {
	target := base.JoinPath("cli", name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "agentsync/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.ContentLength > limit {
		return nil, resp.StatusCode, errors.New("limit")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if int64(len(body)) > limit {
		return nil, resp.StatusCode, errors.New("limit")
	}
	return body, resp.StatusCode, nil
}

func binaryMagic(goos string, body []byte) bool {
	if len(body) < minCLIBytes {
		return false
	}
	switch goos {
	case "windows":
		return bytes.HasPrefix(body, []byte("MZ"))
	case "linux":
		return bytes.HasPrefix(body, []byte{0x7f, 'E', 'L', 'F'})
	case "darwin":
		return bytes.HasPrefix(body, []byte{0xfe, 0xed, 0xfa, 0xcf}) ||
			bytes.HasPrefix(body, []byte{0xfe, 0xed, 0xfa, 0xce}) ||
			bytes.HasPrefix(body, []byte{0xcf, 0xfa, 0xed, 0xfe}) ||
			bytes.HasPrefix(body, []byte{0xce, 0xfa, 0xed, 0xfe}) ||
			bytes.HasPrefix(body, []byte{0xca, 0xfe, 0xba, 0xbe}) ||
			bytes.HasPrefix(body, []byte{0xbe, 0xba, 0xfe, 0xca})
	default:
		return false
	}
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func replaceExecutable(dest string, body []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".agentsync-new-*")
	if err != nil {
		return errUpdateReplace
	}
	tmpName := tmp.Name()
	done := false
	defer func() {
		if !done {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return errUpdateReplace
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return errUpdateReplace
	}
	if err := tmp.Close(); err != nil {
		return errUpdateReplace
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return errUpdateReplace
	}
	if runtime.GOOS == "windows" {
		if err := replaceRunningWindows(dest, tmpName); err != nil {
			return err
		}
		done = true
		return nil
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return errUpdateReplace
	}
	done = true
	return nil
}

// Windows не перезаписывает запущенный exe. Сначала уносим его в .old, затем ставим новый файл на место.
func replaceRunningWindows(dest, tmpName string) error {
	old := dest + ".old"
	_ = os.Remove(old)
	if _, err := os.Stat(dest); err == nil {
		if err := os.Rename(dest, old); err != nil {
			return errUpdateReplace
		}
	}
	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Rename(old, dest)
		return errUpdateReplace
	}
	_ = os.Remove(old)
	return nil
}

func currentExecutable() (string, error) {
	path, err := executablePath()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path, nil
	}
	return resolved, nil
}

func inInstallDir(exe string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	want := filepath.Clean(filepath.Join(home, ".agentsync"))
	got := filepath.Clean(filepath.Dir(exe))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(got, want)
	}
	return got == want
}

func autoUpdateOff() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AGENTSYNC_AUTO_UPDATE"))) {
	case "0", "off", "false", "no":
		return true
	default:
		return false
	}
}

func updateDue(exe string, now time.Time) bool {
	raw, err := os.ReadFile(checkPath(exe))
	if err != nil {
		return true
	}
	sec, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return true
	}
	return !now.Before(time.Unix(sec, 0))
}

func stampUpdate(exe string, until time.Time) {
	_ = os.WriteFile(checkPath(exe), []byte(strconv.FormatInt(until.Unix(), 10)), 0o644)
}

func checkPath(exe string) string {
	return filepath.Join(filepath.Dir(exe), "update-check")
}

func acquireUpdateLock(dir string) (func(), bool) {
	path := filepath.Join(dir, "update.lock")
	if unlock, ok := createLock(path); ok {
		return unlock, true
	}
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) < updateLockTTL {
		return nil, false
	}
	_ = os.Remove(path)
	return createLock(path)
}

func createLock(path string) (func(), bool) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, false
	}
	_ = file.Close()
	return func() { _ = os.Remove(path) }, true
}

func shouldAutoUpdate(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "help", "version", "update", "-h", "--help":
		return false
	}
	for _, arg := range args {
		if isHelpFlag(arg) {
			return false
		}
	}
	return true
}
