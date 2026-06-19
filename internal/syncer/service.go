package syncer

import (
	"path/filepath"

	"RepoMirror/internal/diff"
	"RepoMirror/internal/model"
	"RepoMirror/internal/platform"
)

type DifferencePlanner interface {
	Calculate(request diff.Request) (diff.Result, error)
}

type Service struct {
	fs     platform.FileSystem
	differ DifferencePlanner
}

type Request struct {
	SourceRoot string
	TargetRoot string
}

func NewService(fsys platform.FileSystem, differ DifferencePlanner) *Service {
	return &Service{fs: fsys, differ: differ}
}

func (s *Service) Sync(request Request) error {
	result, err := s.differ.Calculate(diff.Request(request))
	if err != nil {
		return err
	}
	if err := s.applyCopies(request, result.Entries); err != nil {
		return err
	}
	return s.applyDeletes(request, result.Entries)
}

func (s *Service) applyCopies(request Request, entries []model.DiffEntry) error {
	for _, entry := range entries {
		if entry.Kind != model.DiffKindAdded && entry.Kind != model.DiffKindModified {
			continue
		}
		if err := s.copyFile(request, entry.Path); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) applyDeletes(request Request, entries []model.DiffEntry) error {
	for _, entry := range entries {
		if entry.Kind != model.DiffKindDeleted {
			continue
		}
		targetPath := joinPath(request.TargetRoot, entry.Path)
		if err := s.fs.Remove(targetPath); err != nil {
			return err
		}
		if err := s.fs.RemoveEmptyParents(request.TargetRoot, targetPath); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) copyFile(request Request, relPath string) error {
	sourcePath := joinPath(request.SourceRoot, relPath)
	targetPath := joinPath(request.TargetRoot, relPath)
	return s.fs.CopyFile(sourcePath, targetPath)
}

func joinPath(root string, relPath string) string {
	return filepath.Join(root, filepath.FromSlash(relPath))
}
