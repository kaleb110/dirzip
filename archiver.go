package main

import "io/fs"

// ArchiveOptions holds configuration passed to any Archiver implementation.
type ArchiveOptions struct {
	// SourceDir is the root directory to archive.
	SourceDir string

	// OutputPath is the destination file path (e.g. "output.zip").
	OutputPath string

	// ExcludeDirs is the set of directory names to skip entirely.
	// Matching is done against each path component, not the full path,
	// so "node_modules" will match ./a/node_modules and ./b/node_modules.
	ExcludeDirs map[string]struct{}

	// Password, when non-empty, enables encryption for formats that support it.
	// Implementations that do not support encryption MUST return ErrEncryptionUnsupported
	// when this field is set.
	Password string
}

// Archiver is the extension point for different archive formats.
// Implement this interface to add support for tar.gz, 7z, etc.
type Archiver interface {
	// Archive walks sourceDir and writes the archive to the path specified in opts.
	Archive(opts ArchiveOptions) error

	// SupportsEncryption reports whether this implementation supports password protection.
	SupportsEncryption() bool
}

// FileEntry represents a single file collected by the walker.
type FileEntry struct {
	// RelPath is the path relative to the source root, using forward slashes.
	RelPath string
	// Info is the file's fs.FileInfo.
	Info fs.FileInfo
	// AbsPath is the absolute path on disk, used for opening the file.
	AbsPath string
}
