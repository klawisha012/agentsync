package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// ExportItem is the outcome of exporting one AI agent into ~/.agents.
type ExportItem struct {
	Name   string
	Report TransferReport
	Err    error
}

type skillMove struct {
	name  string
	files []File
}

type exportPlan struct {
	skills []skillMove
	docs   []docMove
	inst   *docMove
	moves  []ownMove
	report TransferReport
}

func (p exportPlan) empty() bool {
	return len(p.skills) == 0 && len(p.docs) == 0 && p.inst == nil && len(p.moves) == 0
}

var exportKinds = []string{"skills", "agents", "commands", "instructions", "mcp", "plugins", "hooks"}

// ExportKinds lists the supported element kinds for selective export.
func ExportKinds() []string {
	return slices.Clone(exportKinds)
}

// NormalizeExportKind returns the canonical export kind for name or false if unknown.
func NormalizeExportKind(name string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "subagents" {
		key = "agents"
	}
	if slices.Contains(exportKinds, key) {
		return key, true
	}
	return "", false
}

func normalizeExportKinds(kinds []string) (map[string]bool, error) {
	if len(kinds) == 0 {
		return nil, nil
	}
	want := make(map[string]bool, len(kinds))
	for _, raw := range kinds {
		norm, ok := NormalizeExportKind(raw)
		if !ok {
			return nil, fmt.Errorf("Неизвестный вид «%s». Доступны: %s.", strings.TrimSpace(raw), strings.Join(exportKinds, ", "))
		}
		want[norm] = true
	}
	return want, nil
}

func wantExportKind(want map[string]bool, kind string) bool {
	return len(want) == 0 || want[kind]
}

// Export copies and translates portable elements of the named AI agents (or every installed
// catalog AI agent when names is empty) into ~/.agents without removing the agents' own files.
// When kinds is non-empty, only the named element kinds are exported.
func Export(root, account string, names, kinds []string) ([]ExportItem, error) {
	want, err := normalizeExportKinds(kinds)
	if err != nil {
		return nil, err
	}
	if err := checkCanon(root, canonCard); err != nil {
		return nil, err
	}
	var out []ExportItem
	err = withRootLock(root, func() error {
		targets, labels, err := resolveExportTargets(root, names)
		if err != nil {
			return err
		}
		snapshot := ""
		wroteAny := false
		for i, receiver := range targets {
			plan, planErr := planExportOne(root, receiver, want)
			if planErr != nil {
				if len(names) == 1 {
					return planErr
				}
				out = append(out, ExportItem{Name: labels[i], Err: planErr})
				continue
			}
			if plan.empty() {
				out = append(out, ExportItem{Name: labels[i], Report: plan.report})
				continue
			}
			if snapshot == "" {
				snapshot, err = snapshotTree(root, account, canonCard, canonDir(root))
				if err != nil {
					return err
				}
			}
			if writeErr := applyExportPlan(root, plan); writeErr != nil {
				if !wroteAny {
					_ = removeAll([]string{snapshot})
				}
				if len(names) == 1 {
					return writeErr
				}
				out = append(out, ExportItem{Name: labels[i], Err: writeErr})
				continue
			}
			wroteAny = true
			out = append(out, ExportItem{Name: labels[i], Report: plan.report})
		}
		return nil
	})
	return out, err
}

func resolveExportTargets(root string, names []string) ([]Receiver, []string, error) {
	if len(names) == 0 {
		var targets []Receiver
		var labels []string
		for _, r := range receivers {
			if receiverIsCanon(root, r) || !receiverPresent(root, r) {
				continue
			}
			targets = append(targets, r)
			labels = append(labels, r.Slug)
		}
		if len(targets) == 0 {
			return nil, nil, errors.New("На компьютере нет ИИ-агентов.")
		}
		return targets, labels, nil
	}
	var targets []Receiver
	var labels []string
	seen := map[string]bool{}
	for _, raw := range names {
		if isCanonCard(raw) {
			return nil, nil, errors.New("Настройки agents уже лежат в\u00a0~/.agents.")
		}
		r, ok := FindReceiver(raw)
		if !ok {
			return nil, nil, fmt.Errorf("ИИ-агента «%s» нет в\u00a0каталоге ИИ-агентов.", strings.TrimSpace(raw))
		}
		if receiverIsCanon(root, r) {
			return nil, nil, fmt.Errorf("ИИ-агент «%s» уже читает настройки прямо из\u00a0~/.agents.", r.Display)
		}
		if !receiverPresent(root, r) {
			return nil, nil, errors.New("на компьютере нет этого ИИ-агента")
		}
		if seen[r.Slug] {
			continue
		}
		seen[r.Slug] = true
		targets = append(targets, r)
		labels = append(labels, strings.TrimSpace(raw))
	}
	return targets, labels, nil
}

