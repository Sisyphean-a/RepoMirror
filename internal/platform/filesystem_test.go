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

func TestFileSizeFromRootReturnsNestedFileSize(t *testing.T) {
	root := t.TempDir()
	content := strings.Repeat("abcdef0123456789", 128)
	writeTempFile(t, root, filepath.Join("nested", "file.txt"), content)

	size, err := NewOSFileSystem().FileSizeFromRoot(root, "nested/file.txt")
	if err != nil {
		t.Fatalf("file size from root failed: %v", err)
	}
	if size != int64(len(content)) {
		t.Fatalf("unexpected file size: got %d want %d", size, len(content))
	}
}

func TestCompareFileFromRootsMatchesRelativeFile(t *testing.T) {
	leftRoot := t.TempDir()
	rightRoot := t.TempDir()
	content := strings.Repeat("abcdef0123456789", 256)
	writeTempFile(t, leftRoot, filepath.Join("nested", "file.txt"), content)
	writeTempFile(t, rightRoot, filepath.Join("nested", "file.txt"), content)

	comparison, err := NewOSFileSystem().CompareFileFromRoots(leftRoot, rightRoot, "nested/file.txt")
	if err != nil {
		t.Fatalf("compare file from roots failed: %v", err)
	}
	if !comparison.Equal {
		t.Fatalf("expected files to be equal")
	}
	if comparison.LeftSize != int64(len(content)) {
		t.Fatalf("unexpected left size: got %d want %d", comparison.LeftSize, len(content))
	}
}

func TestCopyFileFromRootsCopiesNestedFile(t *testing.T) {
	sourceRoot := t.TempDir()
	targetRoot := t.TempDir()
	content := strings.Repeat("abcdef0123456789", 256)
	writeTempFile(t, sourceRoot, filepath.Join("nested", "file.txt"), content)

	if err := NewOSFileSystem().CopyFileFromRoots(sourceRoot, targetRoot, "nested/file.txt"); err != nil {
		t.Fatalf("copy file from roots failed: %v", err)
	}
	assertPath := filepath.Join(targetRoot, "nested", "file.txt")
	data, err := os.ReadFile(assertPath)
	if err != nil {
		t.Fatalf("read copied file failed: %v", err)
	}
	if string(data) != content {
		t.Fatalf("unexpected copied content")
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

func TestRemoveEmptyParentsRemovesNestedEmptyDirs(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir nested dirs failed: %v", err)
	}

	start := filepath.Join(targetDir, "file.txt")
	if err := NewOSFileSystem().RemoveEmptyParents(root, start); err != nil {
		t.Fatalf("remove empty parents failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "a")); !os.IsNotExist(err) {
		t.Fatalf("expected nested parent dirs to be removed, got err=%v", err)
	}
}

func TestRemoveEmptyParentsStopsAtNonEmptyParent(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir nested dirs failed: %v", err)
	}
	writeTempFile(t, root, filepath.Join("a", "keep.txt"), "keep")

	start := filepath.Join(targetDir, "file.txt")
	if err := NewOSFileSystem().RemoveEmptyParents(root, start); err != nil {
		t.Fatalf("remove empty parents failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "a")); err != nil {
		t.Fatalf("expected non-empty parent to remain, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "b")); !os.IsNotExist(err) {
		t.Fatalf("expected empty nested parent to be removed, got err=%v", err)
	}
}

func BenchmarkRemoveEmptyParents(b *testing.B) {
	fsys := NewOSFileSystem()
	b.ReportAllocs()
	for iteration := 0; iteration < b.N; iteration++ {
		root := b.TempDir()
		targetDir := filepath.Join(root, "a", "b", "c", "d")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			b.Fatalf("mkdir nested dirs failed: %v", err)
		}
		start := filepath.Join(targetDir, "file.txt")
		b.StartTimer()
		err := fsys.RemoveEmptyParents(root, start)
		b.StopTimer()
		if err != nil {
			b.Fatalf("remove empty parents failed: %v", err)
		}
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
