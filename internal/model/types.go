package model

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	DefaultWindowWidth  = 1280
	DefaultWindowHeight = 820
)

type Direction string

const (
	DirectionAToB Direction = "A_TO_B"
	DirectionBToA Direction = "B_TO_A"
)

func ParseDirection(raw string) (Direction, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case string(DirectionAToB):
		return DirectionAToB, nil
	case string(DirectionBToA):
		return DirectionBToA, nil
	default:
		return "", fmt.Errorf("unsupported direction: %s", raw)
	}
}

func (d Direction) SourceSlot() RepositorySlot {
	if d == DirectionBToA {
		return RepositorySlotB
	}
	return RepositorySlotA
}

func (d Direction) TargetSlot() RepositorySlot {
	if d == DirectionBToA {
		return RepositorySlotA
	}
	return RepositorySlotB
}

type RepositorySlot string

const (
	RepositorySlotA RepositorySlot = "A"
	RepositorySlotB RepositorySlot = "B"
)

func ParseRepositorySlot(raw string) (RepositorySlot, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case string(RepositorySlotA):
		return RepositorySlotA, nil
	case string(RepositorySlotB):
		return RepositorySlotB, nil
	default:
		return "", fmt.Errorf("unsupported repository slot: %s", raw)
	}
}

type AppConfig struct {
	ProjectA     string    `json:"projectA"`
	ProjectB     string    `json:"projectB"`
	Direction    Direction `json:"direction"`
	WindowWidth  int       `json:"windowWidth"`
	WindowHeight int       `json:"windowHeight"`
}

func DefaultConfig() AppConfig {
	return AppConfig{
		Direction:    DirectionAToB,
		WindowWidth:  DefaultWindowWidth,
		WindowHeight: DefaultWindowHeight,
	}
}

func (cfg AppConfig) WithDefaults() AppConfig {
	next := cfg
	if next.Direction == "" {
		next.Direction = DirectionAToB
	}
	if next.WindowWidth <= 0 {
		next.WindowWidth = DefaultWindowWidth
	}
	if next.WindowHeight <= 0 {
		next.WindowHeight = DefaultWindowHeight
	}
	return next
}

func (cfg AppConfig) PathFor(slot RepositorySlot) string {
	if slot == RepositorySlotB {
		return cfg.ProjectB
	}
	return cfg.ProjectA
}

func (cfg *AppConfig) SetPath(slot RepositorySlot, path string) {
	if slot == RepositorySlotB {
		cfg.ProjectB = path
		return
	}
	cfg.ProjectA = path
}

type RepositorySummary struct {
	Slot            RepositorySlot `json:"slot"`
	Path            string         `json:"path"`
	Name            string         `json:"name"`
	IsConfigured    bool           `json:"isConfigured"`
	IsGitRepo       bool           `json:"isGitRepo"`
	ValidationError string         `json:"validationError"`
	Branch          string         `json:"branch"`
	IsClean         bool           `json:"isClean"`
	ModifiedCount   int            `json:"modifiedCount"`
	UntrackedCount  int            `json:"untrackedCount"`
}

type TargetRepositoryStatus struct {
	Path           string `json:"path"`
	Name           string `json:"name"`
	Branch         string `json:"branch"`
	IsGitRepo      bool   `json:"isGitRepo"`
	Error          string `json:"error"`
	IsClean        bool   `json:"isClean"`
	ModifiedCount  int    `json:"modifiedCount"`
	UntrackedCount int    `json:"untrackedCount"`
}

type DiffKind string

const (
	DiffKindAdded     DiffKind = "added"
	DiffKindModified  DiffKind = "modified"
	DiffKindDeleted   DiffKind = "deleted"
	DiffKindProtected DiffKind = "protected"
)

type DiffEntry struct {
	Path      string   `json:"path"`
	Kind      DiffKind `json:"kind"`
	Rule      string   `json:"rule"`
	SizeBytes int64    `json:"sizeBytes"`
}

type DiffSummary struct {
	Total     int `json:"total"`
	Added     int `json:"added"`
	Modified  int `json:"modified"`
	Deleted   int `json:"deleted"`
	Protected int `json:"protected"`
}

func BuildDiffSummary(entries []DiffEntry) DiffSummary {
	summary := DiffSummary{}
	for _, entry := range entries {
		summary.Total++
		switch entry.Kind {
		case DiffKindAdded:
			summary.Added++
		case DiffKindModified:
			summary.Modified++
		case DiffKindDeleted:
			summary.Deleted++
		case DiffKindProtected:
			summary.Protected++
		}
	}
	return summary
}

func (entry DiffEntry) IsActionable() bool {
	return entry.Kind != DiffKindProtected
}

type DashboardState struct {
	Config       AppConfig              `json:"config"`
	RepositoryA  RepositorySummary      `json:"repositoryA"`
	RepositoryB  RepositorySummary      `json:"repositoryB"`
	SourceSlot   RepositorySlot         `json:"sourceSlot"`
	TargetSlot   RepositorySlot         `json:"targetSlot"`
	Differences  []DiffEntry            `json:"differences"`
	Summary      DiffSummary            `json:"summary"`
	TargetStatus TargetRepositoryStatus `json:"targetStatus"`
	CanSync      bool                   `json:"canSync"`
}

func RepositoryName(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	clean := filepath.Clean(path)
	if clean == "." || clean == string(filepath.Separator) {
		return clean
	}
	return filepath.Base(clean)
}
