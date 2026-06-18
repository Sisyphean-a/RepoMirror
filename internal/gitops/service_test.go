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
