package syncer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"RepoMirror/internal/diff"
	"RepoMirror/internal/gitops"
	"RepoMirror/internal/model"
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

func TestApplyCopiesRunsCopyWorkInParallel(t *testing.T) {
	fs := &copyTrackingFileSystem{
		firstStarted:  make(chan struct{}),
		secondStarted: make(chan struct{}),
	}
	service := NewService(fs, nil)

	err := service.applyCopies(
		Request{SourceRoot: "source", TargetRoot: "target"},
		[]model.DiffEntry{
			{Path: "a.txt", Kind: model.DiffKindAdded},
			{Path: "b.txt", Kind: model.DiffKindModified},
		},
		2,
	)
	if err != nil {
		t.Fatalf("apply copies failed: %v", err)
	}
	if fs.copyCount != 2 {
		t.Fatalf("expected 2 copy calls, got %d", fs.copyCount)
	}
}

func TestApplyDeletesRunsDeleteWorkInParallel(t *testing.T) {
	fs := &deleteTrackingFileSystem{
		firstStarted:  make(chan struct{}),
		secondStarted: make(chan struct{}),
	}
	service := NewService(fs, nil)

	err := service.applyDeletes(
		Request{TargetRoot: "target"},
		[]model.DiffEntry{
			{Path: "a.txt", Kind: model.DiffKindDeleted},
			{Path: "b.txt", Kind: model.DiffKindDeleted},
		},
		2,
	)
	if err != nil {
		t.Fatalf("apply deletes failed: %v", err)
	}
	if fs.removeCount != 2 {
		t.Fatalf("expected 2 remove calls, got %d", fs.removeCount)
	}
	if fs.cleanupCount != 1 {
		t.Fatalf("expected 1 cleanup call, got %d", fs.cleanupCount)
	}
}

func TestApplyDeletesDeduplicatesCleanupDirectories(t *testing.T) {
	fs := &deleteTrackingFileSystem{}
	service := NewService(fs, nil)

	err := service.applyDeletes(
		Request{TargetRoot: "target"},
		[]model.DiffEntry{
			{Path: "dir/a.txt", Kind: model.DiffKindDeleted},
			{Path: "dir/b.txt", Kind: model.DiffKindDeleted},
		},
		2,
	)
	if err != nil {
		t.Fatalf("apply deletes failed: %v", err)
	}
	if fs.cleanupCount != 1 {
		t.Fatalf("expected 1 cleanup call, got %d", fs.cleanupCount)
	}
}

func TestApplyDeletesPassesDeletedFilePathToCleanup(t *testing.T) {
	fs := &deleteTrackingFileSystem{}
	service := NewService(fs, nil)

	err := service.applyDeletes(
		Request{TargetRoot: `C:\target`},
		[]model.DiffEntry{{Path: "dir/a.txt", Kind: model.DiffKindDeleted}},
		1,
	)
	if err != nil {
		t.Fatalf("apply deletes failed: %v", err)
	}
	if fs.lastCleanupStart != `C:\target\dir\a.txt` {
		t.Fatalf("unexpected cleanup start: got %q", fs.lastCleanupStart)
	}
}

