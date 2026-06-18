package app

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"RepoMirror/internal/config"
	"RepoMirror/internal/diff"
	"RepoMirror/internal/gitops"
	"RepoMirror/internal/model"
	"RepoMirror/internal/syncer"
)

type Service struct {
	mu           sync.Mutex
	ctx          context.Context
	store        *config.Store
	selector     DirectorySelector
	inspector    *gitops.Service
	differ       *diff.Service
	synchronizer *syncer.Service
	config       model.AppConfig
}

func NewService(
	store *config.Store,
	selector DirectorySelector,
	inspector *gitops.Service,
	differ *diff.Service,
	synchronizer *syncer.Service,
	initialConfig model.AppConfig,
) *Service {
	return &Service{
		store:        store,
		selector:     selector,
		inspector:    inspector,
		differ:       differ,
		synchronizer: synchronizer,
		config:       initialConfig.WithDefaults(),
	}
}

func (s *Service) Startup(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
}

func (s *Service) Shutdown(ctx context.Context) {
	cfg := s.currentConfig()
	width, height := runtime.WindowGetSize(ctx)
	cfg.WindowWidth = width
	cfg.WindowHeight = height
	if err := s.saveConfig(cfg); err != nil {
		runtime.LogErrorf(ctx, "persist config on shutdown: %v", err)
	}
}

func (s *Service) LoadState() (model.DashboardState, error) {
	return s.buildState(s.currentConfig())
}

func (s *Service) Refresh() (model.DashboardState, error) {
	return s.buildState(s.currentConfig())
}

func (s *Service) SaveConfig() (model.DashboardState, error) {
	cfg := s.currentConfig()
	if err := s.saveConfig(cfg); err != nil {
		return model.DashboardState{}, err
	}
	return s.buildState(cfg)
}

func (s *Service) currentConfig() model.AppConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.config.WithDefaults()
}

func (s *Service) updateConfig(mutator func(*model.AppConfig)) (model.AppConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.config
	mutator(&next)
	next = next.WithDefaults()
	if err := s.saveConfig(next); err != nil {
		return model.AppConfig{}, err
	}
	s.config = next
	return s.config, nil
}

func (s *Service) saveConfig(cfg model.AppConfig) error {
	return s.store.Save(cfg.WithDefaults())
}
