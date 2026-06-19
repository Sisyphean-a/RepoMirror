package app

import (
	"fmt"
	"sync"

	"RepoMirror/internal/diff"
	"RepoMirror/internal/model"
)

type repositoryProbe struct {
	root    string
	summary model.RepositorySummary
	status  model.TargetRepositoryStatus
}

var emptyDifferences = make([]model.DiffEntry, 0)

func (s *Service) buildState(cfg model.AppConfig) (model.DashboardState, error) {
	repositoryA, repositoryB := s.resolveRepositories(cfg)
	result, err := s.enrichState(cfg, &repositoryA, &repositoryB)
	if err != nil {
		return model.DashboardState{}, err
	}
	sourceSlot := cfg.Direction.SourceSlot()
	targetSlot := cfg.Direction.TargetSlot()
	state := model.DashboardState{
		Config:       cfg,
		RepositoryA:  repositoryA.summary,
		RepositoryB:  repositoryB.summary,
		SourceSlot:   sourceSlot,
		TargetSlot:   targetSlot,
		Summary:      model.DiffSummary{},
		Differences:  emptyDifferences,
		TargetStatus: targetStatusForSlot(targetSlot, repositoryA, repositoryB),
	}
	if !canApplyDiff(cfg.Direction, repositoryA, repositoryB) {
		return state, nil
	}
	applyDiffResult(&state, result)
	return state, nil
}

func (s *Service) resolveRepositories(cfg model.AppConfig) (repositoryProbe, repositoryProbe) {
	if cfg.ProjectA == "" && cfg.ProjectB == "" {
		return newRepositoryProbe(model.RepositorySlotA, ""), newRepositoryProbe(model.RepositorySlotB, "")
	}
	if cfg.ProjectA == "" {
		return newRepositoryProbe(model.RepositorySlotA, ""), s.resolveRepositoryProbe(model.RepositorySlotB, cfg.ProjectB)
	}
	if cfg.ProjectB == "" {
		return s.resolveRepositoryProbe(model.RepositorySlotA, cfg.ProjectA), newRepositoryProbe(model.RepositorySlotB, "")
	}

	var waitGroup sync.WaitGroup
	var repositoryB repositoryProbe

	waitGroup.Add(1)
	go s.resolveRepositoryProbeInto(model.RepositorySlotB, cfg.ProjectB, &repositoryB, &waitGroup)
	repositoryA := s.resolveRepositoryProbe(model.RepositorySlotA, cfg.ProjectA)
	waitGroup.Wait()

	return repositoryA, repositoryB
}

func (s *Service) resolveRepositoryProbeInto(
	slot model.RepositorySlot,
	path string,
	probe *repositoryProbe,
	waitGroup *sync.WaitGroup,
) {
	*probe = s.resolveRepositoryProbe(slot, path)
	waitGroup.Done()
}

func (s *Service) resolveRepositoryProbe(slot model.RepositorySlot, path string) repositoryProbe {
	probe := newRepositoryProbe(slot, path)
	if !probe.summary.IsConfigured {
		return probe
	}
	root, err := s.inspector.ResolveRepositoryRoot(path)
	if err != nil {
		setProbeError(&probe, err)
		return probe
	}
	probe.root = root
	return probe
}

func newRepositoryProbe(slot model.RepositorySlot, path string) repositoryProbe {
	summary := model.RepositorySummary{
		Slot:         slot,
		Path:         path,
		IsConfigured: path != "",
	}
	probe := repositoryProbe{
		summary: summary,
		status: model.TargetRepositoryStatus{
			Path: summary.Path,
		},
	}
	return probe
}

func setProbeError(probe *repositoryProbe, err error) {
	probe.root = ""
	name := probe.summary.Name
	if name == "" {
		name = model.RepositoryName(probe.summary.Path)
	}
	probe.summary.Name = name
	probe.summary.ValidationError = err.Error()
	probe.summary.IsGitRepo = false
	probe.summary.Branch = ""
	probe.summary.IsClean = false
	probe.summary.ModifiedCount = 0
	probe.summary.UntrackedCount = 0
	probe.status = model.TargetRepositoryStatus{
		Path:  probe.summary.Path,
		Name:  name,
		Error: err.Error(),
	}
}

func (s *Service) enrichState(
	cfg model.AppConfig,
	repositoryA *repositoryProbe,
	repositoryB *repositoryProbe,
) (diff.Result, error) {
	request, shouldDiff := diffRequestForDirection(cfg.Direction, *repositoryA, *repositoryB)
	result, diffErr := s.runStateTasks(repositoryA, repositoryB, request, shouldDiff)
	if !canApplyDiff(cfg.Direction, *repositoryA, *repositoryB) {
		return diff.Result{}, nil
	}
	if diffErr != nil {
		return diff.Result{}, fmt.Errorf("failed to calculate repository differences: %w", diffErr)
	}
	return result, nil
}

