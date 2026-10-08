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
)

type canonSkill struct {
	name string
	rel  string
	abs  string
}

type skillCatalog struct {
	dir     string
	display string
}

type savedLink struct {
	path   string
	target string
}

type catalogUndo struct {
	added       []string
	removed     []savedLink
	createdDirs []string
}

// Fanout shows every outermost skill from one agent's home in each named
// receiver's global catalog. The receiver name and the canon folder are one record.
func Fanout(root, agentName string, targets []string) error {
	unlock, err := lockRoot(root)
	if err != nil {
		return err
	}
	defer unlock()
	home := agentHome(root, agentName)
	skills, files, err := canonSkills(home)
	if err != nil {
		return err
	}
	if len(skills) == 0 {
		return errors.New("Нет навыков.")
	}
	if len(compactNames(targets)) == 0 {
		return errors.New("Назовите приёмника.")
	}
	if msg := sameNameExplanation(skills); msg != "" {
		return errors.New(msg)
	}
	if msg := skillSecret(skills, files); msg != "" {
		return errors.New(msg)
	}
	catalogs, err := globalCatalogs(root, targets)
	if err != nil {
		return err
	}
	if err := occupiedName(catalogs, skills); err != nil {
		return err
	}
	return shareAll(home, catalogs, skills)
}

// Unfanout removes receiver names whose record belongs to this agent's home.
// The home itself stays. A copied folder that is not that record stays.
func Unfanout(root, agentName string, targets []string) error {
	unlock, err := lockRoot(root)
	if err != nil {
		return err
	}
	defer unlock()
	home := agentHome(root, agentName)
	catalogs, err := unfanoutCatalogs(root, targets)
	if err != nil {
		return err
	}
	for _, cat := range catalogs {
		if err := unfanoutCatalog(home, cat.dir); err != nil {
			return err
		}
	}
	return nil
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
		if dir == "." || privatePath(file.Path) || !portablePath(file.Path) {
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

// recordWouldRefuse is true only when запись would stop, not when it would omit the file.
func recordWouldRefuse(path, body string) bool {
	if privatePath(path) || !portablePath(path) {
		return false
	}
	if isHook(path) && absolutePath.MatchString(body) {
		return false
	}
	if isMCP(path) && machineLike(body) {
		return false
	}
	return HasSecret(body)
}

func machineLike(body string) bool {
	if absolutePath.MatchString(body) || strings.Contains(strings.ToLower(body), "authorization") {
		return true
	}
	return HasAssignedSecret(body)
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
	if !HasSecret(body) {
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

func unfanoutCatalogs(root string, targets []string) ([]skillCatalog, error) {
	return receiverCatalogs(root, targets, false)
}

func globalCatalogs(root string, targets []string) ([]skillCatalog, error) {
	return receiverCatalogs(root, targets, true)
}

func receiverCatalogs(root string, targets []string, requireGlobal bool) ([]skillCatalog, error) {
	targets = compactNames(targets)
	if len(targets) == 0 {
		return nil, errors.New("Назовите приёмника.")
	}
	seen := map[string]struct{}{}
	out := make([]skillCatalog, 0, len(targets))
	for _, target := range targets {
		receiver, ok := FindReceiver(target)
		if !ok {
			return nil, fmt.Errorf("Приёмник «%s» не найден.", target)
		}
		dir, have := receiver.GlobalDir(root)
		if !have {
			if requireGlobal {
				return nil, fmt.Errorf("У приёмника «%s» нет глобального каталога.", receiver.Display)
			}
			continue
		}
		key := catalogKey(dir)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, skillCatalog{dir: dir, display: receiver.Display})
	}
	return out, nil
}

func occupiedName(catalogs []skillCatalog, skills []canonSkill) error {
	for _, cat := range catalogs {
		for _, skill := range skills {
			dest := filepath.Join(cat.dir, skill.name)
			_, err := os.Lstat(dest)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			target, linked := linkTarget(dest)
			if linked && samePlace(target, skill.abs) {
				continue
			}
			return fmt.Errorf("У приёмника «%s» навык «%s» уже занят.", cat.display, skill.name)
		}
	}
	return nil
}

func shareAll(home string, catalogs []skillCatalog, skills []canonSkill) error {
	undos := make([]catalogUndo, 0, len(catalogs))
	for _, cat := range catalogs {
		undo, err := shareCatalog(home, cat, skills)
		undos = append(undos, undo)
		if err != nil {
			if undoErr := rollbackCatalogs(undos); undoErr != nil {
				return errors.Join(err, undoErr)
			}
			return err
		}
	}
	return nil
}

func shareCatalog(home string, cat skillCatalog, skills []canonSkill) (catalogUndo, error) {
	var undo catalogUndo
	dir := cat.dir
	failed := func() error {
		return fmt.Errorf("Не удалось показать навыки приёмнику «%s».", cat.display)
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
			return undo, errors.New("Навык лежит вне домашней папки.")
		}
		dest := filepath.Join(dir, skill.name)
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

func unfanoutCatalog(home, dir string) error {
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

func rollbackCatalogs(undos []catalogUndo) error {
	var errs []error
	for i := len(undos) - 1; i >= 0; i-- {
		undo := undos[i]
		for _, link := range undo.added {
			if err := os.Remove(link); err != nil && !errors.Is(err, os.ErrNotExist) {
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
