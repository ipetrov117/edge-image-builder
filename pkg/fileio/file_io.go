package fileio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	// ExecutablePerms are Linux permissions (rwxr--r--) for executable files (scripts, binaries, etc.)
	ExecutablePerms os.FileMode = 0o744
	// NonExecutablePerms are Linux permissions (rw-r--r--) for non-executable files (configs, RPMs, etc.)
	NonExecutablePerms os.FileMode = 0o644
)

func CopyFile(src string, dest string, perms os.FileMode) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer func() {
		_ = destFile.Close()
	}()

	if err = destFile.Chmod(perms); err != nil {
		return fmt.Errorf("adjusting permissions: %w", err)
	}

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return nil
}

func CopyDir(src, dest string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("checking for src dir at %s: %w", src, err)
	}

	if err := os.MkdirAll(dest, srcInfo.Mode()); err != nil {
		return fmt.Errorf("creating dest dir at %s: %w", dest, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("reading source dir: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			t, e := entry.Info()
			if e != nil {
				return fmt.Errorf("retrieving info for file %s: %w", entry.Name(), err)
			}

			if err := CopyFile(srcPath, destPath, t.Mode()); err != nil {
				return err
			}
		}
	}

	return nil
}
