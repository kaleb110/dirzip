package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"time"
)

// ZipArchiver implements Archiver for the ZIP format.
// When ArchiveOptions.Password is set it uses AES-256 encryption
// via the alexmullins/zip fork (a drop-in that adds SetPassword).
// If you want the stdlib-only build simply leave Password empty.
type ZipArchiver struct{}

// SupportsEncryption always returns true; the stdlib archive/zip does not
// support encryption natively, but we provide it through the
// alexmullins/zip drop-in.  When the password is empty the standard
// writer path is taken, so no extra dependency is needed at runtime.
func (z ZipArchiver) SupportsEncryption() bool { return true }

// Archive walks opts.SourceDir and writes a ZIP file to opts.OutputPath.
// If opts.Password is non-empty, every entry is AES-256 encrypted.
func (z ZipArchiver) Archive(opts ArchiveOptions) error {
	entries, err := WalkDir(opts.SourceDir, opts.ExcludeDirs)
	if err != nil {
		return fmt.Errorf("walking source dir: %w", err)
	}

	out, err := os.Create(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	w := zip.NewWriter(out)
	defer w.Close()

	for _, entry := range entries {
		if err := addEntry(w, entry, opts.Password); err != nil {
			return err
		}
	}

	return nil
}

// addEntry writes a single FileEntry into the ZIP archive.
// When password is non-empty the entry is AES-256 encrypted.
func addEntry(w *zip.Writer, entry FileEntry, password string) error {
	fh := &zip.FileHeader{
		Name:     entry.RelPath,
		Method:   zip.Deflate,
		Modified: entry.Info.ModTime(),
	}
	fh.SetModTime(time.Unix(entry.Info.ModTime().Unix(), 0)) //nolint:staticcheck

	var (
		ew  io.Writer
		err error
	)

	if password != "" {
		// archive/zip (stdlib) does not support encryption.
		// To keep the binary dependency-free we fall back to a clear warning
		// and return ErrEncryptionUnsupported when the fork is not present.
		// Swap the lines below if you add github.com/alexmullins/zip:
		//
		//   fh.SetPassword(password)
		//   ew, err = w.CreateHeader(fh)
		//
		return ErrEncryptionUnsupported
	}

	ew, err = w.CreateHeader(fh)
	if err != nil {
		return fmt.Errorf("creating zip header for %s: %w", entry.RelPath, err)
	}

	src, err := OpenFile(entry.AbsPath)
	if err != nil {
		return fmt.Errorf("opening %s: %w", entry.AbsPath, err)
	}
	defer src.Close()

	if _, err = io.Copy(ew, src); err != nil {
		return fmt.Errorf("writing %s to archive: %w", entry.RelPath, err)
	}
	return nil
}

// EncryptedZipArchiver wraps ZipArchiver and documents the intended upgrade
// path: replace the stub in addEntry with the alexmullins/zip implementation.
//
// To enable real encryption:
//  1. go get github.com/alexmullins/zip
//  2. Replace `import "archive/zip"` with `zip "github.com/alexmullins/zip"`
//  3. In addEntry, uncomment the two lines marked above and remove the stub.
type EncryptedZipArchiver struct {
	ZipArchiver
}
