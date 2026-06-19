package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"RepoMirror/internal/config"
	"RepoMirror/internal/diff"
	"RepoMirror/internal/gitops"
	"RepoMirror/internal/model"
	"RepoMirror/internal/platform"
	"RepoMirror/internal/syncer"
	"RepoMirror/internal/testutil"
)

func TestServiceAutoPersistsConfigChanges(t *testing.T) {
	repoA := t.TempDir()
	repoB := t.TempDir()
	testutil.InitRepo(t, repoA)
	testutil.InitRepo(t, repoB)

	store := config.NewStoreWithPath(filepath.Join(t.TempDir(), "config.json"))
	service := newTestService(t, store, &selectorStub{paths: []string{repoA, repoB}}, model.DefaultConfig())
	service.Startup(context.Background())

	state, err := service.SelectRepository("A")
	if err != nil {
		t.Fatalf("select repo A failed: %v", err)
	}
	if !state.RepositoryA.IsGitRepo {
		t.Fatalf("repository A should be valid git repo")
	}

	state, err = service.SelectRepository("B")
	if err != nil {
		t.Fatalf("select repo B failed: %v", err)
	}
	if !state.RepositoryB.IsGitRepo {
		t.Fatalf("repository B should be valid git repo")
	}

	state, err = service.SwapRepositories()
	if err != nil {
		t.Fatalf("swap repositories failed: %v", err)
	}
	if filepath.Clean(state.Config.ProjectA) != filepath.Clean(repoB) ||
		filepath.Clean(state.Config.ProjectB) != filepath.Clean(repoA) {
		t.Fatalf("unexpected repository order after swap: %+v", state.Config)
	}

	state, err = service.SetDirection(string(model.DirectionBToA))
	if err != nil {
		t.Fatalf("set direction failed: %v", err)
	}
	if state.SourceSlot != model.RepositorySlotB || state.TargetSlot != model.RepositorySlotA {
		t.Fatalf("unexpected slots after direction change: %+v", state)
	}

	reloadedConfig, err := store.Load()
	if err != nil {
		t.Fatalf("reload config failed: %v", err)
	}
	if filepath.Clean(reloadedConfig.ProjectA) != filepath.Clean(repoB) ||
		filepath.Clean(reloadedConfig.ProjectB) != filepath.Clean(repoA) {
		t.Fatalf("expected swapped repositories in saved config, got %+v", reloadedConfig)
	}
	if reloadedConfig.Direction != model.DirectionBToA {
		t.Fatalf("expected saved direction %s, got %s", model.DirectionBToA, reloadedConfig.Direction)
	}
	reloaded := newTestService(t, store, &selectorStub{}, reloadedConfig)
	reloaded.Startup(context.Background())
	reloadedState, err := reloaded.LoadState()
	if err != nil {
		t.Fatalf("load reloaded state failed: %v", err)
	}

	if reloadedState.Config.Direction != model.DirectionBToA {
		t.Fatalf("expected restored direction %s, got %s", model.DirectionBToA, reloadedState.Config.Direction)
	}
	if reloadedState.RepositoryA.Path == "" || reloadedState.RepositoryB.Path == "" {
		t.Fatalf("expected restored repository paths, got %+v", reloadedState.Config)
	}
}

func TestServiceDoesNotMutateConfigWhenAutoPersistFails(t *testing.T) {
	repoA := t.TempDir()
	testutil.InitRepo(t, repoA)

	store := config.NewStoreWithPath(t.TempDir())
	service := newTestService(t, store, &selectorStub{paths: []string{repoA}}, model.DefaultConfig())
	service.Startup(context.Background())

	if _, err := service.SelectRepository("A"); err == nil {
		t.Fatalf("expected select repository to fail when config persistence fails")
	}

	state, err := service.LoadState()
	if err != nil {
		t.Fatalf("load state after failed save: %v", err)
	}
	if state.Config.ProjectA != "" || state.RepositoryA.IsConfigured {
		t.Fatalf("expected config to stay unchanged after failed save, got %+v", state.Config)
	}
}

