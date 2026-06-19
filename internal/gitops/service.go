package gitops

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"RepoMirror/internal/model"
)

type Runner interface {
	Run(repoPath string, input []byte, args ...string) ([]byte, error)
}

type Service struct {
	runner Runner
	mu     sync.RWMutex
	roots  map[string]string
}

var inputBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 64*1024)
	},
}

func NewService(runner Runner) *Service {
	return &Service{runner: runner, roots: make(map[string]string)}
}

func NewExecRunner() Runner {
	return execRunner{}
}

func (s *Service) ResolveRepositoryRoot(path string) (string, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return "", fmt.Errorf("repository path is empty")
	}
	cacheKey := filepath.Clean(trimmedPath)
	if root, ok := s.cachedRoot(cacheKey); ok {
		return root, nil
	}
	output, err := s.runner.Run(trimmedPath, nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to detect git repository: %w", err)
	}
	root := filepath.Clean(strings.TrimSpace(string(output)))
	s.rememberRoot(cacheKey, root)
	return root, nil
}

func (s *Service) ListSyncableSourcePaths(repoPath string) ([]string, error) {
	root, err := s.ResolveRepositoryRoot(repoPath)
	if err != nil {
		return nil, err
	}
	return s.ListSyncableSourcePathsFromRoot(root)
}

func (s *Service) ListSyncableSourcePathsFromRoot(root string) ([]string, error) {
	output, err := s.runner.Run(root, nil, "ls-files", "-t", "--cached", "--others", "--deleted", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("failed to list source files: %w", err)
	}
	candidates, deleted, candidatesSorted, deletedSorted := collectSyncablePaths(output)
	if !candidatesSorted {
		sort.Strings(candidates)
	}
	if !deletedSorted {
		sort.Strings(deleted)
	}
	return compactSortedPaths(candidates, deleted), nil
}

func (s *Service) IgnoredPaths(repoPath string, paths []string) (map[string]string, error) {
	root, err := s.ResolveRepositoryRoot(repoPath)
	if err != nil {
		return nil, err
	}
	return s.IgnoredPathsFromRoot(root, paths)
}

func (s *Service) IgnoredPathSetFromRoot(root string, pathGroups ...[]string) (map[string]struct{}, error) {
	input := borrowInputBuffer()
	input = buildLineSeparatedInput(input, pathGroups...)
	defer releaseInputBuffer(input)
	return s.ignoredPathSetFromInput(root, input)
}

func (s *Service) IgnoredPathSetFromRootSorted(root string, paths []string) (map[string]struct{}, error) {
	input := borrowInputBuffer()
	input = buildSingleGroupInputWithoutDedup(input, paths, estimateSingleGroupBytes(paths))
	defer releaseInputBuffer(input)
	return s.ignoredPathSetFromInput(root, input)
}

func (s *Service) ignoredPathSetFromInput(root string, input []byte) (map[string]struct{}, error) {
	if len(input) == 0 {
		return nil, nil
	}
	output, err := s.runner.Run(root, input, "check-ignore", "--stdin", "--no-index")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to evaluate target ignore rules: %w", err)
	}
	return parseIgnoredPathSet(output), nil
}

func (s *Service) IgnoredPathsFromRoot(root string, pathGroups ...[]string) (map[string]string, error) {
	input := borrowInputBuffer()
	input = buildLineSeparatedInput(input, pathGroups...)
	defer releaseInputBuffer(input)
	if len(input) == 0 {
		return nil, nil
	}
	output, err := s.runner.Run(root, input, "check-ignore", "-v", "--stdin", "--no-index")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to evaluate target ignore rules: %w", err)
	}
	return parseIgnoredPaths(output), nil
}

func buildLineSeparatedInput(buffer []byte, pathGroups ...[]string) []byte {
	if len(pathGroups) == 1 {
		return buildSingleGroupInput(buffer, pathGroups[0])
	}
	totalPaths, totalBytes := estimateInputSize(pathGroups)
	buffer = growBuffer(buffer, totalBytes)
	seen := make(map[string]struct{}, totalPaths)
	for _, paths := range pathGroups {
		for _, path := range paths {
			if path == "" {
				continue
			}
			if _, exists := seen[path]; exists {
				continue
			}
			seen[path] = struct{}{}
			buffer = append(buffer, path...)
			buffer = append(buffer, '\n')
		}
	}
	return buffer
}

func buildSingleGroupInput(buffer []byte, paths []string) []byte {
	if len(paths) == 0 {
		return buffer[:0]
	}
	totalBytes, isAscending := analyzeSingleGroupPaths(paths)
	if isAscending {
		return buildSingleGroupInputWithoutDedup(buffer, paths, totalBytes)
	}
	return buildSingleGroupInputDedup(buffer, paths, totalBytes)
}

