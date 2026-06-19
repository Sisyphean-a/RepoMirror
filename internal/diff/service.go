package diff

import (
	"runtime"
	"sync"

	"RepoMirror/internal/model"
	"RepoMirror/internal/platform"
)

type GitInspector interface {
	ListSyncableSourcePathsFromRoot(root string) ([]string, error)
	IgnoredPathSetFromRootSorted(root string, paths []string) (map[string]struct{}, error)
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

const (
	unresolvedAddedSize   int64 = -1
	unresolvedCompareSize int64 = -2
)

func NewService(fsys platform.FileSystem, gitInspector GitInspector) *Service {
	return &Service{fs: fsys, git: gitInspector}
}

func (s *Service) Calculate(request Request) (Result, error) {
	sourceFiles, targetFiles, err := s.loadFiles(request)
	if err != nil {
		return Result{}, err
	}
	ignored, err := s.git.IgnoredPathSetFromRootSorted(request.TargetRoot, sourceFiles)
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

	waitGroup.Add(1)
	go func() {
		sourceFiles, sourceErr = s.git.ListSyncableSourcePathsFromRoot(request.SourceRoot)
		waitGroup.Done()
	}()
	targetFiles, targetErr = s.git.ListSyncableSourcePathsFromRoot(request.TargetRoot)
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
	ignored map[string]struct{},
) ([]model.DiffEntry, model.DiffSummary, error) {
	resolved, err := s.resolveEntries(request, sourceFiles, targetFiles, ignored)
	if err != nil {
		return nil, model.DiffSummary{}, err
	}
	entries, summary := compactEntriesAndSummary(resolved)
	return entries, summary, nil
}

func (s *Service) resolveEntries(
	request Request,
	sourceFiles []string,
	targetFiles []string,
	ignored map[string]struct{},
) ([]model.DiffEntry, error) {
	resolved := make([]model.DiffEntry, mergedPathCount(sourceFiles, targetFiles))
	resolvedCount, err := s.mergeEntries(request, sourceFiles, targetFiles, ignored, resolved)
	if err != nil {
		return nil, err
	}
	return resolved[:resolvedCount], nil
}

func mergedPathCount(sourceFiles []string, targetFiles []string) int {
	count := 0
	sourceIndex := 0
	targetIndex := 0
	for sourceIndex < len(sourceFiles) && targetIndex < len(targetFiles) {
		switch {
		case sourceFiles[sourceIndex] < targetFiles[targetIndex]:
			sourceIndex++
		case sourceFiles[sourceIndex] > targetFiles[targetIndex]:
			targetIndex++
		default:
			sourceIndex++
			targetIndex++
		}
		count++
	}
	return count + len(sourceFiles) - sourceIndex + len(targetFiles) - targetIndex
}

func (s *Service) mergeEntries(
	request Request,
	sourceFiles []string,
	targetFiles []string,
	ignored map[string]struct{},
	resolved []model.DiffEntry,
) (int, error) {
	resolvedCount := enqueueMergedEntries(sourceFiles, targetFiles, ignored, resolved)
	if err := s.resolveComparedEntries(request, ignored, resolved[:resolvedCount], len(sourceFiles)); err != nil {
		return 0, err
	}
	return resolvedCount, nil
}

func (s *Service) resolveComparedEntries(
	request Request,
	ignored map[string]struct{},
	resolved []model.DiffEntry,
	sourceCount int,
) error {
	workerCount := min(sourceCount, diffWorkerCount())
	if workerCount == 0 {
		return nil
	}
	var waitGroup sync.WaitGroup
	var resultErr error
	var errOnce sync.Once

	waitGroup.Add(max(0, workerCount-1))
	for worker := 0; worker < workerCount-1; worker++ {
		go func(worker int) {
			s.resolveComparedEntryRange(request, ignored, resolved, worker, workerCount, &resultErr, &errOnce)
			waitGroup.Done()
		}(worker)
	}
	s.resolveComparedEntryRange(request, ignored, resolved, workerCount-1, workerCount, &resultErr, &errOnce)
	waitGroup.Wait()
	return resultErr
}

func (s *Service) resolveComparedEntryRange(
	request Request,
	ignored map[string]struct{},
	resolved []model.DiffEntry,
	worker int,
	workerCount int,
	resultErr *error,
	errOnce *sync.Once,
) {
	for index := worker; index < len(resolved); index += workerCount {
		relPath, targetExists, ok := unresolvedCompareEntry(resolved[index])
		if !ok {
			continue
		}
		entry, err := s.diffFromSourceFile(request, relPath, targetExists, ignored)
		if err != nil {
			errOnce.Do(func() { *resultErr = err })
			continue
		}
		resolved[index] = entry
	}
}

func enqueueMergedEntries(
	sourceFiles []string,
	targetFiles []string,
	ignored map[string]struct{},
	resolved []model.DiffEntry,
) int {
	resolvedCount := 0
	sourceIndex := 0
	targetIndex := 0
	for sourceIndex < len(sourceFiles) && targetIndex < len(targetFiles) {
		sourcePath := sourceFiles[sourceIndex]
		targetPath := targetFiles[targetIndex]
		switch {
		case sourcePath < targetPath:
			resolved[resolvedCount] = unresolvedAddedEntry(sourcePath)
			sourceIndex++
		case sourcePath > targetPath:
			resolved[resolvedCount] = deletedEntry(targetPath, ignored)
			targetIndex++
		default:
			resolved[resolvedCount] = unresolvedModifiedEntry(sourcePath)
			sourceIndex++
			targetIndex++
		}
		resolvedCount++
	}
	return enqueueRemainingEntries(sourceFiles, targetFiles, ignored, resolved, sourceIndex, targetIndex, resolvedCount)
}

func enqueueRemainingEntries(
	sourceFiles []string,
	targetFiles []string,
	ignored map[string]struct{},
	resolved []model.DiffEntry,
	sourceIndex int,
	targetIndex int,
	resolvedCount int,
) int {
	for ; sourceIndex < len(sourceFiles); sourceIndex++ {
		resolved[resolvedCount] = unresolvedAddedEntry(sourceFiles[sourceIndex])
		resolvedCount++
	}
	for ; targetIndex < len(targetFiles); targetIndex++ {
		resolved[resolvedCount] = deletedEntry(targetFiles[targetIndex], ignored)
		resolvedCount++
	}
	return resolvedCount
}

func unresolvedAddedEntry(relPath string) model.DiffEntry {
	return model.DiffEntry{Path: relPath, SizeBytes: unresolvedAddedSize}
}

func unresolvedModifiedEntry(relPath string) model.DiffEntry {
	return model.DiffEntry{Path: relPath, SizeBytes: unresolvedCompareSize}
}

func unresolvedCompareEntry(entry model.DiffEntry) (string, bool, bool) {
	switch entry.SizeBytes {
	case unresolvedAddedSize:
		return entry.Path, false, true
	case unresolvedCompareSize:
		return entry.Path, true, true
	default:
		return "", false, false
	}
}

func compactEntriesAndSummary(resolved []model.DiffEntry) ([]model.DiffEntry, model.DiffSummary) {
	summary := model.DiffSummary{}
	writeIndex := 0
	for readIndex := range resolved {
		entry := resolved[readIndex]
		if entry.Kind == "" {
			continue
		}
		appendSummary(&summary, entry.Kind)
		if writeIndex != readIndex {
			resolved[writeIndex] = entry
		}
		writeIndex++
	}
	return resolved[:writeIndex], summary
}

func diffWorkerCount() int {
	return max(2, runtime.GOMAXPROCS(0)/2)
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
	}
}

