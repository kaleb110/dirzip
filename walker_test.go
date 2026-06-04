package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// buildTree creates a temporary directory tree and returns its root.
// Layout:
//
//	root/
//	  a.txt
//	  sub/
//	    b.txt
//	  node_modules/
//	    c.txt
//	  .git/
//	    d.txt
//	  nested/
//	    vendor/
//	      e.txt
//	    f.txt
func buildTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	files := []string{
		"a.txt",
		filepath.Join("sub", "b.txt"),
		filepath.Join("node_modules", "c.txt"),
		filepath.Join(".git", "d.txt"),
		filepath.Join("nested", "vendor", "e.txt"),
		filepath.Join("nested", "f.txt"),
	}

	for _, f := range files {
		full := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte("content of "+f), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return root
}

func relPaths(entries []FileEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.RelPath
	}
	sort.Strings(out)
	return out
}

func TestWalkDir_NoExcludes(t *testing.T) {
	root := buildTree(t)
	entries, err := WalkDir(root, nil)
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}

	got := relPaths(entries)
	want := []string{
		".git/d.txt",
		"a.txt",
		"nested/f.txt",
		"nested/vendor/e.txt",
		"node_modules/c.txt",
		"sub/b.txt",
	}
	if !equalSlices(got, want) {
		t.Errorf("got  %v\nwant %v", got, want)
	}
}

func TestWalkDir_ExcludeNodeModules(t *testing.T) {
	root := buildTree(t)
	excl := map[string]struct{}{"node_modules": {}}
	entries, err := WalkDir(root, excl)
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}

	got := relPaths(entries)
	for _, p := range got {
		if containsSegment(p, "node_modules") {
			t.Errorf("node_modules should be excluded, but found %q", p)
		}
	}
}

func TestWalkDir_ExcludeMultiple(t *testing.T) {
	root := buildTree(t)
	excl := map[string]struct{}{
		"node_modules": {},
		".git":         {},
		"vendor":       {},
	}
	entries, err := WalkDir(root, excl)
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}

	got := relPaths(entries)
	want := []string{
		"a.txt",
		"nested/f.txt",
		"sub/b.txt",
	}
	if !equalSlices(got, want) {
		t.Errorf("got  %v\nwant %v", got, want)
	}
}

func TestWalkDir_NonExistentRoot(t *testing.T) {
	_, err := WalkDir("/this/does/not/exist/ever", nil)
	if err == nil {
		t.Error("expected error for non-existent root, got nil")
	}
}

func TestWalkDir_EntryFields(t *testing.T) {
	root := buildTree(t)
	entries, err := WalkDir(root, nil)
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	for _, e := range entries {
		if e.RelPath == "" {
			t.Error("RelPath must not be empty")
		}
		if e.AbsPath == "" {
			t.Error("AbsPath must not be empty")
		}
		if e.Info == nil {
			t.Errorf("Info must not be nil for %s", e.RelPath)
		}
	}
}

// ---------------------------------------------------------------------------
// ParseExcludes tests
// ---------------------------------------------------------------------------

func TestParseExcludes_Empty(t *testing.T) {
	m := ParseExcludes(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestParseExcludes_SingleValues(t *testing.T) {
	m := ParseExcludes([]string{"node_modules", ".git"})
	for _, want := range []string{"node_modules", ".git"} {
		if _, ok := m[want]; !ok {
			t.Errorf("missing key %q", want)
		}
	}
}

func TestParseExcludes_CommaSeparated(t *testing.T) {
	m := ParseExcludes([]string{"node_modules,.git,vendor"})
	for _, want := range []string{"node_modules", ".git", "vendor"} {
		if _, ok := m[want]; !ok {
			t.Errorf("missing key %q", want)
		}
	}
}

func TestParseExcludes_Mixed(t *testing.T) {
	m := ParseExcludes([]string{"node_modules,.git", "vendor"})
	for _, want := range []string{"node_modules", ".git", "vendor"} {
		if _, ok := m[want]; !ok {
			t.Errorf("missing key %q", want)
		}
	}
}

func TestParseExcludes_Whitespace(t *testing.T) {
	m := ParseExcludes([]string{" node_modules , .git "})
	for _, want := range []string{"node_modules", ".git"} {
		if _, ok := m[want]; !ok {
			t.Errorf("missing key %q after trimming whitespace", want)
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsSegment(path, segment string) bool {
	for _, part := range filepath.SplitList(filepath.FromSlash(path)) {
		if part == segment {
			return true
		}
	}
	// Also check slash-split for forward-slash paths
	parts := filepath.ToSlash(path)
	for _, p := range splitPath(parts) {
		if p == segment {
			return true
		}
	}
	return false
}

func splitPath(p string) []string {
	var parts []string
	for {
		dir, file := filepath.Split(p)
		if file != "" {
			parts = append(parts, file)
		}
		if dir == "" || dir == p {
			break
		}
		p = filepath.Clean(dir)
	}
	return parts
}