func buildSingleGroupInputWithoutDedup(buffer []byte, paths []string, totalBytes int) []byte {
	buffer = growBuffer(buffer, totalBytes)
	for _, path := range paths {
		if path == "" {
			continue
		}
		buffer = append(buffer, path...)
		buffer = append(buffer, '\n')
	}
	return buffer
}

func buildSingleGroupInputDedup(buffer []byte, paths []string, totalBytes int) []byte {
	buffer = growBuffer(buffer, totalBytes)
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		buffer = append(buffer, path...)
		buffer = append(buffer, '\n')
	}
	return buffer
}

func growBuffer(buffer []byte, targetCap int) []byte {
	if cap(buffer) < targetCap {
		return make([]byte, 0, targetCap)
	}
	return buffer[:0]
}

func analyzeSingleGroupPaths(paths []string) (int, bool) {
	previous := ""
	totalBytes := 0
	for _, path := range paths {
		totalBytes += len(path) + 1
		if path == "" {
			return totalBytes, false
		}
		if previous != "" && path <= previous {
			return totalBytes, false
		}
		previous = path
	}
	return totalBytes, true
}

func estimateInputSize(pathGroups [][]string) (int, int) {
	totalPaths := 0
	totalBytes := 0
	for _, paths := range pathGroups {
		totalPaths += len(paths)
		for _, path := range paths {
			totalBytes += len(path) + 1
		}
	}
	return totalPaths, totalBytes
}

func estimateSingleGroupBytes(paths []string) int {
	totalBytes := 0
	for _, path := range paths {
		totalBytes += len(path) + 1
	}
	return totalBytes
}

func borrowInputBuffer() []byte {
	return inputBufferPool.Get().([]byte)
}

func releaseInputBuffer(buffer []byte) {
	const maxRetainedInputCap = 512 * 1024
	if cap(buffer) > maxRetainedInputCap {
		return
	}
	inputBufferPool.Put(buffer[:0])
}

func (s *Service) cachedRoot(path string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	root, ok := s.roots[path]
	return root, ok
}

func (s *Service) rememberRoot(path string, root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roots[path] = root
	s.roots[root] = root
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
	cleanRepoPath := filepath.Clean(repoPath)
	status := model.TargetRepositoryStatus{
		Path:      cleanRepoPath,
		Name:      repositoryNameFromCleanPath(cleanRepoPath),
		Branch:    "HEAD",
		IsGitRepo: true,
	}
	for start := 0; start < len(statusOutput); {
		end := bytes.IndexByte(statusOutput[start:], '\n')
		if end == -1 {
			end = len(statusOutput) - start
		}
		line := trimTrailingCarriageReturn(statusOutput[start : start+end])
		if len(line) != 0 {
			if line[0] == '#' {
				if branch, ok := parseBranchHead(line); ok {
					status.Branch = branch
				}
			} else if isUntrackedStatusLine(line) {
				status.UntrackedCount++
			} else {
				status.ModifiedCount++
			}
		}
		if start+end >= len(statusOutput) {
			break
		}
		start += end + 1
	}
	status.IsClean = status.ModifiedCount == 0 && status.UntrackedCount == 0
	return status
}

func parseBranchHead(line []byte) (string, bool) {
	if !bytes.HasPrefix(line, branchHeadPrefixBytes) {
		return "", false
	}
	branch := bytesToStringView(line[len(branchHeadPrefixBytes):])
	if branch == "" || branch == "(detached)" {
		return "HEAD", true
	}
	return branch, true
}

var branchHeadPrefixBytes = []byte("# branch.head ")

func isUntrackedStatusLine(line []byte) bool {
	return len(line) >= 2 && line[0] == '?' && line[1] == ' '
}

func repositoryNameFromCleanPath(cleanPath string) string {
	if cleanPath == "" || cleanPath == "." || cleanPath == string(filepath.Separator) {
		return cleanPath
	}
	return filepath.Base(cleanPath)
}

func parseTaggedPath(item string) (string, string) {
	if len(item) < 3 || item[1] != ' ' {
		return "", filepath.ToSlash(strings.TrimSpace(item))
	}
	return item[:1], filepath.ToSlash(strings.TrimSpace(item[2:]))
}

func compactSortedPaths(candidates []string, deleted []string) []string {
	if len(candidates) == 0 {
		return nil
	}
	writeIndex := 0
	lastPath := ""
	deletedIndex := 0
	for _, relPath := range candidates {
		if relPath == lastPath {
			continue
		}
		lastPath = relPath
		for deletedIndex < len(deleted) && deleted[deletedIndex] < relPath {
			deletedIndex++
		}
		if deletedIndex < len(deleted) && deleted[deletedIndex] == relPath {
			continue
		}
		candidates[writeIndex] = relPath
		writeIndex++
	}
	return candidates[:writeIndex]
}
