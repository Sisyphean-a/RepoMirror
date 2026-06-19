package diff

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"RepoMirror/internal/gitops"
	"RepoMirror/internal/model"
	"RepoMirror/internal/platform"
	"RepoMirror/internal/testutil"
)

func TestCalculateRespectsIgnoreRules(t *testing.T) {
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

	service := NewService(platform.NewOSFileSystem(), gitops.NewService(gitops.NewExecRunner()))
	result, err := service.Calculate(Request{SourceRoot: source, TargetRoot: target})
	if err != nil {
		t.Fatalf("calculate failed: %v", err)
	}

	got := map[string]model.DiffKind{}
	for _, entry := range result.Entries {
		got[entry.Path] = entry.Kind
	}
	assertDiffKind(t, got, "tracked.txt", model.DiffKindModified)
	assertDiffKind(t, got, "notes.md", model.DiffKindAdded)
	assertDiffKind(t, got, "old.txt", model.DiffKindDeleted)
	if _, exists := got["ignored/keep.txt"]; exists {
		t.Fatalf("ignored target file should not enter diff list")
	}
	if result.Summary.Total != 3 {
		t.Fatalf("expected total summary to be 3, got %+v", result.Summary)
	}
	expectedSummary := model.DiffSummary{
		Total:     3,
		Added:     1,
		Modified:  1,
		Deleted:   1,
		Protected: 0,
	}
	if result.Summary != expectedSummary {
		t.Fatalf("unexpected summary: got %+v want %+v", result.Summary, expectedSummary)
	}
	expectedOrder := []string{"notes.md", "old.txt", "tracked.txt"}
	for index, path := range expectedOrder {
		if result.Entries[index].Path != path {
			t.Fatalf("unexpected diff order at %d: got %s want %s", index, result.Entries[index].Path, path)
		}
	}
}

func assertDiffKind(t *testing.T, entries map[string]model.DiffKind, path string, expected model.DiffKind) {
	t.Helper()
	actual, exists := entries[path]
	if !exists {
		t.Fatalf("missing diff entry for %s", path)
	}
	if actual != expected {
		t.Fatalf("unexpected diff kind for %s: %s", path, actual)
	}
}

func TestCalculateKeepsOrderWhenComparisonsFinishOutOfOrder(t *testing.T) {
	fileSystem := &concurrentCompareFileSystem{
		targetFiles:   []string{"a.txt", "b.txt"},
		firstStarted:  make(chan struct{}),
		secondStarted: make(chan struct{}),
	}
	service := NewService(fileSystem, stubGitInspector{sourceFiles: []string{"a.txt", "b.txt"}})

	result, err := service.Calculate(Request{SourceRoot: "source", TargetRoot: "target"})
	if err != nil {
		t.Fatalf("calculate failed: %v", err)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result.Entries))
	}
	if result.Entries[0].Path != "a.txt" || result.Entries[1].Path != "b.txt" {
		t.Fatalf("unexpected entry order: %+v", result.Entries)
	}
	if result.Entries[0].Kind != model.DiffKindModified || result.Entries[1].Kind != model.DiffKindModified {
		t.Fatalf("unexpected diff kinds: %+v", result.Entries)
	}
}

func TestCalculateHandlesDeletedOnlyTargets(t *testing.T) {
	service := NewService(&concurrentCompareFileSystem{targetFiles: []string{"old.txt"}}, stubGitInspector{})

	result, err := service.Calculate(Request{SourceRoot: "source", TargetRoot: "target"})
	if err != nil {
		t.Fatalf("calculate failed: %v", err)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}
	if result.Entries[0].Path != "old.txt" || result.Entries[0].Kind != model.DiffKindDeleted {
		t.Fatalf("unexpected deleted-only entry: %+v", result.Entries)
	}
	expectedSummary := model.DiffSummary{Total: 1, Deleted: 1}
	if result.Summary != expectedSummary {
		t.Fatalf("unexpected deleted-only summary: got %+v want %+v", result.Summary, expectedSummary)
	}
}

type stubGitInspector struct {
	sourceFiles []string
	ignored     map[string]string
}

func (stub stubGitInspector) ListSyncableSourcePathsFromRoot(string) ([]string, error) {
	return stub.sourceFiles, nil
}

func (stub stubGitInspector) IgnoredPathsFromRoot(string, ...[]string) (map[string]string, error) {
	if stub.ignored == nil {
		return map[string]string{}, nil
	}
	return stub.ignored, nil
}

type concurrentCompareFileSystem struct {
	targetFiles   []string
	firstStarted  chan struct{}
	secondStarted chan struct{}
}

func (fs *concurrentCompareFileSystem) ListRegularFiles(string) ([]string, error) {
	return fs.targetFiles, nil
}

func (fs *concurrentCompareFileSystem) Exists(string) (bool, error) {
	panic("unexpected Exists call")
}

func (fs *concurrentCompareFileSystem) CompareFile(left string, _ string) (platform.FileComparison, error) {
	switch filepath.Base(left) {
	case "a.txt":
		close(fs.firstStarted)
		select {
		case <-fs.secondStarted:
			return platform.FileComparison{LeftSize: 5}, nil
		case <-time.After(time.Second):
			return platform.FileComparison{}, fmt.Errorf("second comparison never started")
		}
	case "b.txt":
		close(fs.secondStarted)
		return platform.FileComparison{LeftSize: 5}, nil
	default:
		return platform.FileComparison{LeftSize: 5}, nil
	}
}

func (fs *concurrentCompareFileSystem) FilesEqual(string, string) (bool, error) {
	panic("unexpected FilesEqual call")
}

func (fs *concurrentCompareFileSystem) FileSize(string) (int64, error) {
	panic("unexpected FileSize call")
}

func (fs *concurrentCompareFileSystem) CopyFile(string, string) error {
	panic("unexpected CopyFile call")
}

func (fs *concurrentCompareFileSystem) EnsureDirectory(string) error {
	panic("unexpected EnsureDirectory call")
}

func (fs *concurrentCompareFileSystem) Remove(string) error {
	panic("unexpected Remove call")
}

func (fs *concurrentCompareFileSystem) RemoveEmptyParents(string, string) error {
	panic("unexpected RemoveEmptyParents call")
}
