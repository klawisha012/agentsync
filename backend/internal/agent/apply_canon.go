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
	MCP []string `json:"mcp"`
}

// markFile builds the mark for an upload of home: the records on wrote into a settings file inside it.
func markFile(home string) File {
	mark := canonMark{MCP: []string{}}
	if dir, err := stateDir(); err == nil {
		ledgers, _ := filepath.Glob(filepath.Join(dir, "mcp", "*.json"))
		for _, file := range ledgers {
			var ledger mcpLedger
			if raw, err := os.ReadFile(file); err == nil && json.Unmarshal(raw, &ledger) == nil && within(home, ledger.File) {
				mark.MCP = append(mark.MCP, ledger.Own...)
			}
		}
	}
	raw, _ := json.Marshal(mark)
	return File{Path: MarkFile, Body: string(raw)}
}

// cardPlaces are the canon places of an AI agent, relative to its home; "" when it has none.
type cardPlaces struct {
	skills, agents, commands, instructions, mcp string
	hasMCP                                      bool
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
	return places, true
}

// cardSplit is a card taken apart: canon elements with canon paths, MCP records for the
// canon, and the agent's own files.
type cardSplit struct {
	canon map[string][]File
	mcp   map[string]json.RawMessage
	own   []File
}

// splitCard sorts the card's files by place. Without a mark, the card predates the canon and
// every MCP record goes to the canon; with one, only the records on wrote do.
func splitCard(files []File, places cardPlaces, connectable bool) cardSplit {
	split := cardSplit{canon: map[string][]File{}, mcp: map[string]json.RawMessage{}}
	marked, ours := false, map[string]bool{}
	for _, file := range files {
		if file.Path != MarkFile {
			continue
		}
		var mark canonMark
		if json.Unmarshal([]byte(file.Body), &mark) == nil {
			marked = true
			for _, name := range mark.MCP {
				ours[name] = true
			}
		}
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
			if top, servers, ok := mcpServers(file.Body); ok {
				for name, server := range servers {
					if !marked || ours[name] {
						split.mcp[name] = server
						delete(servers, name)
					}
				}
				if len(servers) == 0 && IsMCP(file.Path) {
					continue
				}
				file.Body = encodeMCP(top, servers)
			}
		}
		split.own = append(split.own, file)
	}
	return split
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
	if len(split.canon)+len(split.mcp) > 0 {
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
		if _, err = connect(root, account, payload.Agent); errors.Is(err, errEmptyCanon) {
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
	return undo.finish()
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
	if len(split.mcp) > 0 {
		file := filepath.Join(canon, "mcp.json")
		if err := undo.keep(file); err != nil {
			return err
		}
		settings, err := readSettings(file)
		if err != nil {
			return err
		}
		for name, server := range split.mcp {
			settings.set(name, string(server))
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
	if len(errs) == 0 {
		return u.finish()
	}
	return errors.Join(errs...)
}

func (u *applyUndo) finish() error {
	if u.aside == "" {
		return nil
	}
	return os.RemoveAll(u.aside)
}
