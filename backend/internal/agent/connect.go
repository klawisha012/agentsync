package agent

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
)

// These match the secret recognition запись already uses.
var (
	secretValue = regexp.MustCompile(`(?i)"?(?:api[_-]?key|secret|token|password|authorization)"?\s*[:=]\s*"?[A-Za-z0-9_+\-/]{8,}`)
	knownSecret = regexp.MustCompile(`(?:sk-[A-Za-z0-9]{10,}|AKIA[0-9A-Z]{16}|-----BEGIN [A-Z ]*PRIVATE KEY-----)`)
	// authValue is a literal Authorization or Bearer value with any scheme; ${NAME} is a secret variable and passes.
	authValue = regexp.MustCompile(`(?i)(?:authorization"?\s*[:=]\s*"?(?:[a-z]+\s+)?|bearer\s+)[A-Za-z0-9._~+/=\-]{8,}`)
)

type canonSkill struct {
	name string
	rel  string
	abs  string
}

type skillCatalog struct {
	dir      string
	display  string
	slug     string
	receiver Receiver
}

// canonLink is one canon element an AI agent sees by a link at dest.
type canonLink struct {
	rel  string
	abs  string
	dest string
}

// connectPlan holds the links for one AI agent and the places where its old canon links live.
type connectPlan struct {
	links  []canonLink
	dirs   []string
	files  []string
	report ConnectReport
}

// ConnectReport names the canon kinds the AI agent sees and the kinds it does not understand.
type ConnectReport struct {
	Got     []string
	Skipped []string
}

type savedLink struct {
	path   string
	target string
}

type catalogUndo struct {
	added       []string
	removed     []savedLink
	moved       []savedLink
	createdDirs []string
}

// Connect shows the canon to the named AI agent by links: outermost skills, subagents, user
// commands and the instructions file, each at the place the catalog names. MCP servers are
// translated into the agent settings file. A kind without a place is skipped. A name the
// agent already holds goes to its snapshot first.
func Connect(root, account, agentName string) (ConnectReport, error) {
	unlock, err := lockRoot(root)
	if err != nil {
		return ConnectReport{}, err
	}
	defer unlock()
	return connect(root, account, agentName)
}

var errEmptyCanon = errors.New("В\u00a0каноне нечего показать.")

func connect(root, account, agentName string) (ConnectReport, error) {
	canon := canonDir(root)
	skills, files, err := canonSkills(canon)
	if err != nil {
		return ConnectReport{}, err
	}
	cat, err := agentCatalog(root, agentName)
	if err != nil {
		return ConnectReport{}, err
	}
	plan, err := planConnect(root, cat, skills)
	if err != nil {
		return ConnectReport{}, err
	}
	servers, haveMCP, err := canonMCP(root)
	if err != nil {
		return plan.report, err
	}
	if haveMCP && cat.receiver.MCP != "" {
		plan.report.Got = append(plan.report.Got, "MCP-серверы")
	} else if haveMCP {
		plan.report.Skipped = append(plan.report.Skipped, "MCP-серверы")
	}
	if len(plan.report.Got)+len(plan.report.Skipped) == 0 {
		return plan.report, errEmptyCanon
	}
	if !insideCanon(root, cat.dir) {
		if msg := sameNameExplanation(skills); msg != "" {
			return plan.report, errors.New(msg)
		}
	}
	if msg := linkSecret(plan.links, files); msg != "" {
		return plan.report, errors.New(msg)
	}
	occupied, err := occupiedNames(plan.links)
	if err != nil {
		return plan.report, err
	}
	snapshot, home := "", ""
	if len(occupied) > 0 {
		if snapshot, home, err = snapshotAgent(root, account, cat); err != nil {
			return plan.report, err
		}
	}
	undo, err := shareCanon(canon, snapshot, home, cat.display, plan, occupied)
	if err == nil && cat.receiver.MCP != "" {
		err = connectMCP(root, cat.receiver, servers)
	}
	if err != nil {
		undoErr := rollbackCatalog(undo)
		// The snapshot holds the moved items until every one is back in place.
		if undoErr == nil && snapshot != "" {
			_ = os.RemoveAll(snapshot)
		}
		return plan.report, errors.Join(err, undoErr)
	}
	return plan.report, nil
}

