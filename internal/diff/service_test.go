package diff

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
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
		Total:    3,
		Added:    1,
		Modified: 1,
		Deleted:  1,
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
		firstStarted:  make(chan struct{}),
		secondStarted: make(chan struct{}),
	}
	inspector := &stubGitInspector{
		filesByRoot: map[string][]string{
			"source": {"a.txt", "b.txt"},
			"target": {"a.txt", "b.txt"},
		},
	}
	service := NewService(fileSystem, inspector)

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
	inspector := &stubGitInspector{
		filesByRoot: map[string][]string{
			"source": nil,
			"target": {"old.txt"},
		},
	}
	service := NewService(&concurrentCompareFileSystem{}, inspector)

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

func TestCalculateChecksIgnoreRulesAgainstSourceFilesOnly(t *testing.T) {
	inspector := &stubGitInspector{
		filesByRoot: map[string][]string{
			"source": {"added.txt"},
			"target": {"old.txt"},
		},
	}
	service := NewService(&concurrentCompareFileSystem{}, inspector)

	_, err := service.Calculate(Request{SourceRoot: "source", TargetRoot: "target"})
	if err != nil {
		t.Fatalf("calculate failed: %v", err)
	}

	if len(inspector.ignoredCalls) != 1 {
		t.Fatalf("expected 1 ignore check, got %d", len(inspector.ignoredCalls))
	}
	call := inspector.ignoredCalls[0]
	if call.root != "target" {
		t.Fatalf("unexpected ignore root: %s", call.root)
	}
	if !reflect.DeepEqual(call.paths, []string{"added.txt"}) {
		t.Fatalf("unexpected ignore paths: got %v want %v", call.paths, []string{"added.txt"})
	}
}

type ignoredCall struct {
	root  string
	paths []string
}

type stubGitInspector struct {
	filesByRoot  map[string][]string
	ignored      map[string]struct{}
	ignoredCalls []ignoredCall
}

func (stub stubGitInspector) ListSyncableSourcePathsFromRoot(root string) ([]string, error) {
	return append([]string(nil), stub.filesByRoot[root]...), nil
}

func (stub *stubGitInspector) IgnoredPathSetFromRootSorted(root string, paths []string) (map[string]struct{}, error) {
	stub.ignoredCalls = append(stub.ignoredCalls, ignoredCall{
		root:  root,
		paths: append([]string(nil), paths...),
	})
	if stub.ignored == nil {
		return map[string]struct{}{}, nil
	}
	return stub.ignored, nil
}

type concurrentCompareFileSystem struct {
	firstStarted  chan struct{}
	secondStarted chan struct{}
}