func receiverIsCanon(root string, r Receiver) bool {
	dir, have := r.GlobalDir(root)
	if !have {
		return r.MCP == ""
	}
	return samePlace(realPath(filepath.Dir(dir)), realPath(canonDir(root))) && r.MCP == ""
}

func receiverPresent(root string, r Receiver) bool {
	if dir, have := r.GlobalDir(root); have {
		home := filepath.Dir(dir)
		if !samePlace(realPath(home), realPath(canonDir(root))) && !samePlace(realPath(home), realPath(root)) {
			if info, err := os.Stat(home); err == nil && info.IsDir() {
				return true
			}
		}
	}
	if r.MCP != "" {
		if info, err := os.Stat(r.mcpFile(root)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func planExportOne(root string, receiver Receiver, want map[string]bool) (exportPlan, error) {
	var plan exportPlan
	dir, _ := receiver.GlobalDir(root)

	var (
		skills  []skillMove
		docs    []docMove
		inst    *docMove
		hooks   []ownMove
		servers []ownMove
		plugins []ownMove
		err     error
	)
	if wantExportKind(want, "skills") {
		skills, err = exportSkills(root, dir, &plan.report)
		if err != nil {
			return plan, err
		}
	}
	if wantExportKind(want, "agents") || wantExportKind(want, "commands") {
		docs, err = exportDocs(root, receiver, dir, want, &plan.report)
		if err != nil {
			return plan, err
		}
	}
	if wantExportKind(want, "instructions") {
		inst, err = exportInstructions(root, receiver, dir, &plan.report)
		if err != nil {
			return plan, err
		}
	}
	if wantExportKind(want, "hooks") {
		hooks, err = exportHooks(root, receiver, dir)
		if err != nil {
			return plan, err
		}
	}
	if wantExportKind(want, "mcp") {
		servers, err = exportServers(root, receiver, &plan.report)
		if err != nil {
			return plan, err
		}
	}
	if wantExportKind(want, "plugins") {
		plugins, err = exportPlugins(root, receiver, dir, &plan.report)
		if err != nil {
			return plan, err
		}
	}

	plan.skills = skills
	plan.docs = docs
	plan.inst = inst
	plan.moves = append(append(hooks, servers...), plugins...)

	if len(skills) > 0 {
		names := make([]string, len(skills))
		for i, s := range skills {
			names[i] = s.name
		}
		slices.Sort(names)
		plan.report.Moved = append(plan.report.Moved, "навыки: "+strings.Join(names, ", "))
	}
	for _, kind := range []docKind{subagentDoc, commandDoc} {
		var names []string
		for _, move := range docs {
			if move.kind.key == kind.key {
				names = append(names, strings.TrimSuffix(strings.TrimPrefix(move.rel, kind.key+"/"), ".md"))
			}
		}
		if len(names) > 0 {
			slices.Sort(names)
			plan.report.Moved = append(plan.report.Moved, kind.name+": "+strings.Join(names, ", "))
		}
	}
	if inst != nil {
		plan.report.Moved = append(plan.report.Moved, "файл инструкций")
	}
	if len(hooks) > 0 {
		plan.report.Moved = append(plan.report.Moved, fmt.Sprintf("хуки (%d)", len(hooks)))
	}
	if len(servers) > 0 {
		names := make([]string, len(servers))
		for i, m := range servers {
			names[i] = m.canon
		}
		slices.Sort(names)
		plan.report.Moved = append(plan.report.Moved, "MCP-серверы: "+strings.Join(names, ", "))
	}
	for _, item := range []struct {
		kind  recordKind
		label string
	}{{pluginRecords, "плагины"}, {marketRecords, "маркетплейсы"}} {
		var names []string
		for _, m := range plugins {
			if m.kind == item.kind {
				names = append(names, m.canon)
			}
		}
		if len(names) > 0 {
			slices.Sort(names)
			plan.report.Moved = append(plan.report.Moved, item.label+": "+strings.Join(names, ", "))
		}
	}
	return plan, nil
}

func applyExportPlan(root string, plan exportPlan) error {
	canon := canonDir(root)
	var undo applyUndo
	err := func() error {
		for _, skill := range plan.skills {
			if err := undo.setAside(root, filepath.Join(canon, "skills", skill.name)); err != nil {
				return err
			}
			for _, f := range skill.files {
				if err := writeFile(canon, f); err != nil {
					return err
				}
			}
		}
		for _, move := range plan.docs {
			if err := undo.keep(filepath.Join(canon, filepath.FromSlash(move.rel))); err != nil {
				return err
			}
			if err := writeFile(canon, File{Path: move.rel, Body: move.body}); err != nil {
				return err
			}
		}
		if plan.inst != nil {
			if err := undo.keep(filepath.Join(canon, filepath.FromSlash(plan.inst.rel))); err != nil {
				return err
			}
			if err := writeFile(canon, File{Path: plan.inst.rel, Body: plan.inst.body}); err != nil {
				return err
			}
		}
		return writeCanonMoves(canon, plan.moves, &undo)
	}()
	if err != nil {
		return errors.Join(err, undo.rollback())
	}
	return undo.finish(canon)
}

func writeCanonMoves(canon string, moves []ownMove, undo *applyUndo) error {
	if len(moves) == 0 {
		return nil
	}
	byFile := map[string][]ownMove{}
	for _, move := range moves {
		file := filepath.Join(canon, canonRecordFile[move.kind])
		byFile[file] = append(byFile[file], move)
	}
	for _, file := range slices.Sorted(maps.Keys(byFile)) {
		if err := undo.keep(file); err != nil {
			return err
		}
		fileMoves := byFile[file]
		if filepath.Base(file) == "hooks.json" {
			settings, err := readCanonHooks(file)
			if err != nil {
				return err
			}
			for _, m := range fileMoves {
				settings.set(m.canon, m.body)
			}
			if err := writeSettings(file, settings); err != nil {
				return err
			}
			continue
		}
		for _, kind := range []recordKind{mcpRecords, pluginRecords, marketRecords} {
			var kindMoves []ownMove
			for _, m := range fileMoves {
				if m.kind == kind {
					kindMoves = append(kindMoves, m)
				}
			}
			if len(kindMoves) == 0 {
				continue
			}
			settings, err := openSettings(kind, file)
			if err != nil {
				return err
			}
			for _, m := range kindMoves {
				settings.set(m.canon, m.body)
			}
			if err := writeSettings(file, settings); err != nil {
				return err
			}
		}
	}
	return nil
}

func exportSkills(root, dir string, report *TransferReport) ([]skillMove, error) {
	if dir == "" || insideCanon(root, dir) {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	canon := canonDir(root)
	var out []skillMove
	for _, entry := range entries {
		name := entry.Name()
		skillDir := filepath.Join(dir, name)
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if target, ok := linkTarget(skillDir); ok && within(realPath(canon), realPath(target)) {
			continue
		}
		if !entry.IsDir() || PrivatePath("skills/"+name) || !PortablePath("skills/"+name) {
			continue
		}
		files, err := skillTree(skillDir, name)
		if err != nil {
			return nil, err
		}
		if !slices.ContainsFunc(files, func(f File) bool { return strings.EqualFold(f.Path, "skills/"+name+"/SKILL.md") }) {
			continue
		}
		for _, f := range files {
			rel := strings.TrimPrefix(f.Path, "skills/"+name+"/")
			if msg := SecretExplanation(filepath.Join(skillDir, filepath.FromSlash(rel)), f.Body); msg != "" {
				return nil, errors.New(msg)
			}
		}
		dest := filepath.Join(canon, "skills", name)
		if _, err := os.Lstat(dest); err == nil {
			if have, err := skillTree(dest, name); err == nil && equalFileSlices(have, files) {
				continue
			}
			report.Kept = append(report.Kept, "навык "+name+" (имя занято в\u00a0agents)")
			continue
		}
		out = append(out, skillMove{name: name, files: files})
	}
	return out, nil
}

func skillTree(skillDir, name string) ([]File, error) {
	raw, err := readTree(skillDir)
	if err != nil {
		return nil, err
	}
	prefixed := make([]File, len(raw))
	for i, f := range raw {
		prefixed[i] = File{Path: "skills/" + name + "/" + f.Path, Body: f.Body}
	}
	return portableOnly(prefixed), nil
}

func equalFileSlices(a, b []File) bool {
	if len(a) != len(b) {
		return false
	}
	sa := slices.Clone(a)
	sb := slices.Clone(b)
	cmpFile := func(x, y File) int { return strings.Compare(x.Path, y.Path) }
	slices.SortFunc(sa, cmpFile)
	slices.SortFunc(sb, cmpFile)
	for i := range sa {
		if sa[i].Path != sb[i].Path || sa[i].Body != sb[i].Body {
			return false
		}
	}
	return true
}

func exportDocs(root string, receiver Receiver, dir string, want map[string]bool, report *TransferReport) ([]docMove, error) {
	if dir == "" {
		return nil, nil
	}
	_, ledger, err := loadFileLedger(receiver.Slug)
	if err != nil {
		return nil, err
	}
	d := dialects[receiver.Slug]
	canon := canonDir(root)
	var moves []docMove
	for _, item := range []struct {
		kind  docKind
		place string
	}{{subagentDoc, receiver.Agents}, {commandDoc, receiver.Commands}} {
		if !wantExportKind(want, item.kind.key) || receiver.Way(item.kind.key) != WayTranslate {
			continue
		}
		place := filepath.Join(filepath.Dir(dir), filepath.FromSlash(item.place))
		if insideCanon(root, place) {
			continue
		}
		entries, err := os.ReadDir(place)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			file := filepath.Join(place, entry.Name())
			if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() || containsPath(ledger.Own, file) {
				continue
			}
			if _, linked := linkTarget(file); linked {
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
			body, ok := fromNativeDoc(item.kind, d, string(raw))
			rel := item.kind.key + "/" + stem + ".md"
			dest := filepath.Join(canon, filepath.FromSlash(rel))
			if _, err := os.Lstat(dest); err == nil {
				if ok {
					if have, err := os.ReadFile(dest); err == nil && string(have) == body {
						continue
					}
				}
				report.Kept = append(report.Kept, item.kind.one+" "+stem+" (имя занято в\u00a0agents)")
				continue
			}
			if !ok {
				report.Kept = append(report.Kept, item.kind.one+" "+stem)
				continue
			}
			moves = append(moves, docMove{kind: item.kind, file: file, rel: rel, body: body})
		}
	}
	return moves, nil
}

func exportInstructions(root string, receiver Receiver, dir string, report *TransferReport) (*docMove, error) {
	if dir == "" || receiver.Instructions == "" {
		return nil, nil
	}
	instFile := filepath.Join(filepath.Dir(dir), filepath.FromSlash(receiver.Instructions))
	if insideCanon(root, filepath.Dir(instFile)) {
		return nil, nil
	}
	info, err := os.Lstat(instFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
		return nil, nil
	}
	if _, linked := linkTarget(instFile); linked {
		return nil, nil
	}
	raw, err := os.ReadFile(instFile)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return nil, nil
	}
	if msg := SecretExplanation(instFile, string(raw)); msg != "" {
		return nil, errors.New(msg)
	}
	dest := filepath.Join(canonDir(root), "AGENTS.md")
	if _, err := os.Lstat(dest); err == nil {
		if have, err := os.ReadFile(dest); err == nil && string(have) == string(raw) {
			return nil, nil
		}
		report.Kept = append(report.Kept, "файл инструкций (имя занято в\u00a0agents)")
		return nil, nil
	}
	return &docMove{file: instFile, rel: "AGENTS.md", body: string(raw)}, nil
}

func exportHooks(root string, receiver Receiver, dir string) ([]ownMove, error) {
	if dir == "" || receiver.Hooks == "" {
		return nil, nil
	}
	hookFile := filepath.Join(filepath.Dir(dir), filepath.FromSlash(receiver.Hooks))
	if insideCanon(root, filepath.Dir(hookFile)) {
		return nil, nil
	}
	moves, err := ownHooks(root, receiver, dir)
	if err != nil || len(moves) == 0 {
		return moves, err
	}
	canon, err := readCanonHooks(filepath.Join(canonDir(root), "hooks.json"))
	if err != nil {
		return nil, err
	}
	var fresh []ownMove
	for _, m := range moves {
		if _, have := canon.get(m.canon); have {
			continue
		}
		fresh = append(fresh, m)
	}
	return fresh, nil
}

func exportServers(root string, receiver Receiver, report *TransferReport) ([]ownMove, error) {
	if receiver.MCP == "" {
		return nil, nil
	}
	file := receiver.mcpFile(root)
	settings, err := readSettings(file, mcpRecords.key)
	if err != nil {
		return nil, err
	}
	canon, err := readSettings(filepath.Join(canonDir(root), "mcp.json"), mcpRecords.key)
	if err != nil {
		return nil, err
	}
	_, ledger, err := loadLedger(mcpRecords, receiver.Slug)
	if err != nil {
		return nil, err
	}
	servers := map[string]json.RawMessage{}
	switch own := settings.(type) {
	case *jsonSettings:
		for name, server := range own.servers {
			servers[name] = fromGemini(server)
		}
	case *tomlSettings:
		for _, name := range own.names() {
			block, _ := own.get(name)
			server, ok := parseTOMLServer(block)
			if !ok {
				if !slices.Contains(ledger.Own, name) {
					report.Kept = append(report.Kept, "MCP-сервер "+name)
				}
				continue
			}
			servers[name] = server
		}
	}
	var moves []ownMove
	for _, name := range slices.Sorted(maps.Keys(servers)) {
		server := servers[name]
		if slices.Contains(ledger.Own, name) || AbsolutePath.Match(server) {
			continue
		}
		if msg := SecretExplanation(file, string(server)); msg != "" {
			return nil, errors.New(msg)
		}
		if have, taken := canon.get(name); taken {
			if canonicalJSON(json.RawMessage(have)) == canonicalJSON(server) {
				continue
			}
			report.Kept = append(report.Kept, "MCP-сервер "+name+" (имя занято в\u00a0agents)")
			continue
		}
		moves = append(moves, ownMove{kind: mcpRecords, file: file, native: name, canon: name, body: string(server)})
	}
	return moves, nil
}

func exportPlugins(root string, receiver Receiver, dir string, report *TransferReport) ([]ownMove, error) {
	if dir == "" || receiver.Plugins == "" {
		return nil, nil
	}
	file := filepath.Join(filepath.Dir(dir), filepath.FromSlash(receiver.Plugins))
	if insideCanon(root, filepath.Dir(file)) {
		return nil, nil
	}
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	var moves []ownMove
	for _, item := range []struct {
		kind recordKind
		one  string
	}{{pluginRecords, "плагин"}, {marketRecords, "маркетплейс"}} {
		settings, err := readSettings(file, item.kind.key)
		if err != nil {
			return nil, err
		}
		canon, err := readSettings(filepath.Join(canonDir(root), "plugins.json"), item.kind.key)
		if err != nil {
			return nil, err
		}
		_, ledger, err := loadLedger(item.kind, receiver.Slug)
		if err != nil {
			return nil, err
		}
		own, ok := settings.(*jsonSettings)
		if !ok {
			continue
		}
		for _, name := range slices.Sorted(maps.Keys(own.servers)) {
			record := own.servers[name]
			if slices.Contains(ledger.Own, name) || AbsolutePath.Match(record) {
				continue
			}
			if msg := SecretExplanation(file, string(record)); msg != "" {
				return nil, errors.New(msg)
			}
			if have, taken := canon.get(name); taken {
				if canonicalJSON(json.RawMessage(have)) == canonicalJSON(record) {
					continue
				}
				report.Kept = append(report.Kept, item.one+" "+name+" (имя занято в\u00a0agents)")
				continue
			}
			moves = append(moves, ownMove{kind: item.kind, file: file, native: name, canon: name, body: string(record)})
		}
	}
	return moves, nil
}
