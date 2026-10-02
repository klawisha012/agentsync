package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/klawisha012/agentsync/internal/agent"
)

const protocolVersion = "2025-06-18"

type Env struct {
	Root    string
	Server  string
	Token   string
	Account string
	Cookie  string
}

type request struct {
	ID     *json.RawMessage `json:"id"`
	Method string           `json:"method"`
	Params json.RawMessage  `json:"params"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *callError      `json:"error,omitempty"`
}

type callError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type toolArgs struct {
	Agent   string   `json:"agent"`
	Author  string   `json:"author"`
	Number  any      `json:"number"`
	Version int      `json:"version"`
	Skills  []string `json:"skills"`
	Into    []string `json:"into"`
	Place   string   `json:"place"`
}

func Run(ctx context.Context, in io.Reader, out io.Writer, env Env) error {
	dec := json.NewDecoder(in)
	enc := json.NewEncoder(out)
	for {
		if ctx.Err() != nil {
			return nil
		}
		var req request
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read mcp message: %w", err)
		}
		if req.ID == nil || string(*req.ID) == "null" {
			continue
		}
		if err := enc.Encode(dispatch(ctx, req, env)); err != nil {
			return fmt.Errorf("write mcp message: %w", err)
		}
	}
}

func dispatch(ctx context.Context, req request, env Env) response {
	switch req.Method {
	case "initialize":
		result, err := initialize(req.Params)
		if err != nil {
			return fail(req, -32602, "invalid params")
		}
		return ok(req, result)
	case "ping":
		return ok(req, map[string]any{})
	case "tools/list":
		return ok(req, map[string]any{"tools": toolList()})
	case "tools/call":
		return ok(req, callTool(ctx, req.Params, env))
	default:
		return fail(req, -32601, "method not found")
	}
}

func initialize(raw json.RawMessage) (map[string]any, error) {
	var params struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
	}
	version := params.ProtocolVersion
	if version == "" {
		version = protocolVersion
	}
	return map[string]any{
		"protocolVersion": version,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "agentsync",
			"version": "dev",
		},
	}, nil
}

func toolList() []map[string]any {
	agentSchema := objectSchema([]string{"agent"})
	return []map[string]any{
		tool("push", "Записывает переносимую настройку ИИ-агента в\u00a0публикацию.", agentSchema),
		tool("record", "Записывает снимок ИИ-агента в\u00a0хранилище.", agentSchema),
		tool("store", "Читает снимок ИИ-агента из\u00a0хранилища.", objectSchema([]string{"agent"})),
		tool("revert", "Возвращает последний снимок на\u00a0этой машине.", agentSchema),
		tool("apply", "Применяет публикацию автора к\u00a0ИИ-агенту.", objectSchema([]string{"author", "agent"})),
		tool("skills", "Кладёт выбранные навыки публикации в\u00a0каталоги приёмников.", skillsSchema()),
	}
}

func tool(name, description string, schema map[string]any) map[string]any {
	return map[string]any{
		"name":        name,
		"description": description,
		"inputSchema": schema,
	}
}

func skillsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"author":  map[string]any{"type": "string"},
			"agent":   map[string]any{"type": "string"},
			"version": map[string]any{"type": "integer"},
			"skills": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
			"into": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
			"place": map[string]any{"type": "string", "enum": []string{"global", "project"}},
		},
		"required": []string{"author", "agent", "version", "skills", "into", "place"},
	}
}

func objectSchema(required []string) map[string]any {
	properties := map[string]any{
		"agent":  map[string]any{"type": "string"},
		"author": map[string]any{"type": "string"},
		"number": map[string]any{"type": "string"},
	}
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func callTool(ctx context.Context, raw json.RawMessage, env Env) map[string]any {
	var call toolCall
	if err := json.Unmarshal(raw, &call); err != nil {
		return toolError("invalid params")
	}
	var args toolArgs
	if len(call.Arguments) > 0 && string(call.Arguments) != "null" {
		if err := json.Unmarshal(call.Arguments, &args); err != nil {
			return toolError("invalid params")
		}
	}
	text, err := runTool(ctx, call.Name, args, env)
	if err != nil {
		return toolError(err.Error())
	}
	return toolText(text, false)
}

func runTool(ctx context.Context, name string, args toolArgs, env Env) (string, error) {
	switch name {
	case "push":
		if args.Agent == "" {
			return "", errors.New("назовите ИИ-агента")
		}
		err := agent.Push(
			ctx,
			env.Server,
			env.Root,
			args.Agent,
			env.Token,
			env.Cookie,
		)
		return done(err)
	case "record":
		if args.Agent == "" {
			return "", errors.New("назовите ИИ-агента")
		}
		raw, err := agent.Record(
			ctx,
			env.Server,
			env.Root,
			args.Agent,
			env.Token,
			env.Cookie,
		)
		return string(raw), err
	case "store":
		if args.Agent == "" {
			return "", errors.New("назовите ИИ-агента")
		}
		raw, err := agent.ReadStore(
			ctx,
			env.Server,
			env.Root,
			args.Agent,
			numberText(args.Number),
			env.Token,
			env.Cookie,
		)
		return string(raw), err
	case "revert":
		if args.Agent == "" {
			return "", errors.New("назовите ИИ-агента")
		}
		err := agent.Revert(env.Root, env.Account, args.Agent)
		return done(err)
	case "skills":
		err := agent.CopySkills(ctx, env.Server, env.Root, env.Account, env.Token, env.Cookie, agent.SkillCopy{
			Author:  args.Author,
			Source:  args.Agent,
			Version: args.Version,
			Skills:  args.Skills,
			Targets: args.Into,
			Place:   args.Place,
		})
		return done(err)
	case "apply":
		missingName := args.Author == "" || args.Agent == ""
		if missingName {
			return "", errors.New("назовите автора и ИИ-агента")
		}
		err := agent.Apply(
			ctx,
			env.Server,
			env.Root,
			args.Author,
			args.Agent,
			env.Account,
			env.Token,
			env.Cookie,
		)
		return done(err)
	default:
		return "", errors.New("неизвестный инструмент")
	}
}

func done(err error) (string, error) {
	if err != nil {
		return "", err
	}
	return "Готово.", nil
}

func numberText(value any) string {
	switch n := value.(type) {
	case nil:
		return ""
	case string:
		return n
	case float64:
		return strconv.FormatInt(int64(n), 10)
	default:
		return fmt.Sprint(n)
	}
}

func toolText(text string, isError bool) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": isError,
	}
}

func toolError(text string) map[string]any {
	return toolText(text, true)
}

func ok(req request, result any) response {
	raw, err := json.Marshal(result)
	if err != nil {
		return fail(req, -32603, "internal error")
	}
	return response{JSONRPC: "2.0", ID: *req.ID, Result: raw}
}

func fail(req request, code int, message string) response {
	return response{
		JSONRPC: "2.0",
		ID:      *req.ID,
		Error:   &callError{Code: code, Message: message},
	}
}
