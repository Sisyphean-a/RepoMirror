package diff

import (
	"path/filepath"
	"runtime"
	"sync"

	"RepoMirror/internal/model"
	"RepoMirror/internal/platform"
)

type GitInspector interface {
	ListSyncableSourcePathsFromRoot(root string) ([]string, error)
	IgnoredPathsFromRoot(root string, pathGroups ...[]string) (map[string]string, error)
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

type compareJob struct {
	index        int
	relPath      string
	targetExists bool
}

func NewService(fsys platform.FileSystem, gitInspector GitInspector) *Service {
	return &Service{fs: fsys, git: gitInspector}
}

func (s *Service) Calculate(request Request) (Result, error) {
	sourceFiles, targetFiles, err := s.loadFiles(request)
	if err != nil {
		return Result{}, err
	}
	ignored, err := s.git.IgnoredPathsFromRoot(request.TargetRoot, sourceFiles, targetFiles)
	if err != nil {
		return Result{}, err
	}
	entries, summary, err := s.collectEntries(request, sourceFiles, targetFiles, ignored)
	if err != nil {
		return Result{}, err
	}
	return Result{Entries: entries, Summary: summary}, nil
}

func (s *Service) loadFiles(request Request) ([]string, []string, error) {
	var waitGroup sync.WaitGroup
	var sourceFiles []string
	var targetFiles []string
	var sourceErr error
	var targetErr error

	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		sourceFiles, sourceErr = s.git.ListSyncableSourcePathsFromRoot(request.SourceRoot)
	}()
	go func() {
		defer waitGroup.Done()
		targetFiles, targetErr = s.fs.ListRegularFiles(request.TargetRoot)
	}()
	waitGroup.Wait()

	if sourceErr != nil {
		return nil, nil, sourceErr
	}
	if targetErr != nil {
		return nil, nil, targetErr
	}
	return sourceFiles, targetFiles, nil
}

func (s *Service) collectEntries(
	request Request,
	sourceFiles []string,
	targetFiles []string,
	ignored map[string]string,
) ([]model.DiffEntry, model.DiffSummary, error) {
	resolved, err := s.resolveEntries(request, sourceFiles, targetFiles, ignored)
	if err != nil {
		return nil, model.DiffSummary{}, err
	}
	return compactEntriesAndSummary(resolved)
}

func (s *Service) resolveEntries(
	request Request,
	sourceFiles []string,
	targetFiles []string,
	ignored map[string]string,
) ([]*model.DiffEntry, error) {
	resolved := make([]*model.DiffEntry, len(sourceFiles)+len(targetFiles))
	resolvedCount, err := s.mergeEntries(request, sourceFiles, targetFiles, ignored, resolved)
	if err != nil {
		return nil, err
	}
	return resolved[:resolvedCount], nil
}

func (s *Service) mergeEntries(
	request Request,
	sourceFiles []string,
	targetFiles []string,
	ignored map[string]string,
	resolved []*model.DiffEntry,
) (int, error) {
	jobs, waitWorkers := s.startCompareWorkers(request, ignored, resolved, len(sourceFiles))
	resolvedCount := enqueueMergedEntries(sourceFiles, targetFiles, ignored, resolved, jobs)
	close(jobs)
	return resolvedCount, waitWorkers()
}

func (s *Service) startCompareWorkers(
	request Request,
	ignored map[string]string,
	resolved []*model.DiffEntry,
	sourceCount int,
) (chan compareJob, func() error) {
	workerCount := min(sourceCount, diffWorkerCount())
	jobs := make(chan compareJob, max(1, workerCount))
	if workerCount == 0 {
		return jobs, func() error { return nil }
	}
	var waitGroup sync.WaitGroup
	var resultErr error
	var errOnce sync.Once

	waitGroup.Add(workerCount)
	for worker := 0; worker < workerCount; worker++ {
		go func() {
			defer waitGroup.Done()
			for job := range jobs {
				entry, err := s.diffFromSourceFile(request, job.relPath, job.targetExists, ignored)
				if err != nil {
					errOnce.Do(func() { resultErr = err })
					continue
				}
				resolved[job.index] = entry
			}
		}()
	}
	return jobs, func() error {
		waitGroup.Wait()
		return resultErr
	}
}

