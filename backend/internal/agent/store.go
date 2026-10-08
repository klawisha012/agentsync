package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type storedSnapshot struct {
	Account string `json:"account"`
	Agent   string `json:"agent"`
	Number  int    `json:"number"`
	Current bool   `json:"current"`
	Files   []File `json:"files"`
}

func remember(ctx context.Context, server, root, agentName, token, cookie string) error {
	if strings.TrimSpace(token) == "" && strings.TrimSpace(cookie) == "" {
		return nil
	}
	files, missing, err := readHome(root, agentName)
	if err != nil || missing {
		return err
	}
	files = portableOnly(files)
	if len(files) == 0 {
		return nil
	}
	_, err = recordFiles(ctx, server, root, agentName, token, cookie)
	return err
}

func Record(ctx context.Context, server, root, agentName, token, cookie string) ([]byte, error) {
	var raw []byte
	err := withRootLock(root, func() error {
		var callErr error
		raw, callErr = recordFiles(ctx, server, root, agentName, token, cookie)
		return callErr
	})
	return raw, err
}

func recordFiles(ctx context.Context, server, root, agentName, token, cookie string) ([]byte, error) {
	files, missing, err := readHome(root, agentName)
	if err != nil {
		return nil, err
	}
	if !missing {
		files = portableOnly(files)
	}
	raw, err := postAgent(ctx, server, "/agent/record", agentName, token, cookie, files, missing)
	if err != nil {
		return nil, err
	}
	var view storedSnapshot
	if err := json.Unmarshal(raw, &view); err != nil {
		return nil, err
	}
	if err := writeStoreCache(root, view); err != nil {
		return nil, err
	}
	return raw, nil
}

func ReadStore(ctx context.Context, server, root, agentName, number, token, cookie string) ([]byte, error) {
	unlock, err := lockRoot(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	target := strings.TrimRight(server, "/") + "/agent/store/" + url.PathEscape(agentName)
	if number != "" {
		target += "/" + url.PathEscape(number)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return nil, errors.New(strings.TrimSpace(string(raw)))
	}
	if number == "" {
		return raw, nil
	}
	var view storedSnapshot
	if err := json.Unmarshal(raw, &view); err != nil {
		return nil, err
	}
	if err := writeStoreCache(root, view); err != nil {
		return nil, err
	}
	return raw, nil
}

func readHome(root, agentName string) ([]File, bool, error) {
	files, err := readTree(agentHome(root, agentName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return files, false, nil
}

func postAgent(ctx context.Context, server, path, agentName, token, cookie string, files []File, missing bool) ([]byte, error) {
	body, err := json.Marshal(map[string]any{"agent": agentName, "missing": missing, "files": files})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(server, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusCreated {
		return nil, explanation(raw)
	}
	return raw, nil
}

func explanation(raw []byte) error {
	var body struct {
		Explanation string `json:"explanation"`
	}
	if err := json.Unmarshal(raw, &body); err == nil && strings.TrimSpace(body.Explanation) != "" {
		return errors.New(body.Explanation)
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return errors.New("сервер не ответил")
	}
	return errors.New(text)
}

func writeStoreCache(root string, view storedSnapshot) error {
	if view.Account == "" || view.Agent == "" || view.Number < 1 {
		return errors.New("сервер не вернул снимок")
	}
	dir := filepath.Join(root, ".agentsync", "store", sanitize(view.Account), sanitize(view.Agent), fmtSeq(view.Number))
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if len(view.Files) == 0 {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	for _, file := range view.Files {
		if err := writeFile(dir, file); err != nil {
			return err
		}
	}
	if !view.Current {
		return nil
	}
	mark := filepath.Join(root, ".agentsync", "store", sanitize(view.Account), sanitize(view.Agent), "current")
	return os.WriteFile(mark, []byte(strconv.Itoa(view.Number)+"\n"), 0o644)
}
