package agent

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// MarkFile lists, in an uploaded AI agent snapshot, the MCP records on wrote. Its presence
// also tells a new publication from one made before the canon.
const MarkFile = "agentsync.json"

// canonCard is the card and store name of the canon ~/.agents.
const canonCard = "agents"

type canonMark struct {
	MCP          []string `json:"mcp"`
	Plugins      []string `json:"plugins"`
	Marketplaces []string `json:"marketplaces"`
	Hooks        []string `json:"hooks"`
}

// markLists names the mark list of each record kind.
func (m *canonMark) lists() map[recordKind]*[]string {
	return map[recordKind]*[]string{mcpRecords: &m.MCP, pluginRecords: &m.Plugins, marketRecords: &m.Marketplaces, hookRecords: &m.Hooks}
}

func isCanonCard(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == canonCard || name == ".agents"
}

// withMark adds the mark to an AI agent upload. A canon card has none: its apply is an exact copy.
func withMark(files []File, agentName, home string) []File {
	if isCanonCard(agentName) {
		return files
	}
	return append(files, markFile(home))
}

// applyCanon makes the canon an exact copy of a canon card. Private files such as .env and
// machine MCP servers stay. A break puts the canon back and drops the attempt's snapshot.
func applyCanon(root, account string, files []File, broken bool) error {
	files = slices.DeleteFunc(slices.Clone(files), func(file File) bool { return file.Path == MarkFile })
	if err := pathsFit(files); err != nil {
		return err
	}
	canon := canonDir(root)
	current, err := readTree(canon)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	before := portableOnly(current)
	snapshot, err := writeSnapshot(root, account, canonCard, before)
	if err != nil {
		return err
	}
	err = replacePortable(root, canon, files, broken)
	if err == nil {
		return nil
	}
	undoErr := replacePortable(root, canon, before, false)
	// The snapshot holds the previous canon until it is back in place.
	if undoErr == nil {
		undoErr = os.RemoveAll(snapshot)
	}
	return errors.Join(err, undoErr)
}

func replacePortable(root, dir string, files []File, broken bool) error {
	if _, err := clearPortable(root, dir); err != nil {
		return err
	}
	for _, file := range files {
		if err := writeHomeFile(dir, file); err != nil {
			return err
		}
		if broken {
			return errors.New("применение не выполнено")
		}
	}
	return nil
}

// markFile builds the mark for an upload of home: the records on wrote into a settings file inside it.
func markFile(home string) File {
	mark := canonMark{MCP: []string{}, Plugins: []string{}, Marketplaces: []string{}, Hooks: []string{}}
	if dir, err := stateDir(); err == nil {
		for kind, list := range mark.lists() {
			ledgers, _ := filepath.Glob(filepath.Join(dir, kind.ledger, "*.json"))
			for _, file := range ledgers {
				var ledger recordLedger
				if raw, err := os.ReadFile(file); err == nil && json.Unmarshal(raw, &ledger) == nil && within(home, ledger.File) {
					*list = append(*list, ledger.Own...)
				}
			}
		}
	}
	raw, _ := json.Marshal(mark)
	return File{Path: MarkFile, Body: string(raw)}
}

// cardPlaces are the canon places of an AI agent, relative to its home; "" when it has none.
type cardPlaces struct {
	skills, agents, commands, instructions, mcp, plugins, hooks string
	hasMCP                                                      bool
	hookEvents, hookTools                                       map[string]string
}

