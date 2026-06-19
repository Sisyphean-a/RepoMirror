package main

import (
	"context"

	appsvc "RepoMirror/internal/app"
	"RepoMirror/internal/model"
)

type App struct {
	service *appsvc.Service
}

func NewApp(service *appsvc.Service) *App {
	return &App{service: service}
}

func (a *App) startup(ctx context.Context) {
	a.service.Startup(ctx)
}

func (a *App) shutdown(ctx context.Context) {
	a.service.Shutdown(ctx)
}

func (a *App) LoadState() (model.DashboardState, error) {
	return a.service.LoadState()
}

func (a *App) Refresh() (model.DashboardState, error) {
	return a.service.Refresh()
}

func (a *App) SelectRepository(slot string) (model.DashboardState, error) {
	return a.service.SelectRepository(slot)
}

func (a *App) SwapRepositories() (model.DashboardState, error) {
	return a.service.SwapRepositories()
}

func (a *App) SetDirection(direction string) (model.DashboardState, error) {
	return a.service.SetDirection(direction)
}

func (a *App) SaveConfig() (model.DashboardState, error) {
	return a.service.SaveConfig()
}

func (a *App) SyncRepositories() (model.DashboardState, error) {
	return a.service.SyncRepositories()
}

func (a *App) CommitTarget(message string) (model.DashboardState, error) {
	return a.service.CommitTarget(message)
}

func (a *App) GenerateCommitMessage() (string, error) {
	return a.service.GenerateCommitMessage()
}

func (a *App) SetAICommitAPIKey(apiKey string) (model.DashboardState, error) {
	return a.service.SetAICommitAPIKey(apiKey)
}

func (a *App) PushTarget() (model.DashboardState, error) {
	return a.service.PushTarget()
}
