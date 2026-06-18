package diff

import (
	"testing"

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
	assertDiffKind(t, got, "ignored/keep.txt", model.DiffKindProtected)
	if result.Summary.Total != 4 {
		t.Fatalf("expected total summary to be 4, got %+v", result.Summary)
	}
	if result.Summary.Protected != 1 {
		t.Fatalf("expected protected summary to be 1, got %+v", result.Summary)
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
