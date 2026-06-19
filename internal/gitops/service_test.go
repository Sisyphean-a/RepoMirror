package gitops

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"RepoMirror/internal/model"
	"RepoMirror/internal/testutil"
)

func TestIgnoredPathsReturnsEmptyWhenNothingMatches(t *testing.T) {
	repo := t.TempDir()
	testutil.InitRepo(t, repo)
	testutil.WriteFile(t, repo, ".gitignore", "ignored/\n")
	testutil.CommitAll(t, repo, "init")

	service := NewService(NewExecRunner())
	ignored, err := service.IgnoredPaths(repo, []string{"tracked.txt", "notes.md"})
	if err != nil {
		t.Fatalf("ignored paths failed: %v", err)
	}
	if len(ignored) != 0 {
		t.Fatalf("expected no ignored paths, got %+v", ignored)
	}
}

func TestIgnoredPathsReturnsRuleLabels(t *testing.T) {
	repo := t.TempDir()
	testutil.InitRepo(t, repo)
	testutil.WriteFile(t, repo, ".gitignore", "ignored/\n*.env\nconfig/*.yaml\n")
	testutil.CommitAll(t, repo, "init")

	service := NewService(NewExecRunner())
	ignored, err := service.IgnoredPaths(repo, []string{"ignored/a.txt", "prod.env", "config/a.yaml"})
	if err != nil {
		t.Fatalf("ignored paths failed: %v", err)
	}
	if ignored["ignored/a.txt"] != "ignore-protected" {
		t.Fatalf("unexpected ignored label for directory rule: %+v", ignored)
	}
	if ignored["prod.env"] != "env-protected" {
		t.Fatalf("unexpected ignored label for env rule: %+v", ignored)
	}
	if ignored["config/a.yaml"] != "cfg-protected" {
		t.Fatalf("unexpected ignored label for config rule: %+v", ignored)
	}
}

func TestIgnoredPathSetFromRootReturnsMatchingPaths(t *testing.T) {
	runner := &runnerStub{
		responses: map[string][]byte{
			"check-ignore --stdin --no-index": []byte("ignored/a.txt\nprod.env\n"),
		},
	}

	ignored, err := NewService(runner).IgnoredPathSetFromRoot("repo", []string{"ignored/a.txt", "prod.env", "keep.txt"})
	if err != nil {
		t.Fatalf("ignored path set failed: %v", err)
	}

	expected := map[string]struct{}{
		"ignored/a.txt": {},
		"prod.env":      {},
	}
	if !reflect.DeepEqual(ignored, expected) {
		t.Fatalf("unexpected ignored path set: got %v want %v", ignored, expected)
	}
}

func TestIgnoredPathSetFromRootSortedReturnsMatchingPaths(t *testing.T) {
	runner := &runnerStub{
		responses: map[string][]byte{
			"check-ignore --stdin --no-index": []byte("ignored/a.txt\nprod.env\n"),
		},
	}

	ignored, err := NewService(runner).IgnoredPathSetFromRootSorted("repo", []string{"ignored/a.txt", "prod.env", "keep.txt"})
	if err != nil {
		t.Fatalf("ignored sorted path set failed: %v", err)
	}

	expected := map[string]struct{}{
		"ignored/a.txt": {},
		"prod.env":      {},
	}
	if !reflect.DeepEqual(ignored, expected) {
		t.Fatalf("unexpected ignored sorted path set: got %v want %v", ignored, expected)
	}
}

func TestListSyncableSourcePathsSkipsDeletedTrackedFiles(t *testing.T) {
	repo := t.TempDir()
	testutil.InitRepo(t, repo)
	testutil.WriteFile(t, repo, "keep.txt", "keep")
	testutil.WriteFile(t, repo, "delete.txt", "delete")
	testutil.CommitAll(t, repo, "init")
	if err := os.Remove(repo + string(os.PathSeparator) + "delete.txt"); err != nil {
		t.Fatalf("remove tracked file failed: %v", err)
	}
	testutil.WriteFile(t, repo, "new.txt", "new")

	paths, err := NewService(NewExecRunner()).ListSyncableSourcePaths(repo)
	if err != nil {
		t.Fatalf("list syncable source paths failed: %v", err)
	}

	expected := []string{"keep.txt", "new.txt"}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected syncable paths: got %v want %v", paths, expected)
	}
}

