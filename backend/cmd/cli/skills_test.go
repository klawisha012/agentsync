package main

import (
	"strings"
	"testing"
)

func TestParseSkillArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		version int
		skills  string
		targets string
		place   string
		err     string
	}{
		{
			name:    "flags",
			args:    []string{"Author", "Grok", "--version", "4", "--global", "--skill", "ru-text", "--into", "cursor"},
			version: 4,
			skills:  "ru-text",
			targets: "cursor",
			place:   "global",
		},
		{
			name:    "equals form",
			args:    []string{"--project", "--skill=graphify", "Author", "Grok", "--into=windsurf", "--version=2", "--skill", "ru-text"},
			version: 2,
			skills:  "graphify,ru-text",
			targets: "windsurf",
			place:   "project",
		},
		{
			name: "missing version",
			args: []string{"Author", "Grok", "--global", "--skill", "ru-text", "--into", "cursor"},
			err:  "номер версии",
		},
		{
			name: "bad version",
			args: []string{"Author", "Grok", "--version", "0", "--global", "--skill", "ru-text", "--into", "cursor"},
			err:  "номер версии",
		},
		{
			name: "unknown flag",
			args: []string{"Author", "Grok", "--fast"},
			err:  "--fast",
		},
		{
			name:    "command without place",
			args:    []string{"Author", "Grok", "--version", "4", "ru-text"},
			version: 4,
			skills:  "ru-text",
		},
		{
			name: "both places",
			args: []string{"Author", "Grok", "--version", "4", "--global", "--project", "--into", "cursor", "ru-text"},
			err:  "одно место",
		},
		{
			name:    "skill list",
			args:    []string{"Author", "Grok", "--version", "4", "--global", "--into", "cursor", "--into", "windsurf", "ru-text", "graphify"},
			version: 4,
			skills:  "ru-text,graphify",
			targets: "cursor,windsurf",
			place:   "global",
		},
		{
			name:    "skill before flags",
			args:    []string{"Author", "Grok", "ru-text", "--version", "1", "--project", "--skill", "graphify", "--into", "cursor"},
			version: 1,
			skills:  "ru-text,graphify",
			targets: "cursor",
			place:   "project",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSkillArgs(tt.args)
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("error %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.author != "Author" || got.source != "Grok" || got.version != tt.version || got.place != tt.place {
				t.Fatalf("call %+v", got)
			}
			if strings.Join(got.skills, ",") != tt.skills || strings.Join(got.targets, ",") != tt.targets {
				t.Fatalf("lists %+v", got)
			}
		})
	}
}
