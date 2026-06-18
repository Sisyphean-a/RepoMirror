package app

import (
	"fmt"
	"strings"

	"RepoMirror/internal/model"
	"RepoMirror/internal/syncer"
)

func (s *Service) SelectRepository(slot string) (model.DashboardState, error) {
	repoSlot, err := model.ParseRepositorySlot(slot)
	if err != nil {
		return model.DashboardState{}, err
	}
	cfg := s.currentConfig()
	selectedPath, err := s.selector.Open(s.ctx, cfg.PathFor(repoSlot), fmt.Sprintf("Select Repository %s", repoSlot))
	if err != nil || selectedPath == "" {
		if err != nil {
			return model.DashboardState{}, err
		}
		return s.buildState(cfg)
	}
	root, err := s.inspector.ResolveRepositoryRoot(selectedPath)
	if err != nil {
		return model.DashboardState{}, err
	}
	cfg = s.updateConfig(func(next *model.AppConfig) {
		next.SetPath(repoSlot, root)
	})
	return s.buildState(cfg)
}

func (s *Service) SwapRepositories() (model.DashboardState, error) {
	cfg := s.updateConfig(func(next *model.AppConfig) {
		next.ProjectA, next.ProjectB = next.ProjectB, next.ProjectA
	})
	return s.buildState(cfg)
}

func (s *Service) SetDirection(direction string) (model.DashboardState, error) {
	parsed, err := model.ParseDirection(direction)
	if err != nil {
		return model.DashboardState{}, err
	}
	cfg := s.updateConfig(func(next *model.AppConfig) {
		next.Direction = parsed
	})
	return s.buildState(cfg)
}

func (s *Service) SyncRepositories() (model.DashboardState, error) {
	sourceRoot, targetRoot, err := s.resolveSyncRoots(s.currentConfig())
	if err != nil {
		return model.DashboardState{}, err
	}
	if err := s.synchronizer.Sync(syncer.Request{SourceRoot: sourceRoot, TargetRoot: targetRoot}); err != nil {
		return model.DashboardState{}, err
	}
	return s.buildState(s.currentConfig())
}

func (s *Service) CommitTarget(message string) (model.DashboardState, error) {
	targetRoot, err := s.resolveTargetRoot(s.currentConfig())
	if err != nil {
		return model.DashboardState{}, err
	}
	if strings.TrimSpace(message) == "" {
		return model.DashboardState{}, fmt.Errorf("commit message is required")
	}
	if err := s.inspector.Commit(targetRoot, message); err != nil {
		return model.DashboardState{}, err
	}
	return s.buildState(s.currentConfig())
}

func (s *Service) PushTarget() (model.DashboardState, error) {
	targetRoot, err := s.resolveTargetRoot(s.currentConfig())
	if err != nil {
		return model.DashboardState{}, err
	}
	if err := s.inspector.Push(targetRoot); err != nil {
		return model.DashboardState{}, err
	}
	return s.buildState(s.currentConfig())
}

func (s *Service) resolveSyncRoots(cfg model.AppConfig) (string, string, error) {
	sourceRoot, err := s.resolveRepository(cfg.PathFor(cfg.Direction.SourceSlot()), "source")
	if err != nil {
		return "", "", err
	}
	targetRoot, err := s.resolveRepository(cfg.PathFor(cfg.Direction.TargetSlot()), "target")
	if err != nil {
		return "", "", err
	}
	return sourceRoot, targetRoot, nil
}

func (s *Service) resolveTargetRoot(cfg model.AppConfig) (string, error) {
	return s.resolveRepository(cfg.PathFor(cfg.Direction.TargetSlot()), "target")
}

func (s *Service) resolveRepository(path string, label string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%s repository is not configured", label)
	}
	root, err := s.inspector.ResolveRepositoryRoot(path)
	if err != nil {
		return "", fmt.Errorf("%s repository is invalid: %w", label, err)
	}
	return root, nil
}
