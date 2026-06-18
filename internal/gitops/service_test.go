package gitops

import (
	"testing"

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
