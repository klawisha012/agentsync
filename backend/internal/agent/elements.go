package agent

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// docKind is a kind of agents elements kept as one Markdown file with frontmatter.
type docKind struct {
	key  string // the kind in the catalog: "agents" or "commands"
	name string // the kind in a report
	core []string
}

var (
	subagentDoc = docKind{key: "agents", name: "субагенты", core: []string{"name", "description", "tools"}}
	commandDoc  = docKind{key: "commands", name: "пользовательские команды", core: []string{"description", "argument-hint"}}
	// readOnlyTools are the tool words that never change files or run commands.
	readOnlyTools = []string{"read", "glob", "grep", "list", "web_fetch", "web_search"}
	numberedArg   = regexp.MustCompile(`\$[1-9]`)
)

type docField struct {
	key   string
	value *yaml.Node
}

// canonDoc is a subagent or command file in the agents format: core fields, extension sections
// under an AI agent's catalog name, and the body. tools is nil when the file allows every tool.
type canonDoc struct {
	fields []docField
	ext    map[string][]docField
	tools  []string
	body   string
}

// parseCanonDoc reads a file of agents. A field outside the core and the extension sections, or a
// tool outside the vocabulary, is a refusal: the file is still in an AI agent's own format.
func parseCanonDoc(kind docKind, rel, text string) (canonDoc, error) {
	doc := canonDoc{ext: map[string][]docField{}, body: text}
	front, body, ok := splitFrontmatter(text)
	if !ok {
		return doc, nil
	}
	doc.body = body
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(front), &root); err != nil {
		return doc, fmt.Errorf("Frontmatter файла ~/.agents/%s не разобран.", rel)
	}
	if len(root.Content) == 0 {
		return doc, nil
	}
	mapping := root.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return doc, fmt.Errorf("Frontmatter файла ~/.agents/%s не разобран.", rel)
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i].Value, mapping.Content[i+1]
		if slices.Contains(kind.core, key) {
			doc.fields = append(doc.fields, docField{key, value})
			continue
		}
		receiver, found := FindReceiver(key)
		if !found || value.Kind != yaml.MappingNode {
			return doc, fmt.Errorf("В ~/.agents/%s поле «%s» не из ядра agents. Перенесите его в раздел своего ИИ-агента, например claude:.", rel, key)
		}
		for j := 0; j+1 < len(value.Content); j += 2 {
			doc.ext[receiver.Slug] = append(doc.ext[receiver.Slug], docField{value.Content[j].Value, value.Content[j+1]})
		}
	}
	if tools, ok := doc.field("tools"); ok {
		names := nodeList(tools)
		if !(len(names) == 1 && names[0] == "*") {
			doc.tools = []string{}
			for _, name := range names {
				if !knownTool(name) {
					return doc, fmt.Errorf("В ~/.agents/%s инструмент «%s» не из словаря agents.", rel, name)
				}
				doc.tools = append(doc.tools, name)
			}
		}
	}
	return doc, nil
}

func splitFrontmatter(text string) (string, string, bool) {
	normal := strings.ReplaceAll(text, "\r\n", "\n")
	rest, ok := strings.CutPrefix(normal, "---\n")
	if !ok {
		return "", text, false
	}
	front, body, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		front, ok = strings.CutSuffix(rest, "\n---")
		body = ""
	}
	if !ok {
		return "", text, false
	}
	return front, body, true
}