// agentPlaces finds the canon places of the card's AI agent. ok is false when the home is not
// the catalog agent's home: then every file of the card is the agent's own.
func agentPlaces(root, home, name string) (cardPlaces, bool) {
	receiver, found := FindReceiver(name)
	if !found {
		return cardPlaces{}, false
	}
	dir, have := receiver.GlobalDir(root)
	if !have || !within(home, dir) || insideCanon(root, dir) {
		return cardPlaces{}, false
	}
	rel := func(place string) string {
		out, err := filepath.Rel(home, place)
		if err != nil || !filepath.IsLocal(out) {
			return ""
		}
		return filepath.ToSlash(out)
	}
	beside := func(place string) string {
		if place == "" {
			return ""
		}
		return rel(filepath.Join(filepath.Dir(dir), filepath.FromSlash(place)))
	}
	places := cardPlaces{
		skills:       rel(dir),
		agents:       beside(receiver.Agents),
		commands:     beside(receiver.Commands),
		instructions: beside(receiver.Instructions),
		hasMCP:       receiver.MCP != "",
	}
	if places.hasMCP {
		places.mcp = rel(receiver.mcpFile(root))
	}
	places.plugins = beside(receiver.Plugins)
	places.hooks = beside(receiver.Hooks)
	places.hookEvents, places.hookTools = receiver.HookEvents, receiver.HookTools
	return places, true
}

// cardSplit is a card taken apart: canon elements with canon paths, MCP records for the
// canon, and the agent's own files.
type cardSplit struct {
	canon   map[string][]File
	records map[recordKind]map[string]json.RawMessage
	own     []File
}

// splitCard sorts the card's files by place. Without a mark, the card predates the canon and
// every MCP and plugin record goes to the canon; with one, only the records on wrote do.
func splitCard(files []File, places cardPlaces, connectable bool) cardSplit {
	split := cardSplit{canon: map[string][]File{}, records: map[recordKind]map[string]json.RawMessage{}}
	for _, kind := range recordKinds {
		split.records[kind] = map[string]json.RawMessage{}
	}
	marked, ours := false, map[recordKind]map[string]bool{}
	for _, file := range files {
		if file.Path != MarkFile {
			continue
		}
		var mark canonMark
		if json.Unmarshal([]byte(file.Body), &mark) == nil {
			marked = true
			for kind, list := range mark.lists() {
				ours[kind] = map[string]bool{}
				for _, name := range *list {
					ours[kind][name] = true
				}
			}
		}
	}
	// take moves the records of one kind that go to the canon out of a settings body and
	// returns how many stay; -1 when the body has none of that kind.
	take := func(kind recordKind, top map[string]json.RawMessage) int {
		var records map[string]json.RawMessage
		if json.Unmarshal(top[kind.key], &records) != nil || records == nil {
			return -1
		}
		for name, record := range records {
			if !marked || ours[kind][name] {
				split.records[kind][name] = record
				delete(records, name)
			}
		}
		raw, _ := json.Marshal(records)
		top[kind.key] = raw
		return len(records)
	}
	// takeHooks moves the hook groups that go to the canon out of a settings body, back in Claude form.
	takeHooks := func(top map[string]json.RawMessage) bool {
		settings := &hookSettings{top: top}
		if json.Unmarshal(top["hooks"], &settings.events) != nil || settings.events == nil {
			return false
		}
		events, tools := invert(places.hookEvents), invert(places.hookTools)
		for name, group := range settings.records() {
			if marked && !ours[hookRecords][name] {
				continue
			}
			event, _, _ := strings.Cut(name, " ")
			canonEvent, record, ok := translateHook(event, group, events, tools)
			if !ok {
				continue
			}
			split.records[hookRecords][hookName(canonEvent, json.RawMessage(record))] = json.RawMessage(record)
			settings.remove(name)
		}
		settings.encode()
		return true
	}
	for _, file := range files {
		if file.Path == MarkFile {
			continue
		}
		if !connectable {
			split.own = append(split.own, file)
			continue
		}
		if element, rel, ok := canonPath(file.Path, places); ok {
			split.canon[element] = append(split.canon[element], File{Path: rel, Body: file.Body})
			continue
		}
		if places.hasMCP && (IsMCP(file.Path) || file.Path == places.mcp) {
			if top, _, ok := mcpServers(file.Body); ok {
				if take(mcpRecords, top) == 0 && IsMCP(file.Path) {
					continue
				}
				file.Body = reencode(top)
			}
		}
		plugins, hooks := places.plugins != "" && file.Path == places.plugins, places.hooks != "" && file.Path == places.hooks
		if plugins || hooks {
			var top map[string]json.RawMessage
			if json.Unmarshal([]byte(file.Body), &top) == nil && top != nil {
				changed := plugins && take(pluginRecords, top)+take(marketRecords, top) > -2
				if hooks && takeHooks(top) {
					changed = true
				}
				if changed {
					file.Body = reencode(top)
				}
			}
		}
		split.own = append(split.own, file)
	}
	return split
}

