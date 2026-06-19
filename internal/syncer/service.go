package syncer

import (
	"runtime"
	"sync"
	"sync/atomic"

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
	copyCount := result.Summary.Added + result.Summary.Modified
	deleteCount := result.Summary.Deleted
	if err := s.applyCopies(request, result.Entries, copyCount); err != nil {
		return err
	}
	return s.applyDeletes(request, result.Entries, deleteCount)
}

func (s *Service) applyCopies(request Request, entries []model.DiffEntry, copyCount int) error {
	if copyCount == 0 {
		return nil
	}
	workerCount := min(copyCount, copyWorkerCount())
	var waitGroup sync.WaitGroup
	var firstErr error
	var errOnce sync.Once
	var stop atomic.Bool
	copyWork := func(worker int) {
		defer waitGroup.Done()
		for index := worker; index < len(entries); index += workerCount {
			if stop.Load() {
				return
			}
			entry := entries[index]
			if entry.Kind != model.DiffKindAdded && entry.Kind != model.DiffKindModified {
				continue
			}
			if err := s.copyFile(request, entry.Path); err != nil {
				errOnce.Do(func() {
					firstErr = err
					stop.Store(true)
				})
				return
			}
		}
	}

	waitGroup.Add(workerCount)
	for worker := 0; worker < workerCount; worker++ {
		go copyWork(worker)
	}
	waitGroup.Wait()
	return firstErr
}

func (s *Service) applyDeletes(request Request, entries []model.DiffEntry, deleteCount int) error {
	if deleteCount == 0 {
		return nil
	}
	workerCount := min(deleteCount, copyWorkerCount())
	cleanupGroups := make([]cleanupGroup, workerCount)
	var waitGroup sync.WaitGroup
	var firstErr error
	var errOnce sync.Once
	var stop atomic.Bool
	deleteWork := func(worker int) {
		defer waitGroup.Done()
		group := &cleanupGroups[worker]
		group.entries = group.inline[:0]
		for index := worker; index < len(entries); index += workerCount {
			if stop.Load() {
				return
			}
			entry := entries[index]
			if entry.Kind != model.DiffKindDeleted {
				continue
			}
			if err := s.fs.RemoveFromRoot(request.TargetRoot, entry.Path); err != nil {
				errOnce.Do(func() {
					firstErr = err
					stop.Store(true)
				})
				return
			}
			directoryKey := relativeDirectoryKey(entry.Path)
			if group.seen != nil {
				if _, exists := group.seen[directoryKey]; exists {
					continue
				}
				group.seen[directoryKey] = struct{}{}
				group.entries = append(group.entries, cleanupPath{directory: directoryKey, relPath: entry.Path})
				continue
			}
			if hasCleanupDirectory(group.entries, directoryKey) {
				continue
			}
			group.entries = append(group.entries, cleanupPath{directory: directoryKey, relPath: entry.Path})
			if len(group.entries) == cleanupLinearScanLimit+1 {
				group.seen = cleanupSeenSet(group.entries)
			}
		}
	}

	waitGroup.Add(workerCount)
	for worker := 0; worker < workerCount; worker++ {
		go deleteWork(worker)
	}
	waitGroup.Wait()
	if firstErr != nil {
		return firstErr
	}
	return removeUniqueCleanupPaths(s.fs, request.TargetRoot, cleanupGroups)
}

func (s *Service) copyFile(request Request, relPath string) error {
	return s.fs.CopyFileFromRoots(request.SourceRoot, request.TargetRoot, relPath)
}

func joinPath(root string, relPath string) string {
	if root == "" {
		return nativeRelativePath(relPath)
	}
	buffer := make([]byte, 0, len(root)+len(relPath)+1)
	buffer = append(buffer, root...)
	if !isPathSeparator(root[len(root)-1]) {
		buffer = append(buffer, '\\')
	}
	buffer = appendNativeRelativePath(buffer, relPath)
	return string(buffer)
}

func copyWorkerCount() int {
	return max(2, runtime.GOMAXPROCS(0)/2)
}

func nativeRelativePath(relPath string) string {
	if indexByte(relPath, '/') < 0 {
		return relPath
	}
	buffer := make([]byte, 0, len(relPath))
	buffer = appendNativeRelativePath(buffer, relPath)
	return string(buffer)
}

func appendNativeRelativePath(buffer []byte, relPath string) []byte {
	for index := 0; index < len(relPath); index++ {
		if relPath[index] == '/' {
			buffer = append(buffer, '\\')
			continue
		}
		buffer = append(buffer, relPath[index])
	}
	return buffer
}

func relativeDirectoryKey(relPath string) string {
	for index := len(relPath) - 1; index >= 0; index-- {
		if isPathSeparator(relPath[index]) {
			return relPath[:index]
		}
	}
	return ""
}

func isPathSeparator(char byte) bool {
	return char == '/' || char == '\\'
}

func indexByte(value string, target byte) int {
	for index := 0; index < len(value); index++ {
		if value[index] == target {
			return index
		}
	}
	return -1
}