// Disconnect removes the links to the canon from every place of the AI agent.
// The canon stays. A file or folder that is not a link to the canon stays.
func Disconnect(root, agentName string) error {
	unlock, err := lockRoot(root)
	if err != nil {
		return err
	}
	defer unlock()
	receiver, ok := FindReceiver(agentName)
	if !ok {
		return fmt.Errorf("ИИ-агента «%s» нет в\u00a0каталоге ИИ-агентов.", strings.TrimSpace(agentName))
	}
	dir, have := receiver.GlobalDir(root)
	if !have {
		return nil
	}
	if insideCanon(root, dir) {
		return fmt.Errorf("ИИ-агент «%s» читает навыки прямо из\u00a0канона, отключить их нельзя.", receiver.Display)
	}
	cat := skillCatalog{dir: dir, display: receiver.Display, slug: receiver.Slug, receiver: receiver}
	plan, err := planConnect(root, cat, nil)
	if err != nil {
		return err
	}
	stale, err := staleLinks(canonDir(root), plan.dirs, plan.files, nil)
	if err != nil {
		return err
	}
	for _, link := range stale {
		if err := os.Remove(link); err != nil {
			return err
		}
	}
	return disconnectMCP(receiver.Slug)
}

// planConnect lays out the links. A place inside the canon is left alone: the agent reads the canon there.
func planConnect(root string, cat skillCatalog, skills []canonSkill) (connectPlan, error) {
	canon := canonDir(root)
	var plan connectPlan
	if len(skills) > 0 {
		plan.report.Got = append(plan.report.Got, "навыки")
	}
	if !insideCanon(root, cat.dir) {
		plan.dirs = append(plan.dirs, cat.dir)
		for _, skill := range skills {
			plan.links = append(plan.links, canonLink{rel: skill.rel, abs: skill.abs, dest: filepath.Join(cat.dir, skill.name)})
		}
	}
	home := filepath.Dir(cat.dir)
	for _, kind := range []struct {
		name, rel, place string
		dir              bool
	}{
		{"субагенты", "agents", cat.receiver.Agents, true},
		{"пользовательские команды", "commands", cat.receiver.Commands, true},
		{"файл инструкций", "AGENTS.md", cat.receiver.Instructions, false},
	} {
		names, err := canonNames(canon, kind.rel, kind.dir)
		if err != nil {
			return plan, err
		}
		if kind.place == "" {
			if len(names) > 0 {
				plan.report.Skipped = append(plan.report.Skipped, kind.name)
			}
			continue
		}
		if len(names) > 0 {
			plan.report.Got = append(plan.report.Got, kind.name)
		}
		place := filepath.Join(home, filepath.FromSlash(kind.place))
		if !kind.dir {
			if insideCanon(root, filepath.Dir(place)) {
				continue
			}
			plan.files = append(plan.files, place)
			if len(names) > 0 {
				plan.links = append(plan.links, canonLink{rel: kind.rel, abs: filepath.Join(canon, kind.rel), dest: place})
			}
			continue
		}
		if insideCanon(root, place) {
			continue
		}
		plan.dirs = append(plan.dirs, place)
		for _, name := range names {
			plan.links = append(plan.links, canonLink{
				rel:  kind.rel + "/" + name,
				abs:  filepath.Join(canon, kind.rel, name),
				dest: filepath.Join(place, name),
			})
		}
	}
	return plan, nil
}

