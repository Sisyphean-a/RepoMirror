package app

import (
	"errors"
	"testing"
	"time"

	"RepoMirror/internal/diff"
	"RepoMirror/internal/model"
	"RepoMirror/internal/syncer"
)

func TestBuildStateStartsDiffBeforeStatusesFinish(t *testing.T) {
	diffStarted := make(chan struct{})
	statusRelease := make(chan struct{})
	service := NewService(
		nil,
		&selectorStub{},
		inspectorStub{
			resolveRepositoryRoot: func(path string) (string, error) {
				return path + "-root", nil
			},
			readTargetStatusRoot: func(root string) (model.TargetRepositoryStatus, error) {
				<-statusRelease
				return model.TargetRepositoryStatus{
					Path:      root,
					Name:      model.RepositoryName(root),
					Branch:    "main",
					IsGitRepo: true,
					IsClean:   true,
				}, nil
			},
		},
		differenceCalculatorStub{
			calculate: func(diff.Request) (diff.Result, error) {
				close(diffStarted)
				return diff.Result{}, nil
			},
		},
		synchronizerStub{},
		model.AppConfig{ProjectA: "source", ProjectB: "target", Direction: model.DirectionAToB},
	)

	done := make(chan error, 1)
	go func() {
		_, err := service.LoadState()
		done <- err
	}()

	select {
	case <-diffStarted:
	case <-time.After(time.Second):
		t.Fatal("expected diff calculation to start before repository status completed")
	}

	close(statusRelease)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("load state failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("load state did not finish after releasing status reads")
	}
}

func TestBuildStateIgnoresDiffFailureWhenStatusFails(t *testing.T) {
	service := NewService(
		nil,
		&selectorStub{},
		inspectorStub{
			resolveRepositoryRoot: func(path string) (string, error) {
				return path + "-root", nil
			},
			readTargetStatusRoot: func(string) (model.TargetRepositoryStatus, error) {
				return model.TargetRepositoryStatus{}, errors.New("status failed")
			},
		},
		differenceCalculatorStub{
			calculate: func(diff.Request) (diff.Result, error) {
				return diff.Result{}, errors.New("diff failed")
			},
		},
		synchronizerStub{},
		model.AppConfig{ProjectA: "source", ProjectB: "target", Direction: model.DirectionAToB},
	)

	state, err := service.LoadState()
	if err != nil {
		t.Fatalf("load state failed: %v", err)
	}
	if state.RepositoryA.ValidationError != "status failed" {
		t.Fatalf("expected repository A error to be preserved, got %+v", state.RepositoryA)
	}
	if state.TargetStatus.Error != "status failed" {
		t.Fatalf("expected target status error to be preserved, got %+v", state.TargetStatus)
	}
	if state.Summary.Total != 0 || state.CanSync {
		t.Fatalf("expected no diff state when repository status fails, got %+v", state.Summary)
	}
}

type inspectorStub struct {
	resolveRepositoryRoot func(path string) (string, error)
	readTargetStatus      func(path string) (model.TargetRepositoryStatus, error)
	readTargetStatusRoot  func(root string) (model.TargetRepositoryStatus, error)
}

func (stub inspectorStub) ResolveRepositoryRoot(path string) (string, error) {
	return stub.resolveRepositoryRoot(path)
}

func (stub inspectorStub) ReadTargetStatus(path string) (model.TargetRepositoryStatus, error) {
	if stub.readTargetStatus == nil {
		return model.TargetRepositoryStatus{}, errors.New("unexpected ReadTargetStatus call")
	}
	return stub.readTargetStatus(path)
}

func (stub inspectorStub) ReadTargetStatusFromRoot(root string) (model.TargetRepositoryStatus, error) {
	if stub.readTargetStatusRoot == nil {
		return model.TargetRepositoryStatus{}, errors.New("unexpected ReadTargetStatusFromRoot call")
	}
	return stub.readTargetStatusRoot(root)
}

func (inspectorStub) DescribeWorkingTree(string) (string, error) {
	return "", nil
}

func (inspectorStub) Commit(string, string) error {
	return nil
}

func (inspectorStub) Push(string) error {
	return nil
}

type differenceCalculatorStub struct {
	calculate func(request diff.Request) (diff.Result, error)
}

func (stub differenceCalculatorStub) Calculate(request diff.Request) (diff.Result, error) {
	return stub.calculate(request)
}

type synchronizerStub struct{}

func (synchronizerStub) Sync(syncer.Request) error {
	return nil
}

