package config

import (
	"path/filepath"
	"testing"

	"RepoMirror/internal/model"
)

func TestStoreSaveAndLoad(t *testing.T) {
	store := NewStoreWithPath(filepath.Join(t.TempDir(), "config.json"))
	expected := model.AppConfig{
		ProjectA:     "E:/repo-a",
		ProjectB:     "E:/repo-b",
		Direction:    model.DirectionBToA,
		WindowWidth:  1440,
		WindowHeight: 900,
	}

	if err := store.Save(expected); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	actual, err := store.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if actual != expected {
		t.Fatalf("unexpected config: %#v", actual)
	}
}