// canonNames lists a kind in the canon: the top-level .md files and folders of a kind folder,
// or the file itself for the instructions file.
func canonNames(canon, rel string, dir bool) ([]string, error) {
	full := filepath.Join(canon, rel)
	if !dir {
		info, err := os.Stat(full)
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		if err != nil || info.IsDir() {
			return nil, err
		}
		return []string{rel}, nil
	}
	entries, err := os.ReadDir(full)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if PrivatePath(rel+"/"+name) || !(entry.IsDir() || strings.HasSuffix(strings.ToLower(name), ".md")) {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

func canonDir(root string) string {
	return filepath.Join(root, ".agents")
}

// insideCanon is true when the agent's skills directory is the canon itself or resolves into it.
func insideCanon(root, dir string) bool {
	return within(realPath(canonDir(root)), realPath(dir))
}

func canonSkills(home string) ([]canonSkill, []File, error) {
	files, err := readTree(home)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var dirs []string
	for _, file := range files {
		if !strings.EqualFold(path.Base(file.Path), "SKILL.md") {
			continue
		}
		dir := path.Dir(file.Path)
		if !strings.HasPrefix(dir, "skills/") || PrivatePath(file.Path) || !PortablePath(file.Path) {
			continue
		}
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)
	var kept []string
	for _, dir := range dirs {
		if insideSkill(dir, kept) {
			continue
		}
		kept = append(kept, dir)
	}
	skills := make([]canonSkill, 0, len(kept))
	for _, dir := range kept {
		abs := filepath.Join(home, filepath.FromSlash(dir))
		skills = append(skills, canonSkill{name: filepath.Base(abs), rel: dir, abs: abs})
	}
	return skills, files, nil
}

func insideSkill(dir string, outers []string) bool {
	for _, outer := range outers {
		if strings.HasPrefix(dir, outer+"/") {
			return true
		}
	}
	return false
}

func sameNameExplanation(skills []canonSkill) string {
	grouped := map[string][]string{}
	order := make([]string, 0)
	for _, skill := range skills {
		key := nameKey(skill.name)
		if _, ok := grouped[key]; !ok {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], skill.rel)
	}
	var clashes []string
	for _, key := range order {
		paths := grouped[key]
		if len(paths) < 2 {
			continue
		}
		slices.Sort(paths)
		quoted := make([]string, len(paths))
		for i, item := range paths {
			quoted[i] = "«" + item + "»"
		}
		clashes = append(clashes, strings.Join(quoted, " и "))
	}
	if len(clashes) == 0 {
		return ""
	}
	return sameFolderNameMessage(clashes)
}

func linkSecret(links []canonLink, files []File) string {
	for _, file := range files {
		if !insideLink(file.Path, links) || !recordWouldRefuse(file.Path, file.Body) {
			continue
		}
		return SecretExplanation(file.Path, file.Body)
	}
	return ""
}

// recordWouldRefuse is true only when a snapshot would stop, not when it would omit the file.
func recordWouldRefuse(path, body string) bool {
	file, ok := portableFile(File{Path: path, Body: body})
	return ok && SecretExplanation(file.Path, file.Body) != ""
}

// HasAssignedSecret matches a named secret assignment. It does not match a bare key such as sk-.
func HasAssignedSecret(body string) bool {
	return secretValue.MatchString(body)
}

// HasSecret reports a body запись treats as a secret. The value itself is not returned.
func HasSecret(body string) bool {
	return HasAssignedSecret(body) || knownSecret.MatchString(body)
}

// SecretExplanation is the пояснение запись shows for a secret in a named file.
func SecretExplanation(path, body string) string {
	if !HasSecret(body) && !(IsMCP(path) && authValue.MatchString(body)) {
		return ""
	}
	return "Секрет в\u00a0файле " + path + "."
}

func insideLink(rel string, links []canonLink) bool {
	for _, link := range links {
		if rel == link.rel || strings.HasPrefix(rel, link.rel+"/") {
			return true
		}
	}
	return false
}

func agentCatalog(root, agentName string) (skillCatalog, error) {
	name := strings.TrimSpace(agentName)
	receiver, ok := FindReceiver(name)
	if !ok {
		return skillCatalog{}, fmt.Errorf("ИИ-агента «%s» нет в\u00a0каталоге ИИ-агентов.", name)
	}
	dir, have := receiver.GlobalDir(root)
	if !have {
		return skillCatalog{}, fmt.Errorf("У\u00a0ИИ-агента «%s» нет глобального каталога.", receiver.Display)
	}
	return skillCatalog{dir: dir, display: receiver.Display, slug: receiver.Slug, receiver: receiver}, nil
}

// occupiedNames lists the link places the agent holds as its own folder, file or foreign link.
func occupiedNames(links []canonLink) (map[string]bool, error) {
	occupied := map[string]bool{}
	for _, link := range links {
		_, err := os.Lstat(link.dest)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if target, linked := linkTarget(link.dest); linked && samePlace(target, link.abs) {
			continue
		}
		occupied[catalogKey(link.dest)] = true
	}
	return occupied, nil
}

// snapshotAgent saves the agent's portable settings before a canon link takes an occupied name.
// It returns the snapshot directory and the home it mirrors: occupied items move there whole,
// private files included.
func snapshotAgent(root, account string, cat skillCatalog) (string, string, error) {
	home := agentHome(root, cat.slug)
	if !within(home, filepath.Dir(cat.dir)) {
		home = filepath.Dir(cat.dir)
	}
	files, err := readSnapshot(root, home)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}
	dir, err := writeSnapshot(root, account, agentDirName(filepath.Base(home)), portableOnly(files))
	if err != nil {
		return "", "", err
	}
	return dir, home, nil
}

