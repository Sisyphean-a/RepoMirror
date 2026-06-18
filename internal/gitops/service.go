package gitops

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"RepoMirror/internal/model"
)

type Runner interface {
	Run(repoPath string, input []byte, args ...string) ([]byte, error)
}

type Service struct {
	runner Runner
}

func NewService(runner Runner) *Service {
	return &Service{runner: runner}
}

func NewExecRunner() Runner {
	return execRunner{}
}

func (s *Service) ResolveRepositoryRoot(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("repository path is empty")
	}
	output, err := s.runner.Run(path, nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to detect git repository: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *Service) ListSyncableSourcePaths(repoPath string) ([]string, error) {
	root, err := s.ResolveRepositoryRoot(repoPath)
	if err != nil {
		return nil, err
	}
	output, err := s.runner.Run(root, nil, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("failed to list source files: %w", err)
	}
	paths := make([]string, 0)
	for _, relPath := range splitNullSeparated(output) {
		if isProtectedPath(relPath) {
			continue
		}
		fullPath := filepath.Join(root, filepath.FromSlash(relPath))
		info, statErr := os.Stat(fullPath)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return nil, statErr
		}
		if info.IsDir() {
			continue
		}
		paths = append(paths, relPath)
	}
	sort.Strings(paths)
	return dedupe(paths), nil
}

func (s *Service) IgnoredPaths(repoPath string, paths []string) (map[string]bool, error) {
	root, err := s.ResolveRepositoryRoot(repoPath)
	if err != nil {
		return nil, err
	}
	input := buildNullSeparatedInput(paths)
	if len(input) == 0 {
		return map[string]bool{}, nil
	}
	output, err := s.runner.Run(root, input, "check-ignore", "--stdin", "-z", "--no-index")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return map[string]bool{}, nil
		}
		return nil, fmt.Errorf("failed to evaluate target ignore rules: %w", err)
	}
	ignored := make(map[string]bool, len(paths))
	for _, relPath := range splitNullSeparated(output) {
		ignored[relPath] = true
	}
	return ignored, nil
}

func buildNullSeparatedInput(paths []string) []byte {
	buffer := bytes.NewBuffer(nil)
	for _, path := range dedupe(paths) {
		if path == "" {
			continue
		}
		buffer.WriteString(path)
		buffer.WriteByte(0)
	}
	return buffer.Bytes()
}

func splitNullSeparated(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	parts := strings.Split(string(raw), "\x00")
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized := filepath.ToSlash(strings.TrimSpace(part))
		if normalized != "" {
			paths = append(paths, normalized)
		}
	}
	return paths
}

func isProtectedPath(relPath string) bool {
	if strings.EqualFold(filepath.Base(relPath), ".gitignore") {
		return true
	}
	for _, part := range strings.Split(filepath.ToSlash(relPath), "/") {
		if strings.EqualFold(part, ".git") {
			return true
		}
	}
	return false
}

func dedupe(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		if seen[path] {
			continue
		}
		seen[path] = true
		unique = append(unique, path)
	}
	return unique
}

type execRunner struct{}

func (execRunner) Run(repoPath string, input []byte, args ...string) ([]byte, error) {
	commandArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", commandArgs...)
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func buildTargetStatus(repoPath string, branch string, statusOutput []byte) model.TargetRepositoryStatus {
	status := model.TargetRepositoryStatus{
		Path:      repoPath,
		Name:      model.RepositoryName(repoPath),
		Branch:    branch,
		IsGitRepo: true,
	}
	for _, line := range strings.Split(strings.TrimSpace(string(statusOutput)), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "??") {
			status.UntrackedCount++
			continue
		}
		status.ModifiedCount++
	}
	status.IsClean = status.ModifiedCount == 0 && status.UntrackedCount == 0
	return status
}
