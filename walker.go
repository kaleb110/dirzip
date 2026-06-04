package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrEncryptionUnsupported is returned by Archiver implementations that do
// not support encryption when a non-empty Password is supplied.
var ErrEncryptionUnsupported = errors.New("this archive format does not support encryption")

// WalkDir traverses root, skipping any directory whose base name appears in
// excludeDirs, and returns a slice of FileEntry for every regular file found.
func WalkDir(root string, excludeDirs map[string]struct{}) ([]FileEntry, error) {
	var entries []FileEntry

	root = filepath.Clean(root)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk error at %s: %w", path, err)
		}

		// Compute the base name once for both directory pruning and path building.
		base := d.Name()

		if d.IsDir() {
			// Always allow the root itself through.
			if path == root {
				return nil
			}
			if _, excluded := excludeDirs[base]; excluded {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip anything that is not a regular file (symlinks, devices, …).
		if !d.Type().IsRegular() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		// Build a clean, forward-slash relative path for use inside the archive.
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("rel path for %s: %w", path, err)
		}
		rel = filepath.ToSlash(rel)

		entries = append(entries, FileEntry{
			RelPath: rel,
			Info:    info,
			AbsPath: path,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// ParseExcludes converts a comma-separated string (or repeated flag values
// collected into a slice) into the map format expected by WalkDir.
func ParseExcludes(dirs []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, item := range dirs {
		// Support both --exclude=a,b and multiple --exclude flags.
		for _, part := range strings.Split(item, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out[part] = struct{}{}
			}
		}
	}
	return out
}

// OpenFile is a thin wrapper so tests can replace it.
var OpenFile = func(path string) (*os.File, error) { return os.Open(path) }
