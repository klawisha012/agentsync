package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMachineConfirmSwitchesAccountAndKeepsChains(t *testing.T) {
	e := New("http://localhost:3000")
	alice := mustAccount(t, e, "alice@example.com", "secret", "Alice")
	body := map[string]string{"id": "pc-1", "host": "desk", "listener": "127.0.0.1:49152"}

	guest := postJSON(t, e, "/machine", body, nil)
	if guest.Code != http.StatusUnauthorized {
		t.Fatalf("guest confirm %d %s", guest.Code, guest.Body.String())
	}

	first := postJSON(t, e, "/machine", body, alice)
	if first.Code != http.StatusOK {
		t.Fatalf("confirm %d %s", first.Code, first.Body.String())
	}
	if strings.Contains(first.Body.String(), "secret") {
		t.Fatalf("confirm leaked the password: %s", first.Body.String())
	}
	bound := decodeMachine(t, first.Body.Bytes())
	if bound.Account != "Alice" || bound.ChainID == "" || bound.AgentToken == "" {
		t.Fatalf("binding %+v", bound)
	}

	seen := getAuth(t, e, "/agent/session", bound.AgentToken)
	if seen.Code != http.StatusOK {
		t.Fatalf("agent session %d %s", seen.Code, seen.Body.String())
	}
	if strings.Contains(seen.Body.String(), "secret") {
		t.Fatalf("agent session leaked the password: %s", seen.Body.String())
	}
	agent := decodeMachine(t, seen.Body.Bytes())
	if agent.Account != "Alice" || agent.ChainID != bound.ChainID {
		t.Fatalf("agent binding %+v", agent)
	}

	off := deleteCookie(t, e, "/machine/pc-1", alice)
	if off.Code != http.StatusNoContent {
		t.Fatalf("unbind %d %s", off.Code, off.Body.String())
	}
	gone := getAuth(t, e, "/agent/session", bound.AgentToken)
	if gone.Code != http.StatusUnauthorized {
		t.Fatalf("agent still inside %d %s", gone.Code, gone.Body.String())
	}
	if strings.Contains(getJSON(t, e, "/accounts/Alice", alice).Body.String(), bound.ChainID) {
		t.Fatal("unbound chain is still on the page")
	}

	boris := mustAccount(t, e, "boris@example.com", "other-secret", "Boris")
	second := postJSON(t, e, "/machine", body, boris)
	if second.Code != http.StatusOK {
		t.Fatalf("second confirm %d %s", second.Code, second.Body.String())
	}
	next := decodeMachine(t, second.Body.Bytes())
	if next.Account != "Boris" || next.ChainID == "" || next.ChainID == bound.ChainID {
		t.Fatalf("second binding %+v", next)
	}
	if getAuth(t, e, "/agent/session", bound.AgentToken).Code != http.StatusUnauthorized {
		t.Fatal("previous agent token still opens the machine")
	}
	borisPage := getJSON(t, e, "/accounts/Boris", boris).Body.String()
	alicePage := getJSON(t, e, "/accounts/Alice", alice).Body.String()
	publicAlice := getJSON(t, e, "/accounts/Alice", nil).Body.String()
	if !strings.Contains(borisPage, next.ChainID) || strings.Contains(borisPage, bound.ChainID) {
		t.Fatalf("boris page chains: %s", borisPage)
	}
	if strings.Contains(alicePage, bound.ChainID) || strings.Contains(alicePage, next.ChainID) {
		t.Fatalf("alice page shows a chain: %s", alicePage)
	}
	if strings.Contains(publicAlice, bound.ChainID) || strings.Contains(publicAlice, next.ChainID) {
		t.Fatalf("guest page shows a chain: %s", publicAlice)
	}

	back := postJSON(t, e, "/machine", body, alice)
	if back.Code != http.StatusOK {
		t.Fatalf("reconfirm %d %s", back.Code, back.Body.String())
	}
	restored := decodeMachine(t, back.Body.Bytes())
	if restored.Account != "Alice" || restored.ChainID != bound.ChainID {
		t.Fatalf("chain was not kept: %+v want %s", restored, bound.ChainID)
	}
	if !strings.Contains(getJSON(t, e, "/accounts/Alice", alice).Body.String(), bound.ChainID) {
		t.Fatal("restored chain is hidden from the owner")
	}
	if strings.Contains(getJSON(t, e, "/accounts/Boris", boris).Body.String(), next.ChainID) {
		t.Fatal("boris still shows the machine chain")
	}
}

func mustAccount(t *testing.T, h http.Handler, email, password, name string) []*http.Cookie {
	t.Helper()
	rec := postJSON(t, h, "/accounts", map[string]string{
		"email": email, "password": password, "name": name,
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed %s %d %s", name, rec.Code, rec.Body.String())
	}
	return readSessionCookie(t, rec)
}

type machineBody struct {
	Account    string `json:"account"`
	ChainID    string `json:"chainId"`
	AgentToken string `json:"agentToken"`
}

func decodeMachine(t *testing.T, raw []byte) machineBody {
	t.Helper()
	var body machineBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func getAuth(t *testing.T, h http.Handler, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