func reencode(top map[string]json.RawMessage) string {
	out, _ := json.MarshalIndent(top, "", "  ")
	return string(out) + "\n"
}

// canonPath maps a card path at a canon place to its canon element and canon path.
func canonPath(rel string, places cardPlaces) (string, string, bool) {
	if places.instructions != "" && rel == places.instructions {
		return "AGENTS.md", "AGENTS.md", true
	}
	for _, kind := range [][2]string{{places.skills, "skills"}, {places.agents, "agents"}, {places.commands, "commands"}} {
		if kind[0] == "" || !strings.HasPrefix(rel, kind[0]+"/") {
			continue
		}
		rest := strings.TrimPrefix(rel, kind[0]+"/")
		first, _, _ := strings.Cut(rest, "/")
		return kind[1] + "/" + first, kind[1] + "/" + rest, true
	}
	return "", "", false
}

// applyAgent applies an AI agent card through the canon (ADR 0037): canon elements replace
// same-name elements whole, own files replace same-name files, the rest stays. Then the agent
// is connected. A break puts the canon and the home back and drops the attempt's snapshots.
func applyAgent(root, account string, payload publicationPayload, broken bool) error {
	if err := pathsFit(payload.Files); err != nil {
		return err
	}
	home := agentHome(root, payload.Agent)
	places, connectable := agentPlaces(root, home, payload.Agent)
	split := splitCard(payload.Files, places, connectable)
	var snapshots []string
	changesCanon := len(split.canon) > 0
	for _, records := range split.records {
		changesCanon = changesCanon || len(records) > 0
	}
	if changesCanon {
		dir, err := snapshotTree(root, account, canonCard, canonDir(root))
		if err != nil {
			return err
		}
		snapshots = append(snapshots, dir)
	}
	dir, err := snapshotTree(root, account, payload.Agent, home)
	if err != nil {
		return errors.Join(err, removeAll(snapshots))
	}
	snapshots = append(snapshots, dir)
	var undo applyUndo
	err = writeCard(root, home, split, &undo, broken)
	if err == nil && connectable {
		if _, err = connect(root, account, payload.Agent, dir, home); errors.Is(err, errEmptyCanon) {
			err = nil
		}
	}
	if err != nil {
		undoErr := undo.rollback()
		// The snapshots hold the previous state until everything is back in place.
		if undoErr == nil {
			undoErr = removeAll(snapshots)
		}
		return errors.Join(err, undoErr)
	}
	return undo.finish(canonDir(root))
}

