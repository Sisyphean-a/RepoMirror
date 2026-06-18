package app

import (
	"context"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type DirectorySelector interface {
	Open(ctx context.Context, defaultDirectory string, title string) (string, error)
}

type WailsDirectorySelector struct{}

func NewWailsDirectorySelector() *WailsDirectorySelector {
	return &WailsDirectorySelector{}
}

func (selector *WailsDirectorySelector) Open(
	ctx context.Context,
	defaultDirectory string,
	title string,
) (string, error) {
	options := runtime.OpenDialogOptions{
		Title:                title,
		DefaultDirectory:     existingDirectory(defaultDirectory),
		CanCreateDirectories: true,
	}
	return runtime.OpenDirectoryDialog(ctx, options)
}

func existingDirectory(path string) string {
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return ""
	}
	return path
}
