package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// SkillCopy names a publication version and the skill folders to place into receivers.
type SkillCopy struct {
	Author  string
	Source  string
	Version int
	Skills  []string
	Targets []string
	Place   string
}

// CopySkills reads one publication version and adds the named skill folders
// to each receiver's skills directory. Other skills stay in place. The store is not written.
func CopySkills(ctx context.Context, server, root, account, token, cookie string, copy SkillCopy) error {
	if copy.Version < 1 {
		return errors.New("Назовите номер версии.")
	}
	place, err := onePlace(copy.Place)
	if err != nil {
		return err
	}
	skills := compactNames(copy.Skills)
	if len(skills) == 0 {
		return errors.New("Назовите навык.")
	}
	targets := compactNames(copy.Targets)
	if len(targets) == 0 {
		return errors.New("Назовите приёмника, куда положить навыки.")
	}
	dirs, err := receiverDirs(root, place, targets)
	if err != nil {
		return err
	}
	payload, err := fetchPublication(ctx, server, copy.Author, copy.Source, token, cookie, copy.Version)
	if err != nil {
		return err
	}
	folders, err := chosenSkillFolders(payload.Files, skills)
	if err != nil {
		return err
	}
	folders, err = uniqueSkillBases(folders)
	if err != nil {
		return err
	}
	installed := relocateSkills(folders)
	var written []backedFile
	for _, dir := range dirs {
		next, err := writeInstalled(dir, installed)
		written = append(written, next...)
		if err != nil {
			if undoErr := restoreFiles(written); undoErr != nil {
				return errors.Join(err, undoErr)
			}
			return err
		}
	}
	return nil
}

type skillFolder struct {
	path  string
	files []File
}

func chosenSkillFolders(all []File, wanted []string) ([]skillFolder, error) {
	folders := skillFolders(all)
	seen := map[string]struct{}{}
	out := make([]skillFolder, 0, len(wanted))
	for _, name := range wanted {
		hits, err := matchFolders(folders, name)
		if err != nil {
			return nil, err
		}
		for _, folder := range hits {
			if _, ok := seen[folder.path]; ok {
				continue
			}
			seen[folder.path] = struct{}{}
			out = append(out, folder)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("В выбранных навыках нет файлов.")
	}
	return out, nil
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

func relocateSkills(folders []skillFolder) []File {
	out := make([]File, 0)
	for _, folder := range folders {
		base := path.Base(folder.path)
		for _, file := range folder.files {
			rel := base
			if file.Path != folder.path {
				rel = base + "/" + strings.TrimPrefix(file.Path, folder.path+"/")
			}
			out = append(out, File{Path: rel, Body: file.Body})
		}
	}
	slices.SortFunc(out, func(a, b File) int {
		return strings.Compare(a.Path, b.Path)
	})
	return out
}

func onePlace(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "global":
		return "global", nil
	case "project":
		return "project", nil
	default:
		return "", errors.New("Выберите глобальный каталог или проект.")
	}
}

func receiverDirs(root, place string, targets []string) ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		receiver, ok := FindReceiver(target)
		if !ok {
			return nil, fmt.Errorf("Приёмник «%s» не найден.", target)
		}
		var dir string
		var have bool
		if place == "project" {
			dir, have = receiver.ProjectDir(cwd)
		} else {
			dir, have = receiver.GlobalDir(root)
		}
		if !have {
			if place == "project" {
				return nil, fmt.Errorf("У приёмника «%s» нет каталога проекта.", receiver.Display)
			}
			return nil, fmt.Errorf("У приёмника «%s» нет глобального каталога.", receiver.Display)
		}
		key := filepath.Clean(dir)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, dir)
	}
	return out, nil
}

type backedFile struct {
	path        string
	existed     bool
	prior       []byte
	createdDirs []string
}

func writeInstalled(dir string, files []File) ([]backedFile, error) {
	written := make([]backedFile, 0, len(files))
	for _, file := range files {
		target := filepath.Join(dir, filepath.FromSlash(file.Path))
		prior := backedFile{path: target, createdDirs: missingDirs(filepath.Dir(target))}
		raw, err := os.ReadFile(target)
		if err == nil {
			prior.existed = true
			prior.prior = raw
		} else if !errors.Is(err, os.ErrNotExist) {
			return written, err
		}
		written = append(written, prior)
		if err := writeFile(dir, file); err != nil {
			return written, err
		}
	}
	return written, nil
}

func restoreFiles(files []backedFile) error {
	var errs []error
	for i := len(files) - 1; i >= 0; i-- {
		file := files[i]
		if file.existed {
			if err := os.MkdirAll(filepath.Dir(file.path), 0o755); err != nil {
				errs = append(errs, err)
				continue
			}
			if err := os.WriteFile(file.path, file.prior, 0o644); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if err := os.Remove(file.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
		for _, dir := range file.createdDirs {
			if err := os.Remove(dir); err != nil && !errors.Is(err, os.ErrNotExist) {
				break
			}
		}
	}
	return errors.Join(errs...)
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
		rel, ok := cleanRel(file.Path)
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

func matchFolders(folders []skillFolder, wanted string) ([]skillFolder, error) {
	raw := strings.TrimSpace(wanted)
	bare := !strings.ContainsAny(raw, `/\`)
	if bare {
		var hits []skillFolder
		for _, folder := range folders {
			if strings.EqualFold(path.Base(folder.path), raw) {
				hits = append(hits, folder)
			}
		}
		return oneFolder(raw, hits)
	}
	want, ok := cleanRel(raw)
	if !ok {
		return nil, fmt.Errorf("В этой версии нет навыка «%s».", raw)
	}
	var exact, folded []skillFolder
	for _, folder := range folders {
		if folder.path == want {
			exact = append(exact, folder)
		} else if strings.EqualFold(folder.path, want) {
			folded = append(folded, folder)
		}
	}
	if len(exact) > 0 {
		return oneFolder(raw, exact)
	}
	return oneFolder(raw, folded)
}

func oneFolder(name string, hits []skillFolder) ([]skillFolder, error) {
	if len(hits) == 0 {
		return nil, fmt.Errorf("В этой версии нет навыка «%s».", name)
	}
	if len(hits) > 1 {
		return nil, fmt.Errorf("Навык «%s» встречается больше одного раза. Укажите путь.", name)
	}
	return hits, nil
}

func cleanRel(value string) (string, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || strings.Contains(value, "\x00") {
		return "", false
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", false
	}
	return cleaned, true
}

func compactNames(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}