func shareCanon(canon, snapshot, home, display string, plan connectPlan, occupied map[string]bool) (catalogUndo, error) {
	var undo catalogUndo
	failed := func() error {
		return fmt.Errorf("Не удалось показать канон ИИ-агенту «%s».", display)
	}
	wanted := map[string]bool{}
	for _, link := range plan.links {
		wanted[catalogKey(link.dest)] = true
	}
	stale, err := staleLinks(canon, plan.dirs, plan.files, wanted)
	if err != nil {
		return undo, failed()
	}
	for _, link := range stale {
		target, _ := linkTarget(link)
		undo.removed = append(undo.removed, savedLink{path: link, target: target})
		if err := os.Remove(link); err != nil {
			return undo, failed()
		}
	}
	for _, link := range plan.links {
		if !within(canon, link.abs) || !oneSegment(filepath.Base(link.abs)) {
			return undo, errors.New("Элемент лежит вне канона.")
		}
		created, err := ensureCatalog(filepath.Dir(link.dest))
		undo.createdDirs = append(undo.createdDirs, created...)
		if err != nil {
			return undo, failed()
		}
		if occupied[catalogKey(link.dest)] {
			rel, err := filepath.Rel(home, link.dest)
			if err != nil || !filepath.IsLocal(rel) {
				return undo, failed()
			}
			kept := filepath.Join(snapshot, rel)
			if err := os.RemoveAll(kept); err != nil {
				return undo, failed()
			}
			if err := os.MkdirAll(filepath.Dir(kept), 0o755); err != nil {
				return undo, failed()
			}
			// shortcut: rename fails across volumes (agent home on another drive than ~/.agentsync); copy then if users hit it.
			if err := os.Rename(link.dest, kept); err != nil {
				return undo, failed()
			}
			undo.moved = append(undo.moved, savedLink{path: link.dest, target: kept})
		}
		if target, ok := linkTarget(link.dest); ok && samePlace(target, link.abs) {
			continue
		}
		if err := os.Symlink(link.abs, link.dest); err != nil {
			return undo, failed()
		}
		undo.added = append(undo.added, link.dest)
	}
	return undo, nil
}

// staleLinks lists the links into the canon in the agent's places that wanted does not hold.
func staleLinks(canon string, dirs, files []string, wanted map[string]bool) ([]string, error) {
	candidates := slices.Clone(files)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			candidates = append(candidates, filepath.Join(dir, entry.Name()))
		}
	}
	var stale []string
	for _, path := range candidates {
		if target, ok := linkTarget(path); ok && within(canon, target) && !wanted[catalogKey(path)] {
			stale = append(stale, path)
		}
	}
	return stale, nil
}

func ensureCatalog(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			return nil, &os.PathError{Op: "mkdir", Path: dir, Err: errors.New("not a directory")}
		}
		return nil, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	created := missingDirs(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return created, err
	}
	return created, nil
}

func rollbackCatalog(undo catalogUndo) error {
	var errs []error
	for _, link := range undo.added {
		if err := os.Remove(link); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	for _, item := range undo.moved {
		if err := os.Rename(item.target, item.path); err != nil {
			errs = append(errs, err)
		}
	}
	for _, link := range undo.removed {
		if err := os.Symlink(link.target, link.path); err != nil && !os.IsExist(err) {
			errs = append(errs, err)
		}
	}
	// Deepest first: a parent goes only once the folders made under it are gone.
	slices.SortFunc(undo.createdDirs, func(a, b string) int { return len(b) - len(a) })
	for _, dir := range undo.createdDirs {
		_ = os.Remove(dir)
	}
	return errors.Join(errs...)
}

func linkTarget(link string) (string, bool) {
	target, err := os.Readlink(link)
	if err != nil || target == "" {
		return "", false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(link), target)
	}
	return filepath.Clean(target), true
}

func within(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if samePlace(root, target) {
		return true
	}
	prefix := root + string(os.PathSeparator)
	if len(target) <= len(prefix) {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(target[:len(prefix)], prefix)
	}
	return strings.HasPrefix(target, prefix)
}

func samePlace(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func nameKey(name string) string {
	return foldKey(name)
}

func catalogKey(dir string) string {
	return foldKey(filepath.Clean(dir))
}

func foldKey(value string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(value)
	}
	return value
}

func oneSegment(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`)
}