func (s *Service) diffFromSourceFile(
	request Request,
	relPath string,
	targetExists bool,
	ignored map[string]struct{},
) (model.DiffEntry, error) {
	if len(ignored) != 0 {
		if _, isIgnored := ignored[relPath]; isIgnored {
			return model.DiffEntry{}, nil
		}
	}
	if !targetExists {
		sizeBytes, err := s.fs.FileSizeFromRoot(request.SourceRoot, relPath)
		if err != nil {
			return model.DiffEntry{}, err
		}
		return entryWithSize(relPath, model.DiffKindAdded, sizeBytes), nil
	}
	comparison, err := s.fs.CompareFileFromRoots(request.SourceRoot, request.TargetRoot, relPath)
	if err != nil {
		return model.DiffEntry{}, err
	}
	if comparison.Equal {
		return model.DiffEntry{}, nil
	}
	return entryWithSize(relPath, model.DiffKindModified, comparison.LeftSize), nil
}

func (s *Service) sizedEntry(path string, relPath string, kind model.DiffKind) (model.DiffEntry, error) {
	sizeBytes, err := s.fs.FileSize(path)
	if err != nil {
		return model.DiffEntry{}, err
	}
	return entryWithSize(relPath, kind, sizeBytes), nil
}

func entryWithSize(relPath string, kind model.DiffKind, sizeBytes int64) model.DiffEntry {
	return model.DiffEntry{
		Path:      relPath,
		Kind:      kind,
		SizeBytes: sizeBytes,
	}
}

func deletedEntry(relPath string, ignored map[string]struct{}) model.DiffEntry {
	if len(ignored) != 0 {
		if _, isIgnored := ignored[relPath]; isIgnored {
			return model.DiffEntry{}
		}
	}
	return entryWithSize(relPath, model.DiffKindDeleted, 0)
}
