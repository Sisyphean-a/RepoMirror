package main

import (
	"embed"
	"fmt"

	appsvc "RepoMirror/internal/app"
	"RepoMirror/internal/config"
	"RepoMirror/internal/diff"
	"RepoMirror/internal/gitops"
	"RepoMirror/internal/platform"
	"RepoMirror/internal/syncer"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	store, err := config.NewStore()
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}
	initialConfig, err := store.Load()
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}

	fileSystem := platform.NewOSFileSystem()
	inspector := gitops.NewService(gitops.NewExecRunner())
	differ := diff.NewService(fileSystem, inspector)
	synchronizer := syncer.NewService(fileSystem, differ)
	service := appsvc.NewService(
		store,
		appsvc.NewWailsDirectorySelector(),
		inspector,
		differ,
		synchronizer,
		initialConfig,
	)
	app := NewApp(service)

	err = wails.Run(&options.App{
		Title:     "RepoMirror",
		Width:     initialConfig.WindowWidth,
		Height:    initialConfig.WindowHeight,
		MinWidth:  1160,
		MinHeight: 760,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 30, B: 34, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}
