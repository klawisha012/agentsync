package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// canonCard is the card and store name of the canon ~/.agents.
const canonCard = "agents"

func isCanonCard(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == canonCard || name == ".agents"
}

// applyCanon makes the canon an exact copy of a canon card. Private files such as .env and
// machine MCP servers stay. A break puts the canon back and drops the attempt's snapshot.
func applyCanon(root, account string, files []File, broken bool) error {
	if err := pathsFit(files); err != nil {
		return err
	}
	for _, file := range files {
		if file.Path == "hooks.json" {
			if _, err := parseCanonHooks([]byte(file.Body)); err != nil {
				return err
			}
		}
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
		return reconnect(root, account)
	}
	undoErr := replacePortable(root, canon, before, false)
	// The snapshot holds the previous canon until it is back in place.
	if undoErr == nil {
		undoErr = os.RemoveAll(snapshot)
	}
	return errors.Join(err, undoErr)
}

// reconnect refreshes every AI agent connected to the canon after the canon changed: links,
// MCP servers, plugins and hooks follow the new canon.
func reconnect(root, account string) error {
	var errs []error
	unset := map[string]bool{}
	for _, receiver := range receivers {
		if !connected(root, receiver) {
			continue
		}
		report, err := connect(root, account, receiver.Slug, "", "")
		if err != nil && !errors.Is(err, errEmptyCanon) {
			errs = append(errs, fmt.Errorf("agents обновлён, но\u00a0ИИ-агент «%s» не\u00a0подключён заново: %w", receiver.Display, err))
		}
		for _, name := range report.Unset {
			unset[name] = true
		}
	}
	if len(errs) == 0 && len(unset) > 0 {
		return UnsetVariables(slices.Sorted(maps.Keys(unset)))
	}
	return errors.Join(errs...)
}

// connected reports an AI agent that holds records on wrote or a link into the canon.
func connected(root string, receiver Receiver) bool {
	for _, kind := range append(slices.Clone(recordKinds), fileRecords) {
		if _, ledger, err := loadLedger(kind, receiver.Slug); err == nil && ledger.File != "" {
			return true
		}
	}
	dir, have := receiver.GlobalDir(root)
	if !have || insideCanon(root, dir) {
		return false
	}
	entries, _ := os.ReadDir(dir)
	canon := canonDir(root)
	for _, entry := range entries {
		if target, ok := linkTarget(filepath.Join(dir, entry.Name())); ok && within(canon, target) {
			return true
		}
	}
	return false
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

// cardSplit holds chosen canon elements and records for selective apply.
type cardSplit struct {
	canon   map[string][]File
	records map[recordKind]map[string]json.RawMessage
}

func splitCanonFile(file File) cardSplit {
	split := cardSplit{canon: map[string][]File{}, records: map[recordKind]map[string]json.RawMessage{}}
	for _, kind := range recordKinds {
		split.records[kind] = map[string]json.RawMessage{}
	}
	if file.Path == "AGENTS.md" {
		split.canon["AGENTS.md"] = []File{{Path: "AGENTS.md", Body: file.Body}}
		return split
	}
	for _, dir := range []string{"skills", "agents", "commands"} {
		if strings.HasPrefix(file.Path, dir+"/") {
			rest := strings.TrimPrefix(file.Path, dir+"/")
			first, _, _ := strings.Cut(rest, "/")
			element := dir + "/" + first
			split.canon[element] = []File{{Path: file.Path, Body: file.Body}}
			return split
		}
	}
	if IsMCP(file.Path) || file.Path == "mcp.json" {
		if _, servers, ok := mcpServers(file.Body); ok {
			maps.Copy(split.records[mcpRecords], servers)
		}
		return split
	}
	if file.Path == "plugins.json" {
		var top map[string]json.RawMessage
		if json.Unmarshal([]byte(file.Body), &top) == nil && top != nil {
			for _, kind := range []recordKind{pluginRecords, marketRecords} {
				var records map[string]json.RawMessage
				if json.Unmarshal(top[kind.key], &records) == nil && records != nil {
					maps.Copy(split.records[kind], records)
				}
			}
		}
	}
	return split
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

func writeCard(root string, split cardSplit, undo *applyUndo, broken bool) error {
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
		var settings mcpSettings
		var err error
		if kind == hookRecords {
			settings, err = readCanonHooks(file)
		} else {
			settings, err = openSettings(kind, file)
		}
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
