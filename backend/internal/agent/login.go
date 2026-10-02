package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

func Login(ctx context.Context, server, email, password string) (string, error) {
	body, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(server, "/")+"/session",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return "", explanation(raw)
	}
	var view struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &view); err != nil {
		return "", err
	}
	var id string
	for _, cookie := range res.Cookies() {
		if cookie.Name == "session" && strings.TrimSpace(cookie.Value) != "" {
			id = cookie.Value
		}
	}
	if id == "" {
		return "", errors.New("сервер не открыл сессию")
	}
	if err := SaveHomeFile("session", id); err != nil {
		return "", err
	}
	if view.Name != "" {
		if err := SaveHomeFile("account", view.Name); err != nil {
			return "", err
		}
	}
	return view.Name, nil
}

func SessionCookie() string {
	if cookie := strings.TrimSpace(EnvOrHome("AGENTSYNC_COOKIE", "cookie")); cookie != "" {
		return cookie
	}
	id := HomeFile("session")
	if id == "" {
		return ""
	}
	return "session=" + id
}
