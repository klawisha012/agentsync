package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// ApplyElements puts the named paths of one publication version into agents. A skill folder
// anywhere becomes skills/<name>; subagent, command and instructions files and MCP, plugin and
// hook records go to their agents places. Same-name elements are replaced whole, the rest stays.
// The AI agent's own files are not written: skipped names them. Connected AI agents reconnect.
func ApplyElements(ctx context.Context, server, root, author, agentName, account, token, cookie string, version int, paths []string) ([]string, error) {
	unlock, err := lockRoot(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if version < 1 {
		return nil, errors.New("Назовите номер версии.")
	}
	if len(paths) == 0 {
		return nil, errors.New("Назовите элемент.")
	}
	payload, err := fetchVersion(ctx, server, author, agentName, token, cookie, version)
	if err != nil {
		return nil, err
	}
	if err := pathsFit(payload.Files); err != nil {
		return nil, err
	}
	// A stored path may hold "..": "agents/../x" would name the whole agents as an element.
	for i := range payload.Files {
		payload.Files[i].Path, _ = CleanRel(payload.Files[i].Path)
	}
	split, skipped, err := chosenElements(root, payload, paths)
	if err != nil {
		return skipped, err
	}
	owner := strings.TrimSpace(account)
	if owner == "" {
		owner = payload.Author
	}
	snapshot, err := snapshotTree(root, owner, canonCard, canonDir(root))
	if err != nil {
		return skipped, err
	}
	var undo applyUndo
	if err := writeCard(root, "", split, &undo, false); err != nil {
		undoErr := undo.rollback()
		// The snapshot holds the previous canon until it is back in place.
		if undoErr == nil {
			undoErr = os.RemoveAll(snapshot)
		}
		return skipped, errors.Join(err, undoErr)
	}
	if err := undo.finish(canonDir(root)); err != nil {
		return skipped, err
	}
	return skipped, reconnect(root, owner)
}

// chosenElements takes the named paths of a card apart into agents elements and records.
// A path is a file or a folder of the card; a file that gives agents nothing is skipped.
func chosenElements(root string, payload publicationPayload, paths []string) (cardSplit, []string, error) {
	split := cardSplit{canon: map[string][]File{}, records: map[recordKind]map[string]json.RawMessage{}}
	for _, kind := range recordKinds {
		split.records[kind] = map[string]json.RawMessage{}
	}
	folders := skillFolders(payload.Files)
	var skills []skillFolder
	var files []File
	seen := map[string]bool{}
	for _, raw := range paths {
		want, ok := CleanRel(raw)
		if !ok {
			return split, nil, fmt.Errorf("В этой версии нет «%s».", raw)
		}
		if i := slices.IndexFunc(folders, func(f skillFolder) bool { return f.path == want }); i >= 0 {
			skills = append(skills, folders[i])
			continue
		}
		found := false
		for _, file := range payload.Files {
			if file.Path == want || strings.HasPrefix(file.Path, want+"/") {
				found = true
				if !seen[file.Path] {
					seen[file.Path] = true
					files = append(files, file)
				}
			}
		}
		if !found {
			return split, nil, fmt.Errorf("В этой версии нет «%s».", raw)
		}
	}
	// A file inside a skill folder brings the whole folder: the apply replaces a skill whole.
	for _, file := range files {
		i := slices.IndexFunc(folders, func(f skillFolder) bool { return strings.HasPrefix(file.Path, f.path+"/") })
		if i >= 0 && !slices.ContainsFunc(skills, func(f skillFolder) bool { return f.path == folders[i].path }) {
			skills = append(skills, folders[i])
		}
	}
	skills, err := uniqueSkillBases(skills)
	if err != nil {
		return split, nil, err
	}
	for _, folder := range skills {
		element := "skills/" + path.Base(folder.path)
		for _, file := range folder.files {
			split.canon[element] = append(split.canon[element], File{Path: element + strings.TrimPrefix(file.Path, folder.path), Body: file.Body})
		}
	}
	var mark []File
	if i := slices.IndexFunc(payload.Files, func(f File) bool { return f.Path == MarkFile }); i >= 0 {
		mark = payload.Files[i : i+1]
	}
	places, connectable := canonPlaces, true
	if !isCanonCard(payload.Agent) {
		places, connectable = agentPlaces(root, agentHome(root, payload.Agent), payload.Agent)
	}
	var skipped []string
	for _, file := range files {
		if file.Path == MarkFile || slices.ContainsFunc(skills, func(f skillFolder) bool { return strings.HasPrefix(file.Path, f.path+"/") }) {
			continue
		}
		var part cardSplit
		if isCanonCard(payload.Agent) && file.Path == canonRecordFile[hookRecords] {
			records, err := canonHookRecords(file.Body)
			if err != nil {
				return split, nil, err
			}
			part.records = map[recordKind]map[string]json.RawMessage{hookRecords: records}
		} else {
			part = splitCard(append(slices.Clone(mark), file), places, connectable)
		}
		given := false
		for element, list := range part.canon {
			split.canon[element] = append(split.canon[element], list...)
			given = true
		}
		for kind, records := range part.records {
			maps.Copy(split.records[kind], records)
			given = given || len(records) > 0
		}
		if !given {
			skipped = append(skipped, file.Path)
		}
	}
	empty := len(split.canon) == 0
	for _, records := range split.records {
		empty = empty && len(records) == 0
	}
	if empty {
		return split, skipped, errors.New("В выбранном нет элементов agents.")
	}
	return split, skipped, nil
}

// canonPlaces are the places of the agents card itself: its paths are already agents paths.
var canonPlaces = cardPlaces{skills: "skills", agents: "agents", commands: "commands", instructions: "AGENTS.md", hasMCP: true, mcp: "mcp.json", plugins: "plugins.json"}

// canonHookRecords lists the records of a hooks.json in the agents format by their canon names.
func canonHookRecords(body string) (map[string]json.RawMessage, error) {
	hooks, err := parseCanonHooks([]byte(body))
	if err != nil {
		return nil, err
	}
	records := hooks.core.records()
	for slug, section := range hooks.ext {
		for name, group := range section.records() {
			records["@"+slug+" "+name] = group
		}
	}
	return records, nil
}

type skillFolder struct {
	path  string
	files []File
}

func uniqueSkillBases(folders []skillFolder) ([]skillFolder, error) {
	grouped := map[string][]string{}
	order := make([]string, 0)
	for _, folder := range folders {
		base := path.Base(folder.path)
		if _, ok := grouped[base]; !ok {
			order = append(order, base)
		}
		grouped[base] = append(grouped[base], folder.path)
	}
	var clashes []string
	for _, base := range order {
		paths := grouped[base]
		if len(paths) < 2 {
			continue
		}
		quoted := make([]string, len(paths))
		for i, item := range paths {
			quoted[i] = "«" + item + "»"
		}
		clashes = append(clashes, strings.Join(quoted, " и "))
	}
	if msg := sameFolderNameMessage(clashes); msg != "" {
		return nil, errors.New(msg)
	}
	return folders, nil
}

func sameFolderNameMessage(groups []string) string {
	if len(groups) == 0 {
		return ""
	}
	return "Навыки " + strings.Join(groups, ", ") + " называются одинаково. Оставьте один."
}

func missingDirs(start string) []string {
	var missing []string
	dir := filepath.Clean(start)
	for {
		if _, err := os.Stat(dir); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			break
		}
		missing = append(missing, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return missing
}

func skillFolders(all []File) []skillFolder {
	dirs := map[string]struct{}{}
	cleaned := make([]File, 0, len(all))
	for _, file := range all {
		rel, ok := CleanRel(file.Path)
		if !ok {
			continue
		}
		file.Path = rel
		cleaned = append(cleaned, file)
		if strings.EqualFold(path.Base(rel), "SKILL.md") {
			dir := path.Dir(rel)
			if dir != "." {
				dirs[dir] = struct{}{}
			}
		}
	}
	folders := make([]skillFolder, 0, len(dirs))
	for dir := range dirs {
		var files []File
		prefix := dir + "/"
		for _, file := range cleaned {
			if file.Path == dir || strings.HasPrefix(file.Path, prefix) {
				files = append(files, file)
			}
		}
		folders = append(folders, skillFolder{path: dir, files: files})
	}
	slices.SortFunc(folders, func(a, b skillFolder) int {
		return strings.Compare(a.path, b.path)
	})
	return folders
}

// CleanRel keeps a publication path inside one directory on every OS.
func CleanRel(value string) (string, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || strings.Contains(value, "\x00") || strings.ContainsAny(value, ":$`'\"") {
		return "", false
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") || strings.ContainsAny(cleaned, ":$`'\"") {
		return "", false
	}
	return cleaned, true
}
