package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesEqualMatchesLargeContent(t *testing.T) {
	root := t.TempDir()
	left := writeTempFile(t, root, "left.txt", strings.Repeat("abcdef0123456789", 8192))
	right := writeTempFile(t, root, "right.txt", strings.Repeat("abcdef0123456789", 8192))

	equal, err := NewOSFileSystem().FilesEqual(left, right)
	if err != nil {
		t.Fatalf("files equal failed: %v", err)
	}
	if !equal {
		t.Fatalf("expected files to be equal")
	}
}

func TestFilesEqualDetectsDifferentContent(t *testing.T) {
	root := t.TempDir()
	left := writeTempFile(t, root, "left.txt", strings.Repeat("abcdef0123456789", 4096))
	right := writeTempFile(t, root, "right.txt", strings.Repeat("abcdef012345678x", 4096))

	equal, err := NewOSFileSystem().FilesEqual(left, right)
	if err != nil {
		t.Fatalf("files equal failed: %v", err)
	}
	if equal {
		t.Fatalf("expected files to be different")
	}
}

func TestCompareFileReturnsSourceSizeForDifferentContent(t *testing.T) {
	root := t.TempDir()
	leftContent := strings.Repeat("abcdef0123456789", 4096)
	left := writeTempFile(t, root, "left.txt", leftContent)
	right := writeTempFile(t, root, "right.txt", strings.Repeat("abcdef012345678x", 4096))

	comparison, err := NewOSFileSystem().CompareFile(left, right)
	if err != nil {
		t.Fatalf("compare file failed: %v", err)
	}
	if comparison.Equal {
		t.Fatalf("expected files to be different")
	}
	if comparison.LeftSize != int64(len(leftContent)) {
		t.Fatalf("unexpected left size: got %d want %d", comparison.LeftSize, len(leftContent))
	}
}

func TestCopyFileCopiesLargeContent(t *testing.T) {
	root := t.TempDir()
	source := writeTempFile(t, root, "source.txt", strings.Repeat("abcdef0123456789", 8192))
	target := filepath.Join(root, "nested", "target.txt")

	if err := NewOSFileSystem().CopyFile(source, target); err != nil {
		t.Fatalf("copy file failed: %v", err)
	}

	equal, err := NewOSFileSystem().FilesEqual(source, target)
	if err != nil {
		t.Fatalf("files equal failed: %v", err)
	}
	if !equal {
		t.Fatalf("expected copied file to match source")
	}
}

func TestListRegularFilesReturnsLexicalOrder(t *testing.T) {
	root := t.TempDir()
	writeTempFile(t, root, "z-last.txt", "z")
	writeTempFile(t, root, filepath.Join("a-dir", "b.txt"), "b")
	writeTempFile(t, root, filepath.Join("a-dir", "a.txt"), "a")
	writeTempFile(t, root, filepath.Join("m-dir", "c.txt"), "c")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git failed: %v", err)
	}
	writeTempFile(t, filepath.Join(root, ".git"), "ignored.txt", "ignored")

	files, err := NewOSFileSystem().ListRegularFiles(root)
	if err != nil {
		t.Fatalf("list regular files failed: %v", err)
	}

	expected := []string{
		"a-dir/a.txt",
		"a-dir/b.txt",
		"m-dir/c.txt",
		"z-last.txt",
	}
	if strings.Join(files, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("unexpected lexical order: got %v want %v", files, expected)
	}
}

func writeTempFile(t *testing.T, root string, name string, content string) string {
	t.Helper()
	path := root + string(filepath.Separator) + name
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir temp file dir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}
	return path
}
