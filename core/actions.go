package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mrinjamul/twinhunter/models"
)

// DeleteFile removes a single file.
func DeleteFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}
	return os.Remove(path)
}

// ReplaceWithHardLink replaces dup with a hard link to keep.
func ReplaceWithHardLink(keep, dup string) error {
	if keep == dup {
		return errors.New("cannot link file to itself")
	}
	if err := os.Remove(dup); err != nil {
		return err
	}
	return os.Link(keep, dup)
}

// ReplaceWithSoftLink replaces dup with a symbolic link to keep.
func ReplaceWithSoftLink(keep, dup string) error {
	if keep == dup {
		return errors.New("cannot link file to itself")
	}
	if err := os.Remove(dup); err != nil {
		return err
	}
	absKeep, err := filepath.Abs(keep)
	if err != nil {
		absKeep = keep
	}
	return os.Symlink(absKeep, dup)
}

// MoveToBackup moves a duplicate file to a backup directory, preserving structure.
func MoveToBackup(dup, backupDir string) error {
	rel, err := filepath.Rel("/", dup)
	if err != nil {
		rel = filepath.Base(dup)
	}
	dest := filepath.Join(backupDir, rel)
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return os.Rename(dup, dest)
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
