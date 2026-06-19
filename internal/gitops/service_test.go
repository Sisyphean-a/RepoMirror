package gitops

import (
	"fmt"
	"os"
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
		Path:           "C:/repo",
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

func TestBuildLineSeparatedInputDeduplicatesAcrossPathGroups(t *testing.T) {
	input := buildLineSeparatedInput(
		[]string{"a.txt", "shared.txt", ""},
		[]string{"shared.txt", "b.txt"},
		[]string{"a.txt", "c.txt"},
	)

	expected := "a.txt\nshared.txt\nb.txt\nc.txt\n"
	if string(input) != expected {
		t.Fatalf("unexpected line-separated input: got %q want %q", input, expected)
	}
}

func TestBuildTargetStatusParsesBranchAndCounts(t *testing.T) {
	status := buildTargetStatus("C:/repo", []byte("# branch.oid abc\n# branch.head main\n1 .M N... 100644 100644 100644 a a tracked.txt\n? untracked.txt\n"))

	expected := model.TargetRepositoryStatus{
		Path:           "C:/repo",
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
