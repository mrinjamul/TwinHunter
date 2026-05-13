package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mrinjamul/twinhunter/models"
)

// DeleteFile removes a single file.
func DeleteFile(path string) error {
	return os.Remove(path)
}

// replaceWithLink is the common crash-safe implementation for replacing dup
// with a link to keep. It renames dup to a temp backup first, then creates
// the link at dup. If the link fails, the backup is restored.
func replaceWithLink(keep, dup string, linkFn func(string, string) error) error {
	if keep == dup {
		return errors.New("cannot link file to itself")
	}

	absKeep, err := filepath.Abs(keep)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for %s: %w", keep, err)
	}

	dir := filepath.Dir(dup)
	safeName := filepath.Join(dir, ".twinhunter.safe."+filepath.Base(dup))

	// Step 1: rename dup to safe backup
	if err := os.Rename(dup, safeName); err != nil {
		return fmt.Errorf("failed to back up %s: %w", dup, err)
	}

	// Step 2: create the link at the original dup path
	if err := linkFn(absKeep, dup); err != nil {
		os.Rename(safeName, dup)
		return fmt.Errorf("failed to create link: %w", err)
	}

	// Step 3: remove the safe backup
	if err := os.Remove(safeName); err != nil {
		return fmt.Errorf("link created but failed to remove backup %s: %w", safeName, err)
	}

	return nil
}

// ReplaceWithHardLink replaces dup with a hard link to keep.
// Crash-safe: renames dup first, only removes backup after link succeeds.
func ReplaceWithHardLink(keep, dup string) error {
	return replaceWithLink(keep, dup, os.Link)
}

// ReplaceWithSoftLink replaces dup with a symbolic link to keep.
// Crash-safe: renames dup first, only removes backup after link succeeds.
func ReplaceWithSoftLink(keep, dup string) error {
	return replaceWithLink(keep, dup, os.Symlink)
}

// MoveToBackup moves a duplicate file to a backup directory, preserving structure.
func MoveToBackup(dup, backupDir string) error {
	rel, err := filepath.Rel("/", dup)
	if err != nil {
		return fmt.Errorf("failed to compute relative path for %s: %w", dup, err)
	}
	dest := filepath.Join(backupDir, rel)
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	err = os.Rename(dup, dest)
	if err != nil {
		return copyThenRemove(dup, dest)
	}
	return nil
}

func copyThenRemove(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source for copy: %w", err)
	}

	destFile, err := os.Create(dest)
	if err != nil {
		srcFile.Close()
		return fmt.Errorf("failed to create destination: %w", err)
	}

	if _, err := io.Copy(destFile, srcFile); err != nil {
		srcFile.Close()
		destFile.Close()
		os.Remove(dest)
		return fmt.Errorf("failed to copy: %w", err)
	}

	srcFile.Close()
	destFile.Close()

	if err := os.Remove(src); err != nil {
		os.Remove(dest)
		return fmt.Errorf("failed to remove source after copy: %w", err)
	}

	return nil
}

// Action represents a file operation to perform.
type Action int

const (
	ActionNone Action = iota
	ActionDelete
	ActionHardLink
	ActionSoftLink
	ActionBackup
)

// ApplyAction performs the given action on a duplicate file.
func ApplyAction(action Action, keep models.FileInfo, dup models.FileInfo, backupDir string) error {
	switch action {
	case ActionDelete:
		return DeleteFile(dup.Path)
	case ActionHardLink:
		return ReplaceWithHardLink(keep.Path, dup.Path)
	case ActionSoftLink:
		return ReplaceWithSoftLink(keep.Path, dup.Path)
	case ActionBackup:
		return MoveToBackup(dup.Path, backupDir)
	default:
		return nil
	}
}