func TestListSyncableSourcePathsParsesTaggedOutputAndDeduplicates(t *testing.T) {
	runner := &runnerStub{
		responses: map[string][]byte{
			"ls-files -t --cached --others --deleted --exclude-standard -z": []byte("? new.txt\x00H keep.txt\x00R gone.txt\x00H keep.txt\x00"),
		},
	}

	paths, err := NewService(runner).ListSyncableSourcePathsFromRoot("repo")
	if err != nil {
		t.Fatalf("list syncable source paths failed: %v", err)
	}

	expected := []string{"keep.txt", "new.txt"}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected tagged output paths: got %v want %v", paths, expected)
	}
}

func TestReadTargetStatusUsesSingleStatusCommandAndParsesDetachedHead(t *testing.T) {
	runner := &runnerStub{
		responses: map[string][]byte{
			"rev-parse --show-toplevel":     []byte("C:/repo\n"),
			"status --porcelain=2 --branch": []byte("# branch.oid abc\n# branch.head (detached)\n1 .M N... 100644 100644 100644 a a tracked.txt\n? untracked.txt\n"),
		},
	}

	status, err := NewService(runner).ReadTargetStatus("repo")
	if err != nil {
		t.Fatalf("read target status failed: %v", err)
	}

	expectedStatus := model.TargetRepositoryStatus{
		Path:           filepath.Clean("C:/repo"),
		Name:           "repo",
		Branch:         "HEAD",
		IsGitRepo:      true,
		IsClean:        false,
		ModifiedCount:  1,
		UntrackedCount: 1,
	}
	if status != expectedStatus {
		t.Fatalf("unexpected target status: got %+v want %+v", status, expectedStatus)
	}

	expectedCalls := map[string]bool{
		"rev-parse --show-toplevel":     true,
		"status --porcelain=2 --branch": true,
	}
	if len(runner.calls) != len(expectedCalls) {
		t.Fatalf("unexpected git call count: got %v want %v", runner.calls, len(expectedCalls))
	}
	for _, call := range runner.calls {
		if !expectedCalls[call] {
			t.Fatalf("unexpected git call: %s", call)
		}
	}
}

func TestReadTargetStatusCachesResolvedRoot(t *testing.T) {
	runner := &runnerStub{
		responses: map[string][]byte{
			"rev-parse --show-toplevel":     []byte("C:/repo\n"),
			"status --porcelain=2 --branch": []byte("# branch.oid abc\n# branch.head main\n"),
		},
	}
	service := NewService(runner)

	for attempt := 0; attempt < 2; attempt++ {
		if _, err := service.ReadTargetStatus("repo"); err != nil {
			t.Fatalf("read target status failed: %v", err)
		}
	}

	revParseCount := 0
	statusCount := 0
	for _, call := range runner.calls {
		switch call {
		case "rev-parse --show-toplevel":
			revParseCount++
		case "status --porcelain=2 --branch":
			statusCount++
		}
	}
	if revParseCount != 1 {
		t.Fatalf("expected rev-parse once, got %d", revParseCount)
	}
	if statusCount != 2 {
		t.Fatalf("expected status twice, got %d", statusCount)
	}
}

func TestBuildLineSeparatedInputDeduplicatesAcrossPathGroups(t *testing.T) {
	input := buildLineSeparatedInput(nil,
		[]string{"a.txt", "shared.txt", ""},
		[]string{"shared.txt", "b.txt"},
		[]string{"a.txt", "c.txt"},
	)

	expected := "a.txt\nshared.txt\nb.txt\nc.txt\n"
	if string(input) != expected {
		t.Fatalf("unexpected line-separated input: got %q want %q", input, expected)
	}
}