func snapshotTree(root, account, name, dir string) (string, error) {
	files, err := readTree(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return writeSnapshot(root, account, name, portableOnly(files))
}

func removeAll(dirs []string) error {
	var errs []error
	for _, dir := range dirs {
		errs = append(errs, os.RemoveAll(dir))
	}
	return errors.Join(errs...)
}

func writeCard(root, home string, split cardSplit, undo *applyUndo, broken bool) error {
	errBroken := errors.New("применение не выполнено")
	canon := canonDir(root)
	for _, element := range slices.Sorted(maps.Keys(split.canon)) {
		if err := undo.setAside(root, filepath.Join(canon, filepath.FromSlash(element))); err != nil {
			return err
		}
		for _, file := range split.canon[element] {
			if err := writeFile(canon, file); err != nil {
				return err
			}
		}
		if broken {
			return errBroken
		}
	}
	for _, kind := range recordKinds {
		records := split.records[kind]
		if len(records) == 0 {
			continue
		}
		file := filepath.Join(canon, canonRecordFile[kind])
		if err := undo.keep(file); err != nil {
			return err
		}
		settings, err := openSettings(kind, file)
		if err != nil {
			return err
		}
		for name, record := range records {
			settings.set(name, string(record))
		}
		if err := writeSettings(file, settings); err != nil {
			return err
		}
		if broken {
			return errBroken
		}
	}
	for _, file := range split.own {
		if err := undo.keep(filepath.Join(home, filepath.FromSlash(file.Path))); err != nil {
			return err
		}
		if err := writeHomeFile(home, file); err != nil {
			return err
		}
		if broken {
			return errBroken
		}
	}
	return nil
}

// applyUndo remembers what an apply replaced: canon elements moved aside and previous file bodies.
type applyUndo struct {
	aside string
	moved []savedLink
	kept  []keptFile
}

type keptFile struct {
	path    string
	body    []byte
	existed bool
}

func (u *applyUndo) setAside(root, path string) error {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		u.moved = append(u.moved, savedLink{path: path})
		return nil
	} else if err != nil {
		return err
	}
	if u.aside == "" {
		base := filepath.Join(root, ".agentsync")
		if err := os.MkdirAll(base, 0o755); err != nil {
			return err
		}
		aside, err := os.MkdirTemp(base, "apply-")
		if err != nil {
			return err
		}
		u.aside = aside
	}
	target := filepath.Join(u.aside, strconv.Itoa(len(u.moved)))
	if err := os.Rename(path, target); err != nil {
		return err
	}
	u.moved = append(u.moved, savedLink{path: path, target: target})
	return nil
}

func (u *applyUndo) keep(path string) error {
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		u.kept = append(u.kept, keptFile{path: path})
		return nil
	}
	if err != nil {
		return err
	}
	u.kept = append(u.kept, keptFile{path: path, body: body, existed: true})
	return nil
}

func (u *applyUndo) rollback() error {
	var errs []error
	for i := len(u.kept) - 1; i >= 0; i-- {
		kept := u.kept[i]
		if !kept.existed {
			if err := os.Remove(kept.path); err != nil && !errors.Is(err, os.ErrNotExist) {
				errs = append(errs, err)
			}
			continue
		}
		if err := os.WriteFile(kept.path, kept.body, 0o644); err != nil {
			errs = append(errs, err)
		}
	}
	for i := len(u.moved) - 1; i >= 0; i-- {
		moved := u.moved[i]
		if err := os.RemoveAll(moved.path); err != nil {
			errs = append(errs, err)
			continue
		}
		if moved.target != "" {
			if err := os.Rename(moved.target, moved.path); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) == 0 && u.aside != "" {
		return os.RemoveAll(u.aside)
	}
	return errors.Join(errs...)
}

// finish puts the private files of replaced canon elements (.env, node_modules) back into the
// new elements, then drops the aside folder. On an error the aside folder stays with what is left.
func (u *applyUndo) finish(canon string) error {
	for _, moved := range u.moved {
		if moved.target == "" {
			continue
		}
		err := filepath.WalkDir(moved.target, func(path string, entry os.DirEntry, err error) error {
			if err != nil || path == moved.target {
				return err
			}
			rel, err := filepath.Rel(moved.target, path)
			if err != nil {
				return err
			}
			dest := filepath.Join(moved.path, rel)
			canonRel, err := filepath.Rel(canon, dest)
			if err != nil || !PrivatePath(filepath.ToSlash(canonRel)) {
				return err
			}
			if _, err := os.Lstat(dest); err == nil {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			if err := os.Rename(path, dest); err != nil {
				return err
			}
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	if u.aside == "" {
		return nil
	}
	return os.RemoveAll(u.aside)
}
