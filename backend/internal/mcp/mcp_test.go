package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":4,"method":"nope"}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"revert","arguments":{"agent":"Grok"}}}`,
	}, "\n")
	var out bytes.Buffer
	err := Run(t.Context(), strings.NewReader(in), &out, Env{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(&out)
	var messages []response
	for dec.More() {
		var msg response
		if err := dec.Decode(&msg); err != nil {
			t.Fatal(err)
		}
		messages = append(messages, msg)
	}
	if len(messages) != 5 {
		t.Fatalf("responses %d", len(messages))
	}
	if !strings.Contains(string(messages[0].Result), `"name":"agentsync"`) {
		t.Fatalf("initialize %s", messages[0].Result)
	}
	tools := string(messages[1].Result)
	hasPush := strings.Contains(tools, `"push"`)
	hasApply := strings.Contains(tools, `"apply"`)
	if !hasPush || !hasApply {
		t.Fatalf("tools %s", tools)
	}
	if string(messages[2].Result) != "{}" {
		t.Fatalf("ping %s", messages[2].Result)
	}
	if messages[3].Error == nil || messages[3].Error.Code != -32601 {
		t.Fatalf("unknown %+v", messages[3].Error)
	}
	revert := string(messages[4].Result)
	emptyChain := strings.Contains(revert, "цепочка пуста")
	markedError := strings.Contains(revert, `"isError":true`)
	if !emptyChain || !markedError {
		t.Fatalf("revert %s", revert)
	}
}