func enqueueMergedEntries(
	sourceFiles []string,
	targetFiles []string,
	ignored map[string]string,
	resolved []*model.DiffEntry,
	jobs chan<- compareJob,
) int {
	resolvedCount := 0
	sourceIndex := 0
	targetIndex := 0
	for sourceIndex < len(sourceFiles) && targetIndex < len(targetFiles) {
		sourcePath := sourceFiles[sourceIndex]
		targetPath := targetFiles[targetIndex]
		switch {
		case sourcePath < targetPath:
			queueCompareJob(jobs, resolvedCount, sourcePath, false)
			sourceIndex++
		case sourcePath > targetPath:
			resolved[resolvedCount] = deletedEntry(targetPath, ignored)
			targetIndex++
		default:
			queueCompareJob(jobs, resolvedCount, sourcePath, true)
			sourceIndex++
			targetIndex++
		}
		resolvedCount++
	}
	return enqueueRemainingEntries(sourceFiles, targetFiles, ignored, resolved, jobs, sourceIndex, targetIndex, resolvedCount)
}

func enqueueRemainingEntries(
	sourceFiles []string,
	targetFiles []string,
	ignored map[string]string,
	resolved []*model.DiffEntry,
	jobs chan<- compareJob,
	sourceIndex int,
	targetIndex int,
	resolvedCount int,
) int {
	for ; sourceIndex < len(sourceFiles); sourceIndex++ {
		queueCompareJob(jobs, resolvedCount, sourceFiles[sourceIndex], false)
		resolvedCount++
	}
	for ; targetIndex < len(targetFiles); targetIndex++ {
		resolved[resolvedCount] = deletedEntry(targetFiles[targetIndex], ignored)
		resolvedCount++
	}
	return resolvedCount
}

func queueCompareJob(jobs chan<- compareJob, index int, relPath string, targetExists bool) {
	jobs <- compareJob{index: index, relPath: relPath, targetExists: targetExists}
}

func compactEntriesAndSummary(resolved []*model.DiffEntry) ([]model.DiffEntry, model.DiffSummary, error) {
	entries := make([]model.DiffEntry, 0, len(resolved))
	summary := model.DiffSummary{}
	for _, entry := range resolved {
		if entry != nil {
			entries = append(entries, *entry)
			appendSummary(&summary, entry.Kind)
		}
	}
	return entries, summary, nil
}

func diffWorkerCount() int {
	return max(2, runtime.GOMAXPROCS(0))
}

func appendSummary(summary *model.DiffSummary, kind model.DiffKind) {
	summary.Total++
	switch kind {
	case model.DiffKindAdded:
		summary.Added++
	case model.DiffKindModified:
		summary.Modified++
	case model.DiffKindDeleted:
		summary.Deleted++
	case model.DiffKindProtected:
		summary.Protected++
	}
}

func (s *Service) diffFromSourceFile(
	request Request,
	relPath string,
	targetExists bool,
	ignored map[string]string,
) (*model.DiffEntry, error) {
	if shouldSkipPath(relPath, ignored) {
		return nil, nil
	}
	sourcePath := fullPath(request.SourceRoot, relPath)
	if !targetExists {
		return s.sizedEntry(sourcePath, relPath, model.DiffKindAdded)
	}
	targetPath := fullPath(request.TargetRoot, relPath)
	comparison, err := s.fs.CompareFile(sourcePath, targetPath)
	if err != nil {
		return nil, err
	}
	if comparison.Equal {
		return nil, nil
	}
	return entryWithSize(relPath, model.DiffKindModified, comparison.LeftSize), nil
}

func fullPath(root string, relPath string) string {
	return filepath.Join(root, filepath.FromSlash(relPath))
}

func (s *Service) sizedEntry(path string, relPath string, kind model.DiffKind) (*model.DiffEntry, error) {
	sizeBytes, err := s.fs.FileSize(path)
	if err != nil {
		return nil, err
	}
	return entryWithSize(relPath, kind, sizeBytes), nil
}

func entryWithSize(relPath string, kind model.DiffKind, sizeBytes int64) *model.DiffEntry {
	return &model.DiffEntry{Path: relPath, Kind: kind, SizeBytes: sizeBytes}
}

func deletedEntry(relPath string, ignored map[string]string) *model.DiffEntry {
	if shouldSkipPath(relPath, ignored) {
		return nil
	}
	return &model.DiffEntry{Path: relPath, Kind: model.DiffKindDeleted}
}

func shouldSkipPath(relPath string, ignored map[string]string) bool {
	if isProtected(relPath) {
		return true
	}
	_, isIgnored := ignored[relPath]
	return isIgnored
}