func (s *Service) runStateTasks(
	repositoryA *repositoryProbe,
	repositoryB *repositoryProbe,
	request diff.Request,
	shouldDiff bool,
) (diff.Result, error) {
	statusTasks := 0
	if repositoryA.root != "" {
		statusTasks++
	}
	if repositoryB.root != "" {
		statusTasks++
	}

	switch {
	case statusTasks == 0 && !shouldDiff:
		return diff.Result{}, nil
	case statusTasks == 0:
		return s.differ.Calculate(request)
	case statusTasks == 1 && !shouldDiff:
		s.enrichRepositoryStatus(activeStatusProbe(repositoryA, repositoryB))
		return diff.Result{}, nil
	case statusTasks == 1:
		var waitGroup sync.WaitGroup
		waitGroup.Add(1)
		go s.enrichRepositoryStatusInto(activeStatusProbe(repositoryA, repositoryB), &waitGroup)
		result, diffErr := s.differ.Calculate(request)
		waitGroup.Wait()
		return result, diffErr
	case !shouldDiff:
		var waitGroup sync.WaitGroup
		waitGroup.Add(1)
		go s.enrichRepositoryStatusInto(repositoryB, &waitGroup)
		s.enrichRepositoryStatus(repositoryA)
		waitGroup.Wait()
		return diff.Result{}, nil
	}

	var waitGroup sync.WaitGroup
	var result diff.Result
	var diffErr error

	waitGroup.Add(2)
	go s.enrichRepositoryStatusInto(repositoryA, &waitGroup)
	go s.enrichRepositoryStatusInto(repositoryB, &waitGroup)
	result, diffErr = s.differ.Calculate(request)
	waitGroup.Wait()
	return result, diffErr
}

func activeStatusProbe(repositoryA *repositoryProbe, repositoryB *repositoryProbe) *repositoryProbe {
	if repositoryA.root != "" {
		return repositoryA
	}
	return repositoryB
}

func (s *Service) enrichRepositoryStatusInto(probe *repositoryProbe, waitGroup *sync.WaitGroup) {
	s.enrichRepositoryStatus(probe)
	waitGroup.Done()
}

func (s *Service) enrichRepositoryStatus(probe *repositoryProbe) {
	if probe.root == "" {
		return
	}
	status, err := s.inspector.ReadTargetStatusFromRoot(probe.root)
	if err != nil {
		setProbeError(probe, err)
		return
	}
	applyProbeStatus(probe, status)
}

func applyProbeStatus(probe *repositoryProbe, status model.TargetRepositoryStatus) {
	probe.root = status.Path
	probe.summary.Path = status.Path
	probe.summary.Name = status.Name
	probe.summary.IsGitRepo = true
	probe.summary.ValidationError = ""
	probe.summary.Branch = status.Branch
	probe.summary.IsClean = status.IsClean
	probe.summary.ModifiedCount = status.ModifiedCount
	probe.summary.UntrackedCount = status.UntrackedCount
	probe.status = status
}

func diffRequestForDirection(
	direction model.Direction,
	repositoryA repositoryProbe,
	repositoryB repositoryProbe,
) (diff.Request, bool) {
	source := repositoryA
	target := repositoryB
	if direction == model.DirectionBToA {
		source = repositoryB
		target = repositoryA
	}
	if source.root == "" || target.root == "" {
		return diff.Request{}, false
	}
	return diff.Request{SourceRoot: source.root, TargetRoot: target.root}, true
}

func canApplyDiff(
	direction model.Direction,
	repositoryA repositoryProbe,
	repositoryB repositoryProbe,
) bool {
	source := repositoryA.summary
	target := repositoryB.summary
	if direction == model.DirectionBToA {
		source = repositoryB.summary
		target = repositoryA.summary
	}
	return source.IsGitRepo && target.IsGitRepo
}

func applyDiffResult(state *model.DashboardState, result diff.Result) {
	state.Differences = result.Entries
	state.Summary = result.Summary
	state.CanSync = result.Summary.Added+result.Summary.Modified+result.Summary.Deleted > 0
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
	name := selected.summary.Name
	if name == "" {
		name = model.RepositoryName(selected.summary.Path)
	}
	return model.TargetRepositoryStatus{
		Path:      selected.summary.Path,
		Name:      name,
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