func (fs *concurrentCompareFileSystem) ListRegularFiles(string) ([]string, error) {
	panic("unexpected ListRegularFiles call")
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

func (fs *concurrentCompareFileSystem) CompareFileFromRoots(leftRoot string, _ string, relPath string) (platform.FileComparison, error) {
	return fs.CompareFile(relPath, "")
}

func (fs *concurrentCompareFileSystem) FilesEqual(string, string) (bool, error) {
	panic("unexpected FilesEqual call")
}

func (fs *concurrentCompareFileSystem) FileSize(string) (int64, error) {
	return 5, nil
}

func (fs *concurrentCompareFileSystem) FileSizeFromRoot(_ string, _ string) (int64, error) {
	return 5, nil
}

func (fs *concurrentCompareFileSystem) CopyFileFromRoots(string, string, string) error {
	panic("unexpected CopyFileFromRoots call")
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

func (fs *concurrentCompareFileSystem) RemoveFromRoot(string, string) error {
	panic("unexpected RemoveFromRoot call")
}

func (fs *concurrentCompareFileSystem) RemoveEmptyParents(string, string) error {
	panic("unexpected RemoveEmptyParents call")
}

func (fs *concurrentCompareFileSystem) RemoveEmptyParentsFromRoot(string, string) error {
	panic("unexpected RemoveEmptyParentsFromRoot call")
}

func BenchmarkCalculateLargeDiff(b *testing.B) {
	const fileCount = 4000

	sourceFiles := make([]string, 0, fileCount)
	targetFiles := make([]string, 0, fileCount)
	for index := 0; index < fileCount; index++ {
		name := fmt.Sprintf("dir/file-%04d.txt", index)
		sourceFiles = append(sourceFiles, name)
		if index%4 != 0 {
			targetFiles = append(targetFiles, name)
		}
	}

	inspector := &stubGitInspector{
		filesByRoot: map[string][]string{
			"source": sourceFiles,
			"target": targetFiles,
		},
	}
	fileSystem := &benchmarkCompareFileSystem{}
	service := NewService(fileSystem, inspector)

	b.ReportAllocs()
	b.ResetTimer()

	for iteration := 0; iteration < b.N; iteration++ {
		result, err := service.Calculate(Request{SourceRoot: "source", TargetRoot: "target"})
		if err != nil {
			b.Fatalf("calculate failed: %v", err)
		}
		if len(result.Entries) == 0 {
			b.Fatal("expected diff entries")
		}
	}
}

func BenchmarkMergedPathCount(b *testing.B) {
	const fileCount = 4000
	sourceFiles := make([]string, 0, fileCount)
	targetFiles := make([]string, 0, fileCount)
	for index := 0; index < fileCount; index++ {
		name := fmt.Sprintf("dir/file-%04d.txt", index)
		sourceFiles = append(sourceFiles, name)
		if index%4 != 0 {
			targetFiles = append(targetFiles, name)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		count := mergedPathCount(sourceFiles, targetFiles)
		if count == 0 {
			b.Fatal("expected merged path count")
		}
	}
}

type benchmarkCompareFileSystem struct{}

func (fs *benchmarkCompareFileSystem) ListRegularFiles(string) ([]string, error) {
	panic("unexpected ListRegularFiles call")
}

func (fs *benchmarkCompareFileSystem) Exists(string) (bool, error) {
	panic("unexpected Exists call")
}

func (fs *benchmarkCompareFileSystem) CompareFile(left string, _ string) (platform.FileComparison, error) {
	size := int64(len(left))
	return platform.FileComparison{
		Equal:    strings.HasSuffix(left, "0.txt"),
		LeftSize: size,
	}, nil
}

func (fs *benchmarkCompareFileSystem) CompareFileFromRoots(leftRoot string, _ string, relPath string) (platform.FileComparison, error) {
	size := int64(rootedPathLen(leftRoot, relPath))
	return platform.FileComparison{
		Equal:    strings.HasSuffix(relPath, "0.txt"),
		LeftSize: size,
	}, nil
}

func (fs *benchmarkCompareFileSystem) FilesEqual(string, string) (bool, error) {
	panic("unexpected FilesEqual call")
}

func (fs *benchmarkCompareFileSystem) FileSize(path string) (int64, error) {
	return int64(len(path)), nil
}

func (fs *benchmarkCompareFileSystem) FileSizeFromRoot(root string, relPath string) (int64, error) {
	return int64(rootedPathLen(root, relPath)), nil
}

func (fs *benchmarkCompareFileSystem) CopyFileFromRoots(string, string, string) error {
	panic("unexpected CopyFileFromRoots call")
}

func rootedPathLen(root string, relPath string) int {
	if root == "" {
		return len(relPath)
	}
	length := len(root) + len(relPath)
	last := root[len(root)-1]
	if last != '/' && last != '\\' {
		length++
	}
	return length
}

func (fs *benchmarkCompareFileSystem) CopyFile(string, string) error {
	panic("unexpected CopyFile call")
}

func (fs *benchmarkCompareFileSystem) EnsureDirectory(string) error {
	panic("unexpected EnsureDirectory call")
}

func (fs *benchmarkCompareFileSystem) Remove(string) error {
	panic("unexpected Remove call")
}

func (fs *benchmarkCompareFileSystem) RemoveFromRoot(string, string) error {
	panic("unexpected RemoveFromRoot call")
}

func (fs *benchmarkCompareFileSystem) RemoveEmptyParents(string, string) error {
	panic("unexpected RemoveEmptyParents call")
}

func (fs *benchmarkCompareFileSystem) RemoveEmptyParentsFromRoot(string, string) error {
	panic("unexpected RemoveEmptyParentsFromRoot call")
}
