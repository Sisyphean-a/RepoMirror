package diff

import (
	"path/filepath"
	"sort"

	"RepoMirror/internal/model"
	"RepoMirror/internal/platform"
)

type GitInspector interface {
	ListSyncableSourcePaths(repoPath string) ([]string, error)
	IgnoredPaths(repoPath string, paths []string) (map[string]string, error)
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
	ignored map[string]string,
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
		if sourceSet[relPath] || isProtected(relPath) {
			continue
		}
		if rule, isIgnored := ignored[relPath]; isIgnored {
			entry, err := s.protectedEntry(fullPath(request.TargetRoot, relPath), relPath, rule)
			if err != nil {
				return nil, err
			}
			entries = append(entries, entry)
			continue
		}
		entries = append(entries, model.DiffEntry{Path: relPath, Kind: model.DiffKindDeleted})
	}
	return entries, nil
}

func (s *Service) diffFromSourceFile(
	request Request,
	relPath string,
	ignored map[string]string,
) (*model.DiffEntry, error) {
	if isProtected(relPath) {
		return nil, nil
	}
	if rule, isIgnored := ignored[relPath]; isIgnored {
		entry, err := s.protectedEntry(fullPath(request.SourceRoot, relPath), relPath, rule)
		if err != nil {
			return nil, err
		}
		return &entry, nil
	}
	targetPath := fullPath(request.TargetRoot, relPath)
	exists, err := s.fs.Exists(targetPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return s.sizedEntry(fullPath(request.SourceRoot, relPath), relPath, model.DiffKindAdded)
	}
	equal, err := s.fs.FilesEqual(fullPath(request.SourceRoot, relPath), targetPath)
	if err != nil {
		return nil, err
	}
	if equal {
		return nil, nil
	}
	return s.sizedEntry(fullPath(request.SourceRoot, relPath), relPath, model.DiffKindModified)
}

func fullPath(root string, relPath string) string {
	return filepath.Join(root, filepath.FromSlash(relPath))
}

func (s *Service) sizedEntry(path string, relPath string, kind model.DiffKind) (*model.DiffEntry, error) {
	sizeBytes, err := s.fs.FileSize(path)
	if err != nil {
		return nil, err
	}
	return &model.DiffEntry{Path: relPath, Kind: kind, SizeBytes: sizeBytes}, nil
}

func (s *Service) protectedEntry(path string, relPath string, rule string) (model.DiffEntry, error) {
	sizeBytes, err := s.fs.FileSize(path)
	if err != nil {
		return model.DiffEntry{}, err
	}
	return model.DiffEntry{
		Path:      relPath,
		Kind:      model.DiffKindProtected,
		Rule:      rule,
		SizeBytes: sizeBytes,
	}, nil
}
