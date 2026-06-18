package diff

import (
	"path/filepath"
	"sort"

	"RepoMirror/internal/model"
	"RepoMirror/internal/platform"
)

type GitInspector interface {
	ListSyncableSourcePaths(repoPath string) ([]string, error)
	IgnoredPaths(repoPath string, paths []string) (map[string]bool, error)
}

type Service struct {
	fs  platform.FileSystem
	git GitInspector
}

type Request struct {
	SourceRoot string
	TargetRoot string
}

type Result struct {
	Entries []model.DiffEntry
	Summary model.DiffSummary
}

func NewService(fsys platform.FileSystem, gitInspector GitInspector) *Service {
	return &Service{fs: fsys, git: gitInspector}
}

func (s *Service) Calculate(request Request) (Result, error) {
	sourceFiles, err := s.git.ListSyncableSourcePaths(request.SourceRoot)
	if err != nil {
		return Result{}, err
	}
	targetFiles, err := s.fs.ListRegularFiles(request.TargetRoot)
	if err != nil {
		return Result{}, err
	}
	ignored, err := s.git.IgnoredPaths(request.TargetRoot, append(sourceFiles, targetFiles...))
	if err != nil {
		return Result{}, err
	}
	entries, err := s.collectEntries(request, sourceFiles, targetFiles, ignored)
	if err != nil {
		return Result{}, err
	}
	sort.Slice(entries, func(i int, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return Result{Entries: entries, Summary: model.BuildDiffSummary(entries)}, nil
}

func (s *Service) collectEntries(
	request Request,
	sourceFiles []string,
	targetFiles []string,
	ignored map[string]bool,
) ([]model.DiffEntry, error) {
	entries := make([]model.DiffEntry, 0)
	sourceSet := make(map[string]bool, len(sourceFiles))
	for _, relPath := range sourceFiles {
		sourceSet[relPath] = true
		entry, err := s.diffFromSourceFile(request, relPath, ignored)
		if err != nil {
			return nil, err
		}
		if entry != nil {
			entries = append(entries, *entry)
		}
	}
	for _, relPath := range targetFiles {
		if sourceSet[relPath] || ignored[relPath] || isProtected(relPath) {
			continue
		}
		entries = append(entries, model.DiffEntry{Path: relPath, Kind: model.DiffKindDeleted})
	}
	return entries, nil
}

func (s *Service) diffFromSourceFile(
	request Request,
	relPath string,
	ignored map[string]bool,
) (*model.DiffEntry, error) {
	if ignored[relPath] || isProtected(relPath) {
		return nil, nil
	}
	targetPath := fullPath(request.TargetRoot, relPath)
	exists, err := s.fs.Exists(targetPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return &model.DiffEntry{Path: relPath, Kind: model.DiffKindAdded}, nil
	}
	equal, err := s.fs.FilesEqual(fullPath(request.SourceRoot, relPath), targetPath)
	if err != nil {
		return nil, err
	}
	if equal {
		return nil, nil
	}
	return &model.DiffEntry{Path: relPath, Kind: model.DiffKindModified}, nil
}

func fullPath(root string, relPath string) string {
	return filepath.Join(root, filepath.FromSlash(relPath))
}
