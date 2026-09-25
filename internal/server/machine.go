package server

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type machine struct {
	id       string
	host     string
	listener string
	account  *account
	token    string
}

type machineView struct {
	ID         string `json:"id"`
	Host       string `json:"host"`
	Listener   string `json:"listener"`
	Account    string `json:"account"`
	ChainID    string `json:"chainId"`
	AgentToken string `json:"agentToken,omitempty"`
}

type machinePage struct {
	ID       string `json:"id"`
	Host     string `json:"host"`
	Listener string `json:"listener"`
	ChainID  string `json:"chainId"`
}

func (s *store) confirmMachine(sessionID, id, host, listener string) (machineView, int, string) {
	id = strings.TrimSpace(id)
	host = strings.TrimSpace(host)
	listener = strings.TrimSpace(listener)
	if id == "" || host == "" {
		return machineView{}, http.StatusBadRequest, "Укажите компьютер."
	}
	token, err := newSessionID()
	if err != nil {
		return machineView{}, http.StatusInternalServerError, "Не удалось подтвердить машину."
	}
	freshChain, err := newSessionID()
	if err != nil {
		return machineView{}, http.StatusInternalServerError, "Не удалось подтвердить машину."
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.sessions[sessionID]
	if item == nil {
		return machineView{}, http.StatusUnauthorized, "Войдите в аккаунт."
	}
	if item.chains == nil {
		item.chains = map[string]string{}
	}
	chainID, kept := item.chains[id]
	if !kept {
		chainID = freshChain
		item.chains[id] = chainID
	}
	if prev := s.machines[id]; prev != nil {
		delete(s.agents, prev.token)
	}
	bound := &machine{
		id:       id,
		host:     host,
		listener: listener,
		account:  item,
		token:    token,
	}
	s.machines[id] = bound
	s.agents[token] = bound
	return machineView{
		ID:         id,
		Host:       host,
		Listener:   listener,
		Account:    item.name,
		ChainID:    chainID,
		AgentToken: token,
	}, http.StatusOK, ""
}

func (s *store) releaseMachine(sessionID, id string) (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.sessions[sessionID]
	if item == nil {
		return http.StatusUnauthorized, "Войдите в аккаунт."
	}
	bound := s.machines[id]
	if bound == nil || bound.account != item {
		return http.StatusNotFound, "Эта машина не подтверждена в аккаунте."
	}
	delete(s.agents, bound.token)
	delete(s.machines, id)
	return http.StatusNoContent, ""
}

func (s *store) agentSession(token string) (machineView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bound := s.agents[token]
	if bound == nil || bound.account == nil {
		return machineView{}, false
	}
	return machineView{
		ID:       bound.id,
		Host:     bound.host,
		Listener: bound.listener,
		Account:  bound.account.name,
		ChainID:  bound.account.chains[bound.id],
	}, true
}

// machinesOf reads s.machines. The caller holds s.mu.
func (s *store) machinesOf(item *account) []machinePage {
	pages := make([]machinePage, 0)
	for _, bound := range s.machines {
		if bound.account != item {
			continue
		}
		pages = append(pages, machinePage{
			ID:       bound.id,
			Host:     bound.host,
			Listener: bound.listener,
			ChainID:  item.chains[bound.id],
		})
	}
	if len(pages) == 0 {
		return nil
	}
	return pages
}

func (a *app) confirmMachine(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	var req struct {
		ID       string `json:"id"`
		Host     string `json:"host"`
		Listener string `json:"listener"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Укажите компьютер.")
	}
	view, code, msg := a.accounts.confirmMachine(cookie.Value, req.ID, req.Host, req.Listener)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, view)
}

func (a *app) releaseMachine(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	code, msg := a.accounts.releaseMachine(cookie.Value, c.Param("id"))
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.NoContent(code)
}

func (a *app) agentSession(c echo.Context) error {
	token := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
	token = strings.TrimSpace(token)
	view, ok := a.accounts.agentSession(token)
	if !ok {
		return writeExplanation(c, http.StatusUnauthorized, "Локальный агент не вошёл в аккаунт.")
	}
	return c.JSON(http.StatusOK, view)
}
