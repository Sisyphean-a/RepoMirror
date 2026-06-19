package app

import (
	"fmt"
	"sync"

	"RepoMirror/internal/diff"
	"RepoMirror/internal/model"
)

type repositoryProbe struct {
	summary model.RepositorySummary
	status  model.TargetRepositoryStatus
}

func (s *Service) buildState(cfg model.AppConfig) (model.DashboardState, error) {
	repositoryA, repositoryB := s.probeRepositories(cfg)
	state := model.DashboardState{
		Config:       cfg,
		RepositoryA:  repositoryA.summary,
		RepositoryB:  repositoryB.summary,
		SourceSlot:   cfg.Direction.SourceSlot(),
		TargetSlot:   cfg.Direction.TargetSlot(),
		Summary:      model.BuildDiffSummary(nil),
		Differences:  make([]model.DiffEntry, 0),
		TargetStatus: targetStatusForSlot(cfg.Direction.TargetSlot(), repositoryA, repositoryB),
	}
	if err := s.enrichDifferences(&state); err != nil {
		return model.DashboardState{}, err
	}
	return state, nil
}

func (s *Service) probeRepositories(cfg model.AppConfig) (repositoryProbe, repositoryProbe) {
	var waitGroup sync.WaitGroup
	var repositoryA repositoryProbe
	var repositoryB repositoryProbe

	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		repositoryA = s.probeRepository(model.RepositorySlotA, cfg.ProjectA)
	}()
	go func() {
		defer waitGroup.Done()
		repositoryB = s.probeRepository(model.RepositorySlotB, cfg.ProjectB)
	}()
	waitGroup.Wait()

	return repositoryA, repositoryB
}

func (s *Service) probeRepository(slot model.RepositorySlot, path string) repositoryProbe {
	summary := model.RepositorySummary{
		Slot:         slot,
		Path:         path,
		Name:         model.RepositoryName(path),
		IsConfigured: path != "",
	}
	probe := repositoryProbe{
		summary: summary,
		status: model.TargetRepositoryStatus{
			Path: summary.Path,
			Name: summary.Name,
		},
	}
	if !summary.IsConfigured {
		return probe
	}
	status, err := s.inspector.ReadTargetStatus(path)
	if err != nil {
		probe.summary.ValidationError = err.Error()
		probe.status.Error = err.Error()
		return probe
	}
	probe.summary.Path = status.Path
	probe.summary.Name = status.Name
	probe.summary.IsGitRepo = true
	probe.summary.Branch = status.Branch
	probe.summary.IsClean = status.IsClean
	probe.summary.ModifiedCount = status.ModifiedCount
	probe.summary.UntrackedCount = status.UntrackedCount
	probe.status = status
	return probe
}

func (s *Service) enrichDifferences(state *model.DashboardState) error {
	source := sourceSummary(*state)
	target := targetSummary(*state)
	if !source.IsGitRepo || !target.IsGitRepo {
		state.CanSync = false
		return nil
	}
	result, err := s.differ.Calculate(diff.Request{
		SourceRoot: source.Path,
		TargetRoot: target.Path,
	})
	if err != nil {
		return fmt.Errorf("failed to calculate repository differences: %w", err)
	}
	state.Differences = result.Entries
	state.Summary = result.Summary
	state.CanSync = result.Summary.Added+result.Summary.Modified+result.Summary.Deleted > 0
	return nil
}

func sourceSummary(state model.DashboardState) model.RepositorySummary {
	if state.SourceSlot == model.RepositorySlotB {
		return state.RepositoryB
	}
	return state.RepositoryA
}

func targetStatusForSlot(
	slot model.RepositorySlot,
	repositoryA repositoryProbe,
	repositoryB repositoryProbe,
) model.TargetRepositoryStatus {
	selected := repositoryA
	if slot == model.RepositorySlotB {
		selected = repositoryB
	}
	if selected.summary.IsGitRepo {
		return selected.status
	}
	return model.TargetRepositoryStatus{
		Path:      selected.summary.Path,
		Name:      selected.summary.Name,
		IsGitRepo: selected.summary.IsGitRepo,
		Error:     selected.summary.ValidationError,
	}
}

func targetSummary(state model.DashboardState) model.RepositorySummary {
	if state.TargetSlot == model.RepositorySlotA {
		return state.RepositoryA
	}
	return state.RepositoryB
}
