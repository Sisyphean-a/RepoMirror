package gitops

import (
	"fmt"
	"strings"
	"sync"

	"RepoMirror/internal/model"
)

func (s *Service) ReadTargetStatus(repoPath string) (model.TargetRepositoryStatus, error) {
	var waitGroup sync.WaitGroup
	var root string
	var rootErr error
	var statusOutput []byte
	var statusErr error

	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		root, rootErr = s.ResolveRepositoryRoot(repoPath)
	}()
	go func() {
		defer waitGroup.Done()
		statusOutput, statusErr = s.runner.Run(repoPath, nil, "status", "--porcelain=2", "--branch")
	}()
	waitGroup.Wait()

	if rootErr != nil {
		return model.TargetRepositoryStatus{}, rootErr
	}
	if statusErr != nil {
		return model.TargetRepositoryStatus{}, fmt.Errorf("failed to read target status: %w", statusErr)
	}
	return buildTargetStatus(root, statusOutput), nil
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
