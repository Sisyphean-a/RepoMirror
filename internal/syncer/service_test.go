package syncer

import (
	"os"
	"path/filepath"
	"testing"

	"RepoMirror/internal/diff"
	"RepoMirror/internal/gitops"
	"RepoMirror/internal/platform"
	"RepoMirror/internal/testutil"
)

func TestSyncCopiesAndDeletes(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	testutil.InitRepo(t, source)
	testutil.InitRepo(t, target)

	testutil.WriteFile(t, source, ".gitignore", "ignored/\n")
	testutil.WriteFile(t, source, "tracked.txt", "source tracked")
	testutil.WriteFile(t, source, "notes.md", "source note")
	testutil.WriteFile(t, source, "ignored/skip.txt", "ignored")
	testutil.StageAll(t, source)

	testutil.WriteFile(t, target, ".gitignore", "ignored/\n")
	testutil.WriteFile(t, target, "tracked.txt", "target tracked")
	testutil.WriteFile(t, target, "old.txt", "remove me")
	testutil.WriteFile(t, target, "ignored/keep.txt", "keep me")

	fileSystem := platform.NewOSFileSystem()
	differ := diff.NewService(fileSystem, gitops.NewService(gitops.NewExecRunner()))
	service := NewService(fileSystem, differ)
	if err := service.Sync(Request{SourceRoot: source, TargetRoot: target}); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	assertFileContent(t, filepath.Join(target, "tracked.txt"), "source tracked")
	assertFileContent(t, filepath.Join(target, "notes.md"), "source note")
	if _, err := os.Stat(filepath.Join(target, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old.txt should be removed, got err=%v", err)
	}
	assertFileContent(t, filepath.Join(target, "ignored", "keep.txt"), "keep me")
}

func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != expected {
		t.Fatalf("unexpected content for %s: %s", path, data)
	}
}
