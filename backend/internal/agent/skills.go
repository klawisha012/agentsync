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

// SkillCopy names a publication version and the skill folders to place into other homes.
type SkillCopy struct {
	Author  string
	Source  string
	Version int
	Skills  []string
	Targets []string
}

// CopySkills reads one publication version and adds the named skill folders
// to each target home. Other files in those homes stay in place.
func CopySkills(ctx context.Context, server, root, account, token, cookie string, copy SkillCopy) error {
	if copy.Version < 1 {
		return errors.New("Назовите номер версии.")
	}
	skills := compactNames(copy.Skills)
	if len(skills) == 0 {
		return errors.New("Назовите навык.")
	}
	targets := compactNames(copy.Targets)
	if len(targets) == 0 {
		return errors.New("Назовите ИИ-агента, куда положить навыки.")
	}
	payload, err := fetchPublication(ctx, server, copy.Author, copy.Source, token, cookie, copy.Version)
	if err != nil {
		return err
	}
	files, err := chosenSkillFiles(payload.Files, skills)
	if err != nil {
		return err
	}
	owner := strings.TrimSpace(account)
	if owner == "" {
		owner = payload.Author
	}
	placed := make([]skillPlace, 0, len(targets))
	for _, target := range targets {
		place, err := placeSkills(root, owner, target, files)
		if err != nil {
			if undoErr := rollbackPlaces(placed); undoErr != nil {
				return errors.Join(err, undoErr)
			}
			return err
		}
		placed = append(placed, place)
	}
	return nil
}

type skillFolder struct {
	path  string
	files []File
}

func chosenSkillFiles(all []File, wanted []string) ([]File, error) {
	folders := skillFolders(all)
	chosen := map[string]File{}
	seen := map[string]struct{}{}
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
			for _, file := range folder.files {
				chosen[file.Path] = file
			}
		}
	}
	if len(chosen) == 0 {
		return nil, errors.New("В выбранных навыках нет файлов.")
	}
	out := make([]File, 0, len(chosen))
	for _, file := range chosen {
		out = append(out, file)
	}
	slices.SortFunc(out, func(a, b File) int {
		return strings.Compare(a.Path, b.Path)
	})
	return out, nil
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

type skillPlace struct {
	home    string
	snap    string
	written []string
	prior   map[string]string
	had     map[string]bool
}

func placeSkills(root, account, agentName string, next []File) (skillPlace, error) {
	home := skillHome(root, agentName)
	current, err := readTree(home)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return skillPlace{}, err
	}
	want := map[string]struct{}{}
	for _, file := range next {
		want[file.Path] = struct{}{}
	}
	portable := make([]File, 0)
	prior := map[string]string{}
	had := map[string]bool{}
	for _, file := range current {
		rel := filepath.ToSlash(file.Path)
		if _, ok := want[rel]; ok {
			prior[rel] = file.Body
			had[rel] = true
		}
		if isPortable(file.Path, file.Body) {
			portable = append(portable, file)
		}
	}
	snap, err := writeSnapshot(root, account, agentName, portable)
	if err != nil {
		return skillPlace{}, err
	}
	place := skillPlace{home: home, snap: snap, prior: prior, had: had}
	for _, file := range next {
		if err := writeFile(home, file); err != nil {
			if undoErr := place.rollback(); undoErr != nil {
				return skillPlace{}, errors.Join(err, undoErr)
			}
			return skillPlace{}, err
		}
		place.written = append(place.written, file.Path)
	}
	return place, nil
}

func (place skillPlace) rollback() error {
	var errs []error
	for _, rel := range place.written {
		if place.had[rel] {
			if err := writeFile(place.home, File{Path: rel, Body: place.prior[rel]}); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		err := os.Remove(filepath.Join(place.home, filepath.FromSlash(rel)))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	if place.snap != "" {
		if err := os.RemoveAll(place.snap); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func rollbackPlaces(places []skillPlace) error {
	var errs []error
	for i := len(places) - 1; i >= 0; i-- {
		if err := places[i].rollback(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func skillHome(root, name string) string {
	name = strings.TrimSpace(name)
	home := agentHome(root, name)
	literal := filepath.Join(root, name)
	if home != literal {
		return home
	}
	info, err := os.Stat(literal)
	if err == nil && info.IsDir() {
		return literal
	}
	return filepath.Join(root, "."+strings.ToLower(name))
}