func TestServiceSyncCommitAndPushFlow(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote.git")

	testutil.InitRepo(t, source)
	testutil.InitRepo(t, target)
	testutil.InitBareRepo(t, remote)

	testutil.WriteFile(t, source, ".gitignore", "ignored/\n")
	testutil.WriteFile(t, source, "tracked.txt", "source tracked")
	testutil.WriteFile(t, source, "notes.md", "source note")
	testutil.WriteFile(t, source, "ignored/skip.txt", "ignored")
	testutil.CommitAll(t, source, "source init")

	testutil.WriteFile(t, target, ".gitignore", "ignored/\n")
	testutil.WriteFile(t, target, "tracked.txt", "target old")
	testutil.CommitAll(t, target, "target init")
	testutil.RunGitOutput(t, target, "remote", "add", "origin", remote)
	branch := testutil.CurrentBranch(t, target)
	testutil.RunGitOutput(t, target, "push", "-u", "origin", branch)

	testutil.WriteFile(t, target, "tracked.txt", "target dirty")
	testutil.WriteFile(t, target, "scratch.txt", "untracked")

	service := newTestService(t, config.NewStoreWithPath(filepath.Join(t.TempDir(), "config.json")), &selectorStub{}, model.AppConfig{
		ProjectA:  source,
		ProjectB:  target,
		Direction: model.DirectionAToB,
	})
	service.Startup(context.Background())

	beforeState, err := service.LoadState()
	if err != nil {
		t.Fatalf("load state failed: %v", err)
	}
	if beforeState.TargetStatus.ModifiedCount == 0 || beforeState.TargetStatus.UntrackedCount == 0 {
		t.Fatalf("expected dirty target status, got %+v", beforeState.TargetStatus)
	}
	if beforeState.Summary.Total == 0 {
		t.Fatalf("expected differences before sync")
	}

	syncedState, err := service.SyncRepositories()
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if syncedState.Summary.Total != 0 {
		t.Fatalf("expected no differences after sync, got %+v", syncedState.Summary)
	}
	if syncedState.TargetStatus.IsClean {
		t.Fatalf("target should be dirty after sync before commit")
	}
	if testutil.RunGitOutput(t, target, "status", "--short") == "" {
		t.Fatalf("expected git changes after sync")
	}

	committedState, err := service.CommitTarget("sync target")
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}
	if !committedState.TargetStatus.IsClean {
		t.Fatalf("target should be clean after commit, got %+v", committedState.TargetStatus)
	}

	if _, err := service.PushTarget(); err != nil {
		t.Fatalf("push failed: %v", err)
	}
	localHead := testutil.RunGitOutput(t, target, "rev-parse", "HEAD")
	remoteHead := testutil.RunGitOutput(t, target, "--git-dir", remote, "rev-parse", "refs/heads/"+branch)
	if localHead != remoteHead {
		t.Fatalf("expected remote head %s to match local head %s", remoteHead, localHead)
	}
}

func TestServiceGenerateCommitMessageUsesSavedKey(t *testing.T) {
	target := t.TempDir()
	testutil.InitRepo(t, target)
	testutil.WriteFile(t, target, "tracked.txt", "base")
	testutil.CommitAll(t, target, "init")
	testutil.WriteFile(t, target, "tracked.txt", "next")
	testutil.WriteFile(t, target, "added.txt", "new")

	store := config.NewStoreWithPath(filepath.Join(t.TempDir(), "config.json"))
	service := newTestService(t, store, &selectorStub{}, model.AppConfig{
		ProjectA:       target,
		Direction:      model.DirectionBToA,
		AICommitAPIKey: "saved-key",
	})
	generator := &commitGeneratorStub{message: "feat(sync): 生成提交信息"}
	service.SetCommitGenerator(generator)
	service.Startup(context.Background())

	message, err := service.GenerateCommitMessage()
	if err != nil {
		t.Fatalf("GenerateCommitMessage failed: %v", err)
	}
	if message != generator.message {
		t.Fatalf("unexpected generated message: %s", message)
	}
	if generator.apiKey != "saved-key" {
		t.Fatalf("unexpected api key: %s", generator.apiKey)
	}
	if generator.changes == "" || !strings.Contains(generator.changes, "tracked.txt") {
		t.Fatalf("expected change summary to mention tracked.txt, got %q", generator.changes)
	}
}

func TestServiceSetAICommitAPIKeyPersistsWithoutExposingSecret(t *testing.T) {
	store := config.NewStoreWithPath(filepath.Join(t.TempDir(), "config.json"))
	service := newTestService(t, store, &selectorStub{}, model.DefaultConfig())
	service.Startup(context.Background())

	state, err := service.SetAICommitAPIKey("  secret-key  ")
	if err != nil {
		t.Fatalf("SetAICommitAPIKey failed: %v", err)
	}
	if !state.AICommitConfigured {
		t.Fatal("expected AI commit key to be marked as configured")
	}
	if state.Config.AICommitAPIKey != "" {
		t.Fatalf("state should hide the api key, got %q", state.Config.AICommitAPIKey)
	}
	reloaded, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load failed: %v", err)
	}
	if reloaded.AICommitAPIKey != "secret-key" {
		t.Fatalf("expected trimmed key to persist, got %q", reloaded.AICommitAPIKey)
	}
}

func newTestService(
	t *testing.T,
	store *config.Store,
	selector DirectorySelector,
	initialConfig model.AppConfig,
) *Service {
	t.Helper()
	fileSystem := platform.NewOSFileSystem()
	inspector := gitops.NewService(gitops.NewExecRunner())
	differ := diff.NewService(fileSystem, inspector)
	synchronizer := syncer.NewService(fileSystem, differ)
	return NewService(store, selector, inspector, differ, synchronizer, initialConfig)
}

type selectorStub struct {
	paths []string
	index int
}

type commitGeneratorStub struct {
	apiKey  string
	changes string
	message string
}

func (stub *commitGeneratorStub) Generate(apiKey string, changes string) (string, error) {
	stub.apiKey = apiKey
	stub.changes = changes
	return stub.message, nil
}

func (stub *selectorStub) Open(context.Context, string, string) (string, error) {
	if stub.index >= len(stub.paths) {
		return "", nil
	}
	path := stub.paths[stub.index]
	stub.index++
	return path, nil
}
