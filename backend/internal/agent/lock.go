package agent

import (
	"os"
	"path/filepath"
)

func withRootLock(root string, fn func() error) error {
	unlock, err := lockRoot(root)
	if err != nil {
		return err
	}
	defer unlock()
	return fn()
}

func lockRoot(root string) (func(), error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return lockFileAt(filepath.Join(root, ".agentsync.lock"))
}

func lockDir(dir string, perm os.FileMode) (func(), error) {
	if err := os.MkdirAll(dir, perm); err != nil {
		return nil, err
	}
	return lockFileAt(filepath.Join(dir, "lock"))
}

func lockFileAt(path string) (func(), error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockFile(file); err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() {
		_ = unlockFile(file)
		_ = file.Close()
	}, nil
}