// nodeList reads a YAML list or a comma-separated string.
func nodeList(node *yaml.Node) []string {
	var out []string
	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			out = append(out, strings.TrimSpace(item.Value))
		}
		return out
	}
	for _, part := range strings.Split(node.Value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (d canonDoc) field(key string) (*yaml.Node, bool) {
	for _, field := range d.fields {
		if field.key == key {
			return field.value, true
		}
	}
	return nil, false
}

func (d canonDoc) text(key string) string {
	if node, ok := d.field(key); ok {
		return node.Value
	}
	return ""
}

// nativeDoc is a file an AI agent gets from agents. link means the agent reads the agents file
// as is; skip is why it gets nothing; dropped are the tools it has no counterpart for.
type nativeDoc struct {
	name    string
	body    string
	link    bool
	skip    bool
	dropped []string
}

// translateDoc turns a canon file into the AI agent's file.
func translateDoc(kind docKind, d dialect, base string, doc canonDoc) nativeDoc {
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	ext := doc.ext[d.slug]
	if kind.key == commandDoc.key {
		if d.commandFile != ".toml" {
			return nativeDoc{name: base, link: len(ext) == 0 && len(doc.ext) == 0, body: markdownDoc(doc.fields, ext, doc.body)}
		}
		if numberedArg.MatchString(doc.body) {
			return nativeDoc{skip: true}
		}
		lines := []string{}
		if text := doc.text("description"); text != "" {
			lines = append(lines, "description = "+tomlString(text))
		}
		lines = append(lines, "prompt = "+tomlText(strings.ReplaceAll(doc.body, "$ARGUMENTS", "{{args}}")))
		return nativeDoc{name: stem + ".toml", body: strings.Join(append(lines, tomlFields(ext)...), "\n") + "\n"}
	}
	var out nativeDoc
	var native []string
	readOnly := doc.tools != nil
	for _, word := range doc.tools {
		readOnly = readOnly && slices.Contains(readOnlyTools, word)
		names, ok := d.agentTool(word)
		if !ok {
			out.dropped = append(out.dropped, word)
		}
		for _, name := range names {
			if !slices.Contains(native, name) {
				native = append(native, name)
			}
		}
	}
	switch d.agentTools {
	case "readonly":
		// Without a tool list a restricted subagent keeps its limit only as read-only.
		if doc.tools != nil && !readOnly {
			return nativeDoc{skip: true}
		}
		out.dropped = nil
	default:
		if doc.tools != nil && len(native) == 0 {
			return nativeDoc{skip: true, dropped: out.dropped}
		}
	}
	if d.agentFile == ".toml" {
		// Codex needs both fields; a file without them is named after itself.
		name, description := cmp.Or(doc.text("name"), stem), cmp.Or(doc.text("description"), doc.text("name"), stem)
		lines := []string{"name = " + tomlString(name), "description = " + tomlString(description)}
		if readOnly {
			lines = append(lines, `sandbox_mode = "read-only"`)
		}
		lines = append(lines, "developer_instructions = "+tomlText(doc.body))
		out.name, out.body = stem+".toml", strings.Join(append(lines, tomlFields(ext)...), "\n")+"\n"
		return out
	}
	fields := slices.DeleteFunc(slices.Clone(doc.fields), func(field docField) bool { return field.key == "tools" })
	switch {
	case d.agentTools == "readonly" && readOnly:
		fields = append(fields, docField{"readonly", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}})
	case d.agentTools == "string" && doc.tools != nil:
		fields = append(fields, docField{"tools", &yaml.Node{Kind: yaml.ScalarNode, Value: strings.Join(native, ", ")}})
	case doc.tools != nil && d.agentTools != "readonly":
		list := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
		for _, name := range native {
			list.Content = append(list.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: name})
		}
		fields = append(fields, docField{"tools", list})
	}
	out.name = stem + d.agentFile
	out.body = markdownDoc(fields, ext, doc.body)
	// The agents file already reads as the agent's own: no tools to rename, no extension, same name.
	out.link = doc.tools == nil && len(doc.ext) == 0 && d.agentFile == ".md"
	return out
}

// markdownDoc writes frontmatter from core fields and lifted extension fields, then the body.
func markdownDoc(fields, ext []docField, body string) string {
	all := slices.Clone(fields)
	for _, field := range ext {
		all = slices.DeleteFunc(all, func(have docField) bool { return have.key == field.key })
		all = append(all, field)
	}
	if len(all) == 0 {
		return body
	}
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	for _, field := range all {
		mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: field.key}, field.value)
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	_ = enc.Encode(mapping)
	return "---\n" + out.String() + "---\n" + body
}

// tomlText writes a multi-line literal string when the text allows it, so it reads as written.
func tomlText(text string) string {
	if strings.Contains(text, "'''") || strings.HasSuffix(text, "'") {
		return tomlString(text)
	}
	return "'''\n" + text + "'''"
}

// tomlFields writes scalar extension fields as TOML keys; a list or a table stays out.
func tomlFields(fields []docField) []string {
	var lines []string
	for _, field := range fields {
		if field.value.Kind != yaml.ScalarNode {
			continue
		}
		value := tomlString(field.value.Value)
		if tag := field.value.ShortTag(); tag == "!!bool" || tag == "!!int" || tag == "!!float" {
			value = field.value.Value
		}
		lines = append(lines, tomlKey(field.key)+" = "+value)
	}
	return lines
}

// agentTool gives the native names of a tool word in a subagent's tool list.
func (d dialect) agentTool(word string) ([]string, bool) {
	if names, ok := d.subagentTools[word]; ok {
		return names, true
	}
	if server, tool, ok := mcpName(word); ok && d.subagentMCP != "" {
		return []string{fmt.Sprintf(d.subagentMCP, server, tool)}, true
	}
	if d.subagentTools != nil {
		return nil, false
	}
	return d.nativeTool(word)
}

// canonWrite is a translated agents file an AI agent gets at dest.
type canonWrite struct {
	rel  string // the agents file, relative to ~/.agents
	dest string
	body string
}

// fileRecords is the ledger of files connect wrote into an AI agent's home.
var fileRecords = recordKind{ledger: "files"}

func loadFileLedger(slug string) (string, recordLedger, error) {
	return loadLedger(fileRecords, slug)
}

// removeWrittenFiles deletes the files connect wrote, then the ledger.
func removeWrittenFiles(slug string) error {
	file, ledger, err := loadFileLedger(slug)
	if err != nil || ledger.File == "" && len(ledger.Own) == 0 {
		return err
	}
	for _, path := range ledger.Own {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return os.Remove(file)
}

// canonFiles puts the agents file back, in an upload of home, in place of each file connect
// translated, so applying the card gives agents its own file and not a translation.
func canonFiles(root string, files []File, agentName, home string) []File {
	receiver, found := FindReceiver(agentName)
	if !found {
		return files
	}
	_, ledger, err := loadFileLedger(receiver.Slug)
	if err != nil {
		return files
	}
	for dest, origin := range ledger.Origins {
		rel, err := filepath.Rel(home, dest)
		if err != nil || !filepath.IsLocal(rel) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(canonDir(root), filepath.FromSlash(origin)))
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		for i := range files {
			if files[i].Path == rel {
				files[i] = File{Path: pathpkg.Join(pathpkg.Dir(rel), pathpkg.Base(origin)), Body: string(raw)}
			}
		}
	}
	return files
}
