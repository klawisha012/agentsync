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
		err     string
	}{
		{
			name:    "flags",
			args:    []string{"Author", "Grok", "--version", "4", "--skill", "ru-text", "--into", "Claude"},
			version: 4,
			skills:  "ru-text",
			targets: "Claude",
		},
		{
			name:    "equals form",
			args:    []string{"--skill=graphify", "Author", "Grok", "--into=Grok", "--version=2", "--skill", "ru-text"},
			version: 2,
			skills:  "graphify,ru-text",
			targets: "Grok",
		},
		{
			name: "missing version",
			args: []string{"Author", "Grok", "--skill", "ru-text", "--into", "Grok"},
			err:  "номер версии",
		},
		{
			name: "bad version",
			args: []string{"Author", "Grok", "--version", "0", "--skill", "ru-text", "--into", "Grok"},
			err:  "номер версии",
		},
		{
			name: "unknown flag",
			args: []string{"Author", "Grok", "--fast"},
			err:  "--fast",
		},
		{
			name:    "skill list",
			args:    []string{"Author", "Grok", "--version", "4", "--into", "Grok", "--into", "Claude", "ru-text", "graphify"},
			version: 4,
			skills:  "ru-text,graphify",
			targets: "Grok,Claude",
		},
		{
			name:    "skill before flags",
			args:    []string{"Author", "Grok", "ru-text", "--version", "1", "--skill", "graphify", "--into", "Grok"},
			version: 1,
			skills:  "ru-text,graphify",
			targets: "Grok",
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
			if got.author != "Author" || got.source != "Grok" || got.version != tt.version {
				t.Fatalf("call %+v", got)
			}
			if strings.Join(got.skills, ",") != tt.skills || strings.Join(got.targets, ",") != tt.targets {
				t.Fatalf("lists %+v", got)
			}
		})
	}
}