func TestBuildLineSeparatedInputDeduplicatesWithinSingleGroup(t *testing.T) {
	input := buildLineSeparatedInput(nil, []string{"a.txt", "a.txt", "b.txt", "", "b.txt"})

	expected := "a.txt\nb.txt\n"
	if string(input) != expected {
		t.Fatalf("unexpected single-group input: got %q want %q", input, expected)
	}
}

func TestBuildLineSeparatedInputKeepsOrderedUniqueSingleGroup(t *testing.T) {
	input := buildLineSeparatedInput(nil, []string{"a.txt", "b.txt", "c.txt"})

	expected := "a.txt\nb.txt\nc.txt\n"
	if string(input) != expected {
		t.Fatalf("unexpected ordered input: got %q want %q", input, expected)
	}
}

func TestBuildTargetStatusParsesBranchAndCounts(t *testing.T) {
	status := buildTargetStatus("C:/repo", []byte("# branch.oid abc\n# branch.head main\n1 .M N... 100644 100644 100644 a a tracked.txt\n? untracked.txt\n"))

	expected := model.TargetRepositoryStatus{
		Path:           filepath.Clean("C:/repo"),
		Name:           "repo",
		Branch:         "main",
		IsGitRepo:      true,
		IsClean:        false,
		ModifiedCount:  1,
		UntrackedCount: 1,
	}
	if status != expected {
		t.Fatalf("unexpected target status: got %+v want %+v", status, expected)
	}
}

func TestParseTaggedPath(t *testing.T) {
	status, path := parseTaggedPath("R dir/file.txt")
	if status != "R" || path != "dir/file.txt" {
		t.Fatalf("unexpected tagged path parse: status=%q path=%q", status, path)
	}
}

func BenchmarkListSyncableSourcePathsFromRoot(b *testing.B) {
	var output strings.Builder
	for index := 0; index < 4000; index++ {
		switch {
		case index%11 == 0:
			output.WriteString("R dir/file-")
		case index%7 == 0:
			output.WriteString("? dir/file-")
		default:
			output.WriteString("H dir/file-")
		}
		output.WriteString(fmt.Sprintf("%04d.txt", index))
		output.WriteByte(0)
		if index%9 == 0 {
			output.WriteString("H dir/file-")
			output.WriteString(fmt.Sprintf("%04d.txt", index))
			output.WriteByte(0)
		}
	}

	runner := &runnerStub{
		responses: map[string][]byte{
			"ls-files -t --cached --others --deleted --exclude-standard -z": []byte(output.String()),
		},
	}
	service := NewService(runner)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		paths, err := service.ListSyncableSourcePathsFromRoot("repo")
		if err != nil {
			b.Fatalf("list syncable source paths failed: %v", err)
		}
		if len(paths) == 0 {
			b.Fatal("expected syncable paths")
		}
	}
}

func BenchmarkBuildLineSeparatedInput(b *testing.B) {
	groupA := make([]string, 0, 4000)
	groupB := make([]string, 0, 4000)
	for index := 0; index < 4000; index++ {
		path := fmt.Sprintf("dir/file-%04d.txt", index)
		groupA = append(groupA, path)
		if index%3 != 0 {
			groupB = append(groupB, path)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		input := buildLineSeparatedInput(nil, groupA, groupB)
		if len(input) == 0 {
			b.Fatal("expected line-separated input")
		}
	}
}

func BenchmarkIgnoredPathsFromRoot(b *testing.B) {
	paths := make([]string, 0, 4000)
	var output strings.Builder
	for index := 0; index < 4000; index++ {
		path := fmt.Sprintf("dir/file-%04d.txt", index)
		paths = append(paths, path)
		if index%5 == 0 {
			output.WriteString(".gitignore:1:ignored/\t")
			output.WriteString(path)
			output.WriteByte('\n')
		}
	}

	runner := &runnerStub{
		responses: map[string][]byte{
			"check-ignore -v --stdin --no-index": []byte(output.String()),
		},
	}
	service := NewService(runner)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		ignored, err := service.IgnoredPathsFromRoot("repo", paths)
		if err != nil {
			b.Fatalf("ignored paths failed: %v", err)
		}
		if len(ignored) == 0 {
			b.Fatal("expected ignored paths")
		}
	}
}

