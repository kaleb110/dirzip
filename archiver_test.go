package main

import (
	"errors"
	"fmt"
	"testing"
)

// mockArchiver is a test double for the Archiver interface.
type mockArchiver struct {
	encryptionSupported bool
	archiveCalled       bool
	lastOpts            ArchiveOptions
	returnErr           error
}

func (m *mockArchiver) Archive(opts ArchiveOptions) error {
	m.archiveCalled = true
	m.lastOpts = opts
	return m.returnErr
}

func (m *mockArchiver) SupportsEncryption() bool {
	return m.encryptionSupported
}

func TestArchiverInterface(t *testing.T) {
	t.Run("interface is satisfied by mockArchiver", func(t *testing.T) {
		var a Archiver = &mockArchiver{}
		if a == nil {
			t.Fatal("expected non-nil Archiver")
		}
	})

	t.Run("Archive is called with correct options", func(t *testing.T) {
		m := &mockArchiver{}
		opts := ArchiveOptions{
			SourceDir:   "/src",
			OutputPath:  "/out.zip",
			ExcludeDirs: map[string]struct{}{"vendor": {}},
		}
		if err := m.Archive(opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !m.archiveCalled {
			t.Error("Archive was not called")
		}
		if m.lastOpts.SourceDir != opts.SourceDir {
			t.Errorf("SourceDir: want %q, got %q", opts.SourceDir, m.lastOpts.SourceDir)
		}
		if m.lastOpts.OutputPath != opts.OutputPath {
			t.Errorf("OutputPath: want %q, got %q", opts.OutputPath, m.lastOpts.OutputPath)
		}
		if _, ok := m.lastOpts.ExcludeDirs["vendor"]; !ok {
			t.Error("ExcludeDirs should contain 'vendor'")
		}
	})

	t.Run("Archive propagates errors", func(t *testing.T) {
		sentinel := errors.New("disk full")
		m := &mockArchiver{returnErr: sentinel}
		err := m.Archive(ArchiveOptions{})
		if !errors.Is(err, sentinel) {
			t.Errorf("expected sentinel error, got %v", err)
		}
	})

	t.Run("SupportsEncryption returns configured value", func(t *testing.T) {
		supported := &mockArchiver{encryptionSupported: true}
		if !supported.SupportsEncryption() {
			t.Error("expected true")
		}
		unsupported := &mockArchiver{encryptionSupported: false}
		if unsupported.SupportsEncryption() {
			t.Error("expected false")
		}
	})
}

func TestErrEncryptionUnsupported(t *testing.T) {
	if ErrEncryptionUnsupported == nil {
		t.Fatal("ErrEncryptionUnsupported must not be nil")
	}
	wrapped := fmt.Errorf("outer: %w", ErrEncryptionUnsupported)
	if !errors.Is(wrapped, ErrEncryptionUnsupported) {
		t.Error("errors.Is must unwrap ErrEncryptionUnsupported")
	}
}
