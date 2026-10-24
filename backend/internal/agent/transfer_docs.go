package agent

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// docMove is one own subagent or command file of an AI agent on its way into agents.
type docMove struct {
	kind docKind
	file string // the agent's file
	rel  string // the agents file, relative to ~/.agents
	body string
}

// ownDocs lists the agent's own subagent and command files in the agents format. A file agents
// has a file of the same name for stays, and a file that is not the agent's format goes into the report.
func ownDocs(root string, receiver Receiver, dir string, report *TransferReport) ([]docMove, error) {
	_, ledger, err := loadFileLedger(receiver.Slug)
	if err != nil {
		return nil, err
	}
	d := dialects[receiver.Slug]
	var moves []docMove
	for _, item := range []struct {
		kind  docKind
		place string
	}{{subagentDoc, receiver.Agents}, {commandDoc, receiver.Commands}} {
		if receiver.Way(item.kind.key) != WayTranslate {
			continue
		}
		place := filepath.Join(filepath.Dir(dir), filepath.FromSlash(item.place))
		entries, err := os.ReadDir(place)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			file := filepath.Join(place, entry.Name())
			// A link is agents itself; a folder and a written file are not the agent's own files.
			if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() || slices.Contains(ledger.Own, file) {
				continue
			}
			stem, ok := docStem(item.kind, d, entry.Name())
			if !ok {
				continue
			}
			raw, err := os.ReadFile(file)
			if err != nil {
				return nil, err
			}
			if msg := SecretExplanation(file, string(raw)); msg != "" {
				return nil, errors.New(msg)
			}
			rel := item.kind.key + "/" + stem + ".md"
			if _, err := os.Lstat(filepath.Join(canonDir(root), filepath.FromSlash(rel))); err == nil {
				report.Kept = append(report.Kept, item.kind.one+" "+stem+" (имя занято в agents)")
				continue
			}
			body, ok := fromNativeDoc(item.kind, d, string(raw))
			if !ok {
				report.Kept = append(report.Kept, item.kind.one+" "+stem)
				continue
			}
			moves = append(moves, docMove{kind: item.kind, file: file, rel: rel, body: body})
		}
	}
	return moves, nil
}

// docStem is the element name of an agent's file, or false for a file that is not one.
func docStem(kind docKind, d dialect, name string) (string, bool) {
	suffix := d.agentFile
	if kind.key == commandDoc.key {
		suffix = d.commandFile
	}
	stem, ok := strings.CutSuffix(name, suffix)
	return stem, ok && stem != "" && !PrivatePath(name)
}

// fromNativeDoc turns an agent's own subagent or command file into the agents format. Fields and
// tools the core does not hold go into the agent's extension section, so the agent gets them back.
func fromNativeDoc(kind docKind, d dialect, text string) (string, bool) {
	var core, ext []docField
	body := text
	if strings.HasSuffix(d.agentFile, ".toml") && kind.key == subagentDoc.key || kind.key == commandDoc.key && d.commandFile == ".toml" {
		values, ok := tomlDoc(text)
		if !ok {
			return "", false
		}
		for _, key := range sortedKeys(values) {
			value := values[key]
			switch {
			case key == "developer_instructions" || key == "prompt":
				body = strings.ReplaceAll(value, "{{args}}", "$ARGUMENTS")
			case key == "sandbox_mode" && value == "read-only":
				core = append(core, docField{"tools", wordList(readOnlyTools)})
			case slices.Contains(kind.core, key):
				core = append(core, docField{key, &yaml.Node{Kind: yaml.ScalarNode, Value: value}})
			default:
				ext = append(ext, docField{key, &yaml.Node{Kind: yaml.ScalarNode, Value: value}})
			}
		}
	} else if front, rest, ok := splitFrontmatter(text); ok {
		body = rest
		var root yaml.Node
		if yaml.Unmarshal([]byte(front), &root) != nil || len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
			return "", false
		}
		mapping := root.Content[0]
		for i := 0; i+1 < len(mapping.Content); i += 2 {
			key, value := mapping.Content[i].Value, mapping.Content[i+1]
			switch {
			case key == "tools" && kind.key == subagentDoc.key:
				var words []string
				whole := true
				for _, name := range nodeList(value) {
					word, ok := d.agentToolWord(name)
					if !ok {
						whole = false
						continue
					}
					if !slices.Contains(words, word) {
						words = append(words, word)
					}
				}
				// The core keeps what the vocabulary says; an empty list still limits the others.
				core = append(core, docField{"tools", wordList(words)})
				if !whole {
					ext = append(ext, docField{key, value})
				}
			case key == "readonly" && d.agentTools == "readonly":
				if value.Value == "true" {
					core = append(core, docField{"tools", wordList(readOnlyTools)})
				}
			case slices.Contains(kind.core, key):
				core = append(core, docField{key, value})
			default:
				ext = append(ext, docField{key, value})
			}
		}
	}
	if len(ext) > 0 {
		section := &yaml.Node{Kind: yaml.MappingNode}
		for _, field := range ext {
			section.Content = append(section.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: field.key}, field.value)
		}
		core = append(core, docField{d.slug, section})
	}
	return markdownDoc(core, nil, body), true
}

func wordList(words []string) *yaml.Node {
	list := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
	for _, word := range words {
		list.Content = append(list.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: word})
	}
	return list
}

// agentToolWord gives the word of a native subagent tool name.
func (d dialect) agentToolWord(native string) (string, bool) {
	if d.subagentTools == nil {
		return d.agentsTool(native)
	}
	for _, word := range toolWords {
		if slices.Contains(d.subagentTools[word], native) {
			return word, true
		}
	}
	return "", false
}

// tomlDoc reads the top-level keys of a small TOML file: strings, booleans and numbers, a
// multi-line string included. ok is false for tables and anything else.
func tomlDoc(text string) (map[string]string, bool) {
	values := map[string]string{}
	lines := splitLines(strings.ReplaceAll(text, "\r\n", "\n"))
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return nil, false
		}
		key, raw = strings.TrimSpace(key), strings.TrimSpace(raw)
		if quote := raw[:min(3, len(raw))]; quote == "'''" || quote == `"""` {
			text := strings.TrimPrefix(strings.TrimPrefix(raw, quote), "\n")
			for !strings.Contains(text, quote) && i+1 < len(lines) {
				i++
				text += "\n" + lines[i]
			}
			value, _, ok := strings.Cut(text, quote)
			if !ok {
				return nil, false
			}
			values[key] = strings.TrimPrefix(value, "\n")
			continue
		}
		value, ok := tomlValue(raw)
		if !ok {
			if raw == "" || strings.ContainsAny(raw[:1], "[{") {
				return nil, false
			}
			value = raw // a number stays as written
		}
		switch v := value.(type) {
		case string:
			values[key] = v
		case bool:
			values[key] = map[bool]string{true: "true", false: "false"}[v]
		default:
			return nil, false
		}
	}
	return values, true
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
