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
	dir     string
	display string
	slug    string
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

// Connect shows every outermost canon skill in the named AI agent's global skills
// directory by a link. A name the agent already holds goes to its snapshot first.
func Connect(root, account, agentName string) error {
	unlock, err := lockRoot(root)
	if err != nil {
		return err
	}
	defer unlock()
	canon := canonDir(root)
	skills, files, err := canonSkills(canon)
	if err != nil {
		return err
	}
	if len(skills) == 0 {
		return errors.New("В\u00a0каноне нечего показать.")
	}
	cat, err := agentCatalog(root, agentName)
	if err != nil {
		return err
	}
	if insideCanon(root, cat.dir) {
		return nil
	}
	if msg := sameNameExplanation(skills); msg != "" {
		return errors.New(msg)
	}
	if msg := skillSecret(skills, files); msg != "" {
		return errors.New(msg)
	}
	occupied, err := occupiedNames(canon, cat.dir, skills)
	if err != nil {
		return err
	}
	snapshot, keep := "", ""
	if len(occupied) > 0 {
		if snapshot, keep, err = snapshotAgent(root, account, cat); err != nil {
			return err
		}
	}
	undo, err := shareCatalog(canon, keep, cat, skills, occupied)
	if err != nil {
		undoErr := rollbackCatalog(undo)
		// The snapshot holds the moved folders until every one is back in place.
		if undoErr == nil && snapshot != "" {
			_ = os.RemoveAll(snapshot)
		}
		return errors.Join(err, undoErr)
	}
	return nil
}

// Disconnect removes the links to the canon from the AI agent's global skills directory.
// The canon stays. A folder that is not a link to the canon stays.
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
	return unlinkCanon(canonDir(root), dir)
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

func skillSecret(skills []canonSkill, files []File) string {
	for _, file := range files {
		if !insideShownSkill(file.Path, skills) || !recordWouldRefuse(file.Path, file.Body) {
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

func insideShownSkill(rel string, skills []canonSkill) bool {
	for _, skill := range skills {
		if rel == skill.rel || strings.HasPrefix(rel, skill.rel+"/") {
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
	return skillCatalog{dir: dir, display: receiver.Display, slug: receiver.Slug}, nil
}

// occupiedNames lists canon skill names the agent holds as its own folder, file or foreign link.
func occupiedNames(canon, dir string, skills []canonSkill) (map[string]bool, error) {
	occupied := map[string]bool{}
	for _, skill := range skills {
		dest := filepath.Join(dir, skill.name)
		_, err := os.Lstat(dest)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if target, linked := linkTarget(dest); linked && samePlace(target, skill.abs) {
			continue
		}
		occupied[nameKey(skill.name)] = true
	}
	return occupied, nil
}

// snapshotAgent saves the agent's portable settings before a canon link takes an occupied name.
// keep is the snapshot's skills directory: occupied folders move there whole, private files included.
func snapshotAgent(root, account string, cat skillCatalog) (string, string, error) {
	home := agentHome(root, cat.slug)
	if !within(home, cat.dir) {
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
	rel, err := filepath.Rel(home, cat.dir)
	if err != nil {
		return "", "", err
	}
	return dir, filepath.Join(dir, rel), nil
}

func shareCatalog(home, keep string, cat skillCatalog, skills []canonSkill, occupied map[string]bool) (catalogUndo, error) {
	var undo catalogUndo
	dir := cat.dir
	failed := func() error {
		return fmt.Errorf("Не удалось показать навыки канона ИИ-агенту «%s».", cat.display)
	}
	created, err := ensureCatalog(dir)
	undo.createdDirs = created
	if err != nil {
		return undo, failed()
	}
	byName := map[string]canonSkill{}
	for _, skill := range skills {
		byName[nameKey(skill.name)] = skill
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return undo, failed()
	}
	for _, entry := range entries {
		link := filepath.Join(dir, entry.Name())
		target, ok := linkTarget(link)
		if !ok || !within(home, target) {
			continue
		}
		if _, current := byName[nameKey(entry.Name())]; current {
			continue
		}
		undo.removed = append(undo.removed, savedLink{path: link, target: target})
		if err := os.Remove(link); err != nil {
			return undo, failed()
		}
	}
	for _, skill := range skills {
		if !within(home, skill.abs) || !oneSegment(skill.name) {
			return undo, errors.New("Навык лежит вне канона.")
		}
		dest := filepath.Join(dir, skill.name)
		if occupied[nameKey(skill.name)] {
			kept := filepath.Join(keep, skill.name)
			if err := os.RemoveAll(kept); err != nil {
				return undo, failed()
			}
			if err := os.MkdirAll(keep, 0o755); err != nil {
				return undo, failed()
			}
			// shortcut: rename fails across volumes (skills dir on another drive than ~/.agentsync); copy then if users hit it.
			if err := os.Rename(dest, kept); err != nil {
				return undo, failed()
			}
			undo.moved = append(undo.moved, savedLink{path: dest, target: kept})
		}
		if target, ok := linkTarget(dest); ok && samePlace(target, skill.abs) {
			continue
		}
		if err := os.Symlink(skill.abs, dest); err != nil {
			return undo, failed()
		}
		undo.added = append(undo.added, dest)
	}
	return undo, nil
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

func unlinkCanon(home, dir string) error {
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		link := filepath.Join(dir, entry.Name())
		target, ok := linkTarget(link)
		if !ok || !within(home, target) {
			continue
		}
		if err := os.Remove(link); err != nil {
			return err
		}
	}
	return nil
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
	for _, dir := range undo.createdDirs {
		if err := os.Remove(dir); err != nil && !errors.Is(err, os.ErrNotExist) {
			break
		}
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