func BenchmarkLoadStateConfiguredRepositories(b *testing.B) {
	service := NewService(
		nil,
		&selectorStub{},
		inspectorStub{
			resolveRepositoryRoot: func(path string) (string, error) {
				return path + "-root", nil
			},
			readTargetStatusRoot: func(root string) (model.TargetRepositoryStatus, error) {
				return model.TargetRepositoryStatus{
					Path:      root,
					Name:      model.RepositoryName(root),
					Branch:    "main",
					IsGitRepo: true,
					IsClean:   true,
				}, nil
			},
		},
		differenceCalculatorStub{
			calculate: func(request diff.Request) (diff.Result, error) {
				return diff.Result{
					Entries: []model.DiffEntry{
						{Path: "dir/file.txt", Kind: model.DiffKindModified, SizeBytes: 16},
					},
					Summary: model.DiffSummary{Total: 1, Modified: 1},
				}, nil
			},
		},
		synchronizerStub{},
		model.AppConfig{ProjectA: "source", ProjectB: "target", Direction: model.DirectionAToB},
	)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		state, err := service.LoadState()
		if err != nil {
			b.Fatalf("load state failed: %v", err)
		}
		if state.Summary.Total != 1 || !state.CanSync {
			b.Fatalf("unexpected state: %+v", state)
		}
	}
}

func BenchmarkLoadStateStatusFailureShortCircuit(b *testing.B) {
	service := NewService(
		nil,
		&selectorStub{},
		inspectorStub{
			resolveRepositoryRoot: func(path string) (string, error) {
				return path + "-root", nil
			},
			readTargetStatusRoot: func(string) (model.TargetRepositoryStatus, error) {
				return model.TargetRepositoryStatus{}, errors.New("status failed")
			},
		},
		differenceCalculatorStub{
			calculate: func(diff.Request) (diff.Result, error) {
				return diff.Result{}, nil
			},
		},
		synchronizerStub{},
		model.AppConfig{ProjectA: "source", ProjectB: "target", Direction: model.DirectionAToB},
	)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		state, err := service.LoadState()
		if err != nil {
			b.Fatalf("load state failed: %v", err)
		}
		if state.Summary.Total != 0 || state.CanSync {
			b.Fatalf("unexpected state: %+v", state)
		}
	}
}

func BenchmarkLoadStateUnconfiguredRepositories(b *testing.B) {
	service := NewService(
		nil,
		&selectorStub{},
		inspectorStub{
			resolveRepositoryRoot: func(path string) (string, error) {
				return path + "-root", nil
			},
			readTargetStatusRoot: func(root string) (model.TargetRepositoryStatus, error) {
				return model.TargetRepositoryStatus{
					Path:      root,
					Name:      model.RepositoryName(root),
					Branch:    "main",
					IsGitRepo: true,
					IsClean:   true,
				}, nil
			},
		},
		differenceCalculatorStub{
			calculate: func(diff.Request) (diff.Result, error) {
				return diff.Result{}, nil
			},
		},
		synchronizerStub{},
		model.DefaultConfig(),
	)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		state, err := service.LoadState()
		if err != nil {
			b.Fatalf("load state failed: %v", err)
		}
		if state.RepositoryA.IsConfigured || state.RepositoryB.IsConfigured || state.CanSync {
			b.Fatalf("unexpected state: %+v", state)
		}
	}
}

func BenchmarkLoadStateSingleConfiguredRepository(b *testing.B) {
	service := NewService(
		nil,
		&selectorStub{},
		inspectorStub{
			resolveRepositoryRoot: func(path string) (string, error) {
				return path + "-root", nil
			},
			readTargetStatusRoot: func(root string) (model.TargetRepositoryStatus, error) {
				return model.TargetRepositoryStatus{
					Path:      root,
					Name:      model.RepositoryName(root),
					Branch:    "main",
					IsGitRepo: true,
					IsClean:   true,
				}, nil
			},
		},
		differenceCalculatorStub{
			calculate: func(diff.Request) (diff.Result, error) {
				return diff.Result{}, nil
			},
		},
		synchronizerStub{},
		model.AppConfig{ProjectA: "source", Direction: model.DirectionAToB},
	)

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		state, err := service.LoadState()
		if err != nil {
			b.Fatalf("load state failed: %v", err)
		}
		if !state.RepositoryA.IsConfigured || state.RepositoryB.IsConfigured || state.CanSync {
			b.Fatalf("unexpected state: %+v", state)
		}
	}
}