func TestJoinPath(t *testing.T) {
	testCases := []struct {
		root    string
		relPath string
		want    string
	}{
		{root: `C:\target`, relPath: "dir/file.txt", want: `C:\target\dir\file.txt`},
		{root: `C:\target\`, relPath: "dir/file.txt", want: `C:\target\dir\file.txt`},
		{root: "", relPath: "dir/file.txt", want: `dir\file.txt`},
		{root: "", relPath: `dir\file.txt`, want: `dir\file.txt`},
	}

	for _, testCase := range testCases {
		if got := joinPath(testCase.root, testCase.relPath); got != testCase.want {
			t.Fatalf("joinPath(%q, %q) = %q, want %q", testCase.root, testCase.relPath, got, testCase.want)
		}
	}
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

func BenchmarkJoinPath(b *testing.B) {
	root := `C:\target`
	relPath := "dir/file-1234.txt"

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		path := joinPath(root, relPath)
		if path == "" {
			b.Fatal("expected joined path")
		}
	}
}

func BenchmarkSyncLargeMixed(b *testing.B) {
	const fileCount = 4000
	entries := make([]model.DiffEntry, 0, fileCount)
	for index := 0; index < fileCount; index++ {
		path := filepath.ToSlash(filepath.Join("dir", "nested", "file-"+formatIndex(index)+".txt"))
		switch {
		case index%4 == 0:
			entries = append(entries, model.DiffEntry{Path: path, Kind: model.DiffKindDeleted})
		case index%3 == 0:
			entries = append(entries, model.DiffEntry{Path: path, Kind: model.DiffKindModified})
		default:
			entries = append(entries, model.DiffEntry{Path: path, Kind: model.DiffKindAdded})
		}
	}

	service := NewService(&benchmarkSyncFileSystem{}, benchmarkDifferencePlanner{entries: entries})
	request := Request{SourceRoot: `C:\source`, TargetRoot: `C:\target`}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if err := service.Sync(request); err != nil {
			b.Fatalf("sync failed: %v", err)
		}
	}
}

func BenchmarkApplyDeletesManyFilesSameDirs(b *testing.B) {
	const fileCount = 4000
	entries := make([]model.DiffEntry, 0, fileCount)
	for index := 0; index < fileCount; index++ {
		dir := filepath.ToSlash(filepath.Join("dir", "nested", "bucket-"+formatIndex(index%32)))
		path := filepath.ToSlash(filepath.Join(dir, "file-"+formatIndex(index)+".txt"))
		entries = append(entries, model.DiffEntry{Path: path, Kind: model.DiffKindDeleted})
	}

	service := NewService(&benchmarkSyncFileSystem{}, nil)
	request := Request{TargetRoot: `C:\target`}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if err := service.applyDeletes(request, entries, fileCount); err != nil {
			b.Fatalf("apply deletes failed: %v", err)
		}
	}
}

func BenchmarkApplyCopiesManyFiles(b *testing.B) {
	const fileCount = 4000
	entries := make([]model.DiffEntry, 0, fileCount)
	for index := 0; index < fileCount; index++ {
		path := filepath.ToSlash(filepath.Join("dir", "nested", "file-"+formatIndex(index)+".txt"))
		kind := model.DiffKindAdded
		if index%3 == 0 {
			kind = model.DiffKindModified
		}
		entries = append(entries, model.DiffEntry{Path: path, Kind: kind})
	}

	service := NewService(&benchmarkSyncFileSystem{}, nil)
	request := Request{SourceRoot: `C:\source`, TargetRoot: `C:\target`}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if err := service.applyCopies(request, entries, fileCount); err != nil {
			b.Fatalf("apply copies failed: %v", err)
		}
	}
}

type copyTrackingFileSystem struct {
	copyCount     int
	firstStarted  chan struct{}
	secondStarted chan struct{}
}

func (fs *copyTrackingFileSystem) ListRegularFiles(string) ([]string, error) {
	panic("unexpected ListRegularFiles call")
}

func (fs *copyTrackingFileSystem) Exists(string) (bool, error) {
	panic("unexpected Exists call")
}

func (fs *copyTrackingFileSystem) CompareFile(string, string) (platform.FileComparison, error) {
	panic("unexpected CompareFile call")
}

func (fs *copyTrackingFileSystem) CompareFileFromRoots(string, string, string) (platform.FileComparison, error) {
	panic("unexpected CompareFileFromRoots call")
}

func (fs *copyTrackingFileSystem) FilesEqual(string, string) (bool, error) {
	panic("unexpected FilesEqual call")
}

func (fs *copyTrackingFileSystem) FileSize(string) (int64, error) {
	panic("unexpected FileSize call")
}

func (fs *copyTrackingFileSystem) FileSizeFromRoot(string, string) (int64, error) {
	panic("unexpected FileSizeFromRoot call")
}

func (fs *copyTrackingFileSystem) CopyFile(sourcePath string, _ string) error {
	switch filepath.Base(sourcePath) {
	case "a.txt":
		fs.copyCount++
		close(fs.firstStarted)
		select {
		case <-fs.secondStarted:
			return nil
		case <-time.After(time.Second):
			return os.ErrDeadlineExceeded
		}
	case "b.txt":
		fs.copyCount++
		close(fs.secondStarted)
		return nil
	default:
		return nil
	}
}

func (fs *copyTrackingFileSystem) CopyFileFromRoots(_ string, _ string, relPath string) error {
	return fs.CopyFile(relPath, "")
}

func (fs *copyTrackingFileSystem) EnsureDirectory(string) error {
	panic("unexpected EnsureDirectory call")
}

func (fs *copyTrackingFileSystem) Remove(string) error {
	panic("unexpected Remove call")
}

func (fs *copyTrackingFileSystem) RemoveFromRoot(string, string) error {
	panic("unexpected RemoveFromRoot call")
}

func (fs *copyTrackingFileSystem) RemoveEmptyParents(string, string) error {
	panic("unexpected RemoveEmptyParents call")
}

func (fs *copyTrackingFileSystem) RemoveEmptyParentsFromRoot(string, string) error {
	panic("unexpected RemoveEmptyParentsFromRoot call")
}

type deleteTrackingFileSystem struct {
	removeCount      int
	cleanupCount     int
	firstStarted     chan struct{}
	secondStarted    chan struct{}
	lastCleanupStart string
}

func (fs *deleteTrackingFileSystem) ListRegularFiles(string) ([]string, error) {
	panic("unexpected ListRegularFiles call")
}

func (fs *deleteTrackingFileSystem) Exists(string) (bool, error) {
	panic("unexpected Exists call")
}

func (fs *deleteTrackingFileSystem) CompareFile(string, string) (platform.FileComparison, error) {
	panic("unexpected CompareFile call")
}

func (fs *deleteTrackingFileSystem) CompareFileFromRoots(string, string, string) (platform.FileComparison, error) {
	panic("unexpected CompareFileFromRoots call")
}

func (fs *deleteTrackingFileSystem) FilesEqual(string, string) (bool, error) {
	panic("unexpected FilesEqual call")
}

func (fs *deleteTrackingFileSystem) FileSize(string) (int64, error) {
	panic("unexpected FileSize call")
}

func (fs *deleteTrackingFileSystem) FileSizeFromRoot(string, string) (int64, error) {
	panic("unexpected FileSizeFromRoot call")
}

func (fs *deleteTrackingFileSystem) CopyFile(string, string) error {
	panic("unexpected CopyFile call")
}

func (fs *deleteTrackingFileSystem) CopyFileFromRoots(string, string, string) error {
	panic("unexpected CopyFileFromRoots call")
}

func (fs *deleteTrackingFileSystem) EnsureDirectory(string) error {
	panic("unexpected EnsureDirectory call")
}

func (fs *deleteTrackingFileSystem) Remove(targetPath string) error {
	switch filepath.Base(targetPath) {
	case "a.txt":
		fs.removeCount++
		if fs.firstStarted != nil {
			close(fs.firstStarted)
		}
		if fs.secondStarted == nil {
			return nil
		}
		select {
		case <-fs.secondStarted:
			return nil
		case <-time.After(time.Second):
			return os.ErrDeadlineExceeded
		}
	case "b.txt":
		fs.removeCount++
		if fs.secondStarted != nil {
			close(fs.secondStarted)
		}
		return nil
	default:
		return nil
	}
}

func (fs *deleteTrackingFileSystem) RemoveFromRoot(_ string, relPath string) error {
	return fs.Remove(relPath)
}

func (fs *deleteTrackingFileSystem) RemoveEmptyParents(_ string, start string) error {
	fs.cleanupCount++
	fs.lastCleanupStart = start
	return nil
}

func (fs *deleteTrackingFileSystem) RemoveEmptyParentsFromRoot(root string, relPath string) error {
	return fs.RemoveEmptyParents(root, joinPath(root, relPath))
}

type benchmarkDifferencePlanner struct {
	entries []model.DiffEntry
}

func (planner benchmarkDifferencePlanner) Calculate(diff.Request) (diff.Result, error) {
	return diff.Result{
		Entries: planner.entries,
		Summary: model.BuildDiffSummary(planner.entries),
	}, nil
}

type benchmarkSyncFileSystem struct{}

func (fs *benchmarkSyncFileSystem) ListRegularFiles(string) ([]string, error) {
	panic("unexpected ListRegularFiles call")
}

func (fs *benchmarkSyncFileSystem) Exists(string) (bool, error) {
	panic("unexpected Exists call")
}

func (fs *benchmarkSyncFileSystem) CompareFile(string, string) (platform.FileComparison, error) {
	panic("unexpected CompareFile call")
}

func (fs *benchmarkSyncFileSystem) CompareFileFromRoots(string, string, string) (platform.FileComparison, error) {
	panic("unexpected CompareFileFromRoots call")
}

func (fs *benchmarkSyncFileSystem) FilesEqual(string, string) (bool, error) {
	panic("unexpected FilesEqual call")
}

func (fs *benchmarkSyncFileSystem) FileSize(string) (int64, error) {
	panic("unexpected FileSize call")
}

func (fs *benchmarkSyncFileSystem) FileSizeFromRoot(string, string) (int64, error) {
	panic("unexpected FileSizeFromRoot call")
}

func (fs *benchmarkSyncFileSystem) CopyFile(string, string) error {
	return nil
}

func (fs *benchmarkSyncFileSystem) CopyFileFromRoots(string, string, string) error {
	return nil
}

func (fs *benchmarkSyncFileSystem) EnsureDirectory(string) error {
	panic("unexpected EnsureDirectory call")
}

func (fs *benchmarkSyncFileSystem) Remove(string) error {
	return nil
}

func (fs *benchmarkSyncFileSystem) RemoveFromRoot(string, string) error {
	return nil
}

func (fs *benchmarkSyncFileSystem) RemoveEmptyParents(string, string) error {
	return nil
}

func (fs *benchmarkSyncFileSystem) RemoveEmptyParentsFromRoot(string, string) error {
	return nil
}

func formatIndex(index int) string {
	const digits = "0123456789"
	if index == 0 {
		return "0"
	}
	var buffer [16]byte
	writeIndex := len(buffer)
	for index > 0 {
		writeIndex--
		buffer[writeIndex] = digits[index%10]
		index /= 10
	}
	return string(buffer[writeIndex:])
}
