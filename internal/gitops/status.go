package gitops

import (
	"fmt"
	"strings"

	"RepoMirror/internal/model"
)

func (s *Service) ReadTargetStatus(repoPath string) (model.TargetRepositoryStatus, error) {
	root, err := s.ResolveRepositoryRoot(repoPath)
	if err != nil {
		return model.TargetRepositoryStatus{}, err
	}
	branchOutput, err := s.runner.Run(root, nil, "branch", "--show-current")
	if err != nil {
		return model.TargetRepositoryStatus{}, fmt.Errorf("failed to read branch: %w", err)
	}
	statusOutput, err := s.runner.Run(root, nil, "status", "--short")
	if err != nil {
		return model.TargetRepositoryStatus{}, fmt.Errorf("failed to read target status: %w", err)
	}
	branch := strings.TrimSpace(string(branchOutput))
	if branch == "" {
		branch = "HEAD"
	}
	return buildTargetStatus(root, branch, statusOutput), nil
}

func (s *Service) Commit(repoPath string, message string) error {
	root, err := s.ResolveRepositoryRoot(repoPath)
	if err != nil {
		return err
	}
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return fmt.Errorf("commit message is required")
	}
	if _, err := s.runner.Run(root, nil, "add", "-A"); err != nil {
		return fmt.Errorf("failed to stage target repository changes: %w", err)
	}
	if _, err := s.runner.Run(root, nil, "commit", "-m", trimmedMessage); err != nil {
		return fmt.Errorf("failed to commit target repository changes: %w", err)
	}
	return nil
}

func (s *Service) Push(repoPath string) error {
	root, err := s.ResolveRepositoryRoot(repoPath)
	if err != nil {
		return err
	}
	if _, err := s.runner.Run(root, nil, "push"); err != nil {
		return fmt.Errorf("failed to push target repository changes: %w", err)
	}
	return nil
}