func BenchmarkIgnoredPathSetFromRoot(b *testing.B) {
	paths := make([]string, 0, 4000)
	var output strings.Builder
	for index := 0; index < 4000; index++ {
		path := fmt.Sprintf("dir/file-%04d.txt", index)
		paths = append(paths, path)
		if index%5 == 0 {
			output.WriteString(path)
			output.WriteByte('\n')
		}
	}

	runner := &runnerStub{
		responses: map[string][]byte{
			"check-ignore --stdin --no-index": []byte(output.String()),
		},
	}
	service := NewService(runner)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		ignored, err := service.IgnoredPathSetFromRoot("repo", paths)
		if err != nil {
			b.Fatalf("ignored path set failed: %v", err)
		}
		if len(ignored) == 0 {
			b.Fatal("expected ignored paths")
		}
	}
}

func BenchmarkIgnoredPathSetFromRootSorted(b *testing.B) {
	paths := make([]string, 0, 4000)
	var output strings.Builder
	for index := 0; index < 4000; index++ {
		path := fmt.Sprintf("dir/file-%04d.txt", index)
		paths = append(paths, path)
		if index%5 == 0 {
			output.WriteString(path)
			output.WriteByte('\n')
		}
	}

	runner := &runnerStub{
		responses: map[string][]byte{
			"check-ignore --stdin --no-index": []byte(output.String()),
		},
	}
	service := NewService(runner)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		ignored, err := service.IgnoredPathSetFromRootSorted("repo", paths)
		if err != nil {
			b.Fatalf("ignored sorted path set failed: %v", err)
		}
		if len(ignored) == 0 {
			b.Fatal("expected ignored paths")
		}
	}
}

func BenchmarkBuildTargetStatus(b *testing.B) {
	var output strings.Builder
	output.WriteString("# branch.oid abcdef\n# branch.head main\n")
	for index := 0; index < 4000; index++ {
		if index%5 == 0 {
			output.WriteString("? dir/file-")
			output.WriteString(fmt.Sprintf("%04d.txt\n", index))
			continue
		}
		output.WriteString("1 .M N... 100644 100644 100644 a a dir/file-")
		output.WriteString(fmt.Sprintf("%04d.txt\n", index))
	}

	statusOutput := []byte(output.String())

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		status := buildTargetStatus("C:/repo", statusOutput)
		if status.ModifiedCount == 0 || status.UntrackedCount == 0 {
			b.Fatalf("unexpected status: %+v", status)
		}
	}
}

func BenchmarkReadTargetStatusFromRoot(b *testing.B) {
	var output strings.Builder
	output.WriteString("# branch.oid abcdef\n# branch.head main\n")
	for index := 0; index < 4000; index++ {
		if index%5 == 0 {
			output.WriteString("? dir/file-")
			output.WriteString(fmt.Sprintf("%04d.txt\n", index))
			continue
		}
		output.WriteString("1 .M N... 100644 100644 100644 a a dir/file-")
		output.WriteString(fmt.Sprintf("%04d.txt\n", index))
	}

	runner := &runnerStub{
		responses: map[string][]byte{
			"status --porcelain=2 --branch": []byte(output.String()),
		},
	}
	service := NewService(runner)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		status, err := service.ReadTargetStatusFromRoot("C:/repo")
		if err != nil {
			b.Fatalf("read target status from root failed: %v", err)
		}
		if status.ModifiedCount == 0 || status.UntrackedCount == 0 {
			b.Fatalf("unexpected status: %+v", status)
		}
	}
}

type runnerStub struct {
	mu        sync.Mutex
	calls     []string
	responses map[string][]byte
}

func (stub *runnerStub) Run(_ string, _ []byte, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	stub.mu.Lock()
	stub.calls = append(stub.calls, key)
	output, ok := stub.responses[key]
	stub.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("unexpected args: %s", key)
	}
	return output, nil
}
