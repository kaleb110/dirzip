package main

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// buildZipTree creates a minimal source tree for ZIP tests.
func buildZipTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"hello.txt":          "hello world",
		"sub/world.txt":      "sub content",
		"ignored/secret.txt": "should be excluded",
	}
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return root
}

func TestZipArchiver_SupportsEncryption(t *testing.T) {
	z := ZipArchiver{}
	// The interface promises encryption support; the actual runtime behaviour
	// depends on whether the encrypted fork is wired in (see zip.go comments).
	if !z.SupportsEncryption() {
		t.Error("ZipArchiver.SupportsEncryption() should return true")
	}
}

func TestZipArchiver_Archive_Basic(t *testing.T) {
	src := buildZipTree(t)
	out := filepath.Join(t.TempDir(), "out.zip")

	z := ZipArchiver{}
	err := z.Archive(ArchiveOptions{
		SourceDir:  src,
		OutputPath: out,
	})
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}

	// Open and inspect the resulting archive.
	r, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("zip.OpenReader: %v", err)
	}
	defer r.Close()

	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)

	want := []string{"hello.txt", "ignored/secret.txt", "sub/world.txt"}
	if !equalSlices(names, want) {
		t.Errorf("archive contents:\ngot  %v\nwant %v", names, want)
	}
}

func TestZipArchiver_Archive_WithExcludes(t *testing.T) {
	src := buildZipTree(t)
	out := filepath.Join(t.TempDir(), "out.zip")

	z := ZipArchiver{}
	err := z.Archive(ArchiveOptions{
		SourceDir:   src,
		OutputPath:  out,
		ExcludeDirs: map[string]struct{}{"ignored": {}},
	})
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}

	r, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("zip.OpenReader: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "ignored/secret.txt" {
			t.Errorf("excluded file %q found in archive", f.Name)
		}
	}
}

func TestZipArchiver_Archive_FileContents(t *testing.T) {
	src := buildZipTree(t)
	out := filepath.Join(t.TempDir(), "out.zip")

	z := ZipArchiver{}
	if err := z.Archive(ArchiveOptions{SourceDir: src, OutputPath: out}); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	r, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("zip.OpenReader: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name != "hello.txt" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry: %v", err)
		}
		defer rc.Close()
		buf := make([]byte, 32)
		n, _ := rc.Read(buf)
		if string(buf[:n]) != "hello world" {
			t.Errorf("content: got %q, want %q", buf[:n], "hello world")
		}
	}
}

func TestZipArchiver_Archive_InvalidSource(t *testing.T) {
	z := ZipArchiver{}
	err := z.Archive(ArchiveOptions{
		SourceDir:  "/nonexistent/path/123abc",
		OutputPath: filepath.Join(t.TempDir(), "out.zip"),
	})
	if err == nil {
		t.Error("expected error for invalid source dir, got nil")
	}
}

func TestZipArchiver_Archive_InvalidOutputPath(t *testing.T) {
	src := buildZipTree(t)
	z := ZipArchiver{}
	err := z.Archive(ArchiveOptions{
		SourceDir:  src,
		OutputPath: "/nonexistent/deep/path/out.zip",
	})
	if err == nil {
		t.Error("expected error for invalid output path, got nil")
	}
}

func TestZipArchiver_Archive_WithPassword_ReturnsUnsupported(t *testing.T) {
	src := buildZipTree(t)
	out := filepath.Join(t.TempDir(), "out.zip")

	z := ZipArchiver{}
	err := z.Archive(ArchiveOptions{
		SourceDir:  src,
		OutputPath: out,
		Password:   "s3cr3t",
	})

	// With the stdlib-only build, encryption is not available.
	// This test documents the current behaviour and must be updated when
	// the alexmullins/zip fork is wired in.
	if err == nil {
		t.Fatal("expected ErrEncryptionUnsupported but got nil")
	}
	if !errors.Is(err, ErrEncryptionUnsupported) {
		t.Errorf("expected ErrEncryptionUnsupported, got: %v", err)
	}
}

func TestZipArchiver_Archive_EmptyDir(t *testing.T) {
	src := t.TempDir() // empty
	out := filepath.Join(t.TempDir(), "out.zip")

	z := ZipArchiver{}
	if err := z.Archive(ArchiveOptions{SourceDir: src, OutputPath: out}); err != nil {
		t.Fatalf("Archive empty dir: %v", err)
	}

	r, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("zip.OpenReader: %v", err)
	}
	defer r.Close()

	if len(r.File) != 0 {
		t.Errorf("expected 0 entries in empty archive, got %d", len(r.File))
	}
}

func TestSelectArchiver(t *testing.T) {
	a := selectArchiver("")
	if a == nil {
		t.Fatal("selectArchiver returned nil")
	}
	if _, ok := a.(ZipArchiver); !ok {
		t.Errorf("expected ZipArchiver, got %T", a)
	}
}
