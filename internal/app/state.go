package app

import (
	"fmt"

	"RepoMirror/internal/diff"
	"RepoMirror/internal/model"
)

func (s *Service) buildState(cfg model.AppConfig) (model.DashboardState, error) {
	state := model.DashboardState{
		Config:      cfg,
		RepositoryA: s.buildRepositorySummary(model.RepositorySlotA, cfg.ProjectA),
		RepositoryB: s.buildRepositorySummary(model.RepositorySlotB, cfg.ProjectB),
		SourceSlot:  cfg.Direction.SourceSlot(),
		TargetSlot:  cfg.Direction.TargetSlot(),
		Summary:     model.BuildDiffSummary(nil),
		Differences: make([]model.DiffEntry, 0),
	}
	if err := s.enrichTargetStatus(&state); err != nil {
		return model.DashboardState{}, err
	}
	if err := s.enrichDifferences(&state); err != nil {
		return model.DashboardState{}, err
	}
	return state, nil
}

func (s *Service) buildRepositorySummary(slot model.RepositorySlot, path string) model.RepositorySummary {
	summary := model.RepositorySummary{
		Slot:         slot,
		Path:         path,
		Name:         model.RepositoryName(path),
		IsConfigured: path != "",
	}
	if !summary.IsConfigured {
		return summary
	}
	root, err := s.inspector.ResolveRepositoryRoot(path)
	if err != nil {
		summary.ValidationError = err.Error()
		return summary
	}
	summary.Path = root
	summary.Name = model.RepositoryName(root)
	summary.IsGitRepo = true
	status, err := s.inspector.ReadTargetStatus(root)
	if err != nil {
		summary.ValidationError = err.Error()
		return summary
	}
	summary.Branch = status.Branch
	summary.IsClean = status.IsClean
	summary.ModifiedCount = status.ModifiedCount
	summary.UntrackedCount = status.UntrackedCount
	return summary
}

func (s *Service) enrichTargetStatus(state *model.DashboardState) error {
	target := targetSummary(*state)
	state.TargetStatus = model.TargetRepositoryStatus{
		Path:      target.Path,
		Name:      target.Name,
		IsGitRepo: target.IsGitRepo,
		Error:     target.ValidationError,
	}
	if !target.IsGitRepo {
		return nil
	}
	status, err := s.inspector.ReadTargetStatus(target.Path)
	if err != nil {
		return fmt.Errorf("failed to refresh target repository status: %w", err)
	}
	state.TargetStatus = status
	return nil
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

func targetSummary(state model.DashboardState) model.RepositorySummary {
	if state.TargetSlot == model.RepositorySlotA {
		return state.RepositoryA
	}
	return state.RepositoryB
}
