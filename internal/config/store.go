package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"RepoMirror/internal/model"
)

type Store struct {
	path string
}

func NewStore() (*Store, error) {
	path, err := resolveConfigPath()
	if err != nil {
		return nil, err
	}
	return &Store{path: path}, nil
}

func NewStoreWithPath(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() (model.AppConfig, error) {
	cfg := model.DefaultConfig()
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return model.AppConfig{}, err
	}
	return cfg.WithDefaults(), nil
}

func (s *Store) Save(cfg model.AppConfig) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(cfg.WithDefaults(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, payload, 0o644)
}

func (s *Store) Path() string {
	return s.path
}

func resolveConfigPath() (string, error) {
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "RepoMirror", "config.json"), nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "RepoMirror", "config.json"), nil
}
