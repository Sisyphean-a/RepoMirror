package gitops

import (
	"bytes"
	"errors"
	"fmt"
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
	return s.ListSyncableSourcePathsFromRoot(root)
}

func (s *Service) ListSyncableSourcePathsFromRoot(root string) ([]string, error) {
	output, err := s.runner.Run(root, nil, "ls-files", "--cached", "--others", "--exclude-standard", "--deduplicate", "-z")
	if err != nil {
		return nil, fmt.Errorf("failed to list source files: %w", err)
	}
	deletedPaths, err := s.deletedPathSetFromRoot(root)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for _, relPath := range splitNullSeparated(output) {
		if isProtectedPath(relPath) || deletedPaths[relPath] {
			continue
		}
		paths = append(paths, relPath)
	}
	sort.Strings(paths)
	return paths, nil
}

func (s *Service) IgnoredPaths(repoPath string, paths []string) (map[string]string, error) {
	root, err := s.ResolveRepositoryRoot(repoPath)
	if err != nil {
		return nil, err
	}
	return s.IgnoredPathsFromRoot(root, paths)
}

func (s *Service) IgnoredPathsFromRoot(root string, pathGroups ...[]string) (map[string]string, error) {
	input := buildLineSeparatedInput(pathGroups...)
	if len(input) == 0 {
		return map[string]string{}, nil
	}
	output, err := s.runner.Run(root, input, "check-ignore", "-v", "--stdin", "--no-index")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("failed to evaluate target ignore rules: %w", err)
	}
	ignored := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		relPath, rule := parseIgnoredPathRule(line)
		if relPath == "" {
			continue
		}
		ignored[relPath] = rule
	}
	return ignored, nil
}

func buildLineSeparatedInput(pathGroups ...[]string) []byte {
	buffer := bytes.NewBuffer(nil)
	seen := make(map[string]struct{})
	for _, paths := range pathGroups {
		for _, path := range paths {
			if path == "" {
				continue
			}
			if _, exists := seen[path]; exists {
				continue
			}
			seen[path] = struct{}{}
			buffer.WriteString(path)
			buffer.WriteByte('\n')
		}
	}
	return buffer.Bytes()
}

func splitNullSeparated(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	paths := make([]string, 0, 32)
	for start := 0; start < len(raw); {
		end := bytes.IndexByte(raw[start:], 0)
		if end == -1 {
			end = len(raw) - start
		}
		part := bytes.TrimSpace(raw[start : start+end])
		if len(part) > 0 {
			paths = append(paths, filepath.ToSlash(string(part)))
		}
		if start+end >= len(raw) {
			break
		}
		start += end + 1
	}
	return paths
}

func parseIgnoredPathRule(line string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
	if len(parts) != 2 {
		return "", ""
	}
	meta := parts[0]
	path := filepath.ToSlash(strings.TrimSpace(parts[1]))
	patternStart := strings.LastIndex(meta, ":")
	if patternStart == -1 || patternStart == len(meta)-1 {
		return path, "ignore-protected"
	}
	return path, ignoredRuleLabel(meta[patternStart+1:])
}

func ignoredRuleLabel(pattern string) string {
	lower := strings.ToLower(strings.TrimSpace(pattern))
	switch {
	case strings.Contains(lower, ".env"):
		return "env-protected"
	case strings.Contains(lower, ".yaml"), strings.Contains(lower, ".yml"), strings.Contains(lower, "config"):
		return "cfg-protected"
	case strings.Contains(lower, "secret"), strings.Contains(lower, "key"):
		return "secret-protected"
	default:
		return "ignore-protected"
	}
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

func (s *Service) deletedPathSetFromRoot(root string) (map[string]bool, error) {
	output, err := s.runner.Run(root, nil, "ls-files", "--deleted", "-z")
	if err != nil {
		return nil, fmt.Errorf("failed to list deleted source files: %w", err)
	}
	paths := make(map[string]bool)
	for _, relPath := range splitNullSeparated(output) {
		paths[relPath] = true
	}
	return paths, nil
}

type execRunner struct{}

func (execRunner) Run(repoPath string, input []byte, args ...string) ([]byte, error) {
	commandArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", commandArgs...)
	cmd.Stdin = bytes.NewReader(input)
	hideConsoleWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func buildTargetStatus(repoPath string, statusOutput []byte) model.TargetRepositoryStatus {
	status := model.TargetRepositoryStatus{
		Path:      repoPath,
		Name:      model.RepositoryName(repoPath),
		Branch:    "HEAD",
		IsGitRepo: true,
	}
	for len(statusOutput) > 0 {
		line, rest, found := bytes.Cut(statusOutput, []byte{'\n'})
		statusOutput = rest
		if !found {
			statusOutput = nil
		}
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if branch, ok := parseBranchHead(trimmed); ok {
			status.Branch = branch
			continue
		}
		if trimmed[0] == '#' {
			continue
		}
		if bytes.HasPrefix(trimmed, []byte("? ")) {
			status.UntrackedCount++
			continue
		}
		status.ModifiedCount++
	}
	status.IsClean = status.ModifiedCount == 0 && status.UntrackedCount == 0
	return status
}

func parseBranchHead(line []byte) (string, bool) {
	const prefix = "# branch.head "
	if !bytes.HasPrefix(line, []byte(prefix)) {
		return "", false
	}
	branch := strings.TrimSpace(string(bytes.TrimPrefix(line, []byte(prefix))))
	if branch == "" || branch == "(detached)" {
		return "HEAD", true
	}
	return branch, true
}
