package platform

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const fileCompareBufferSize = 32 * 1024
const maxRetainedPathCap = 4 * 1024

var fileBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, fileCompareBufferSize)
	},
}

var pathBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 256)
	},
}

type FileSystem interface {
	ListRegularFiles(root string) ([]string, error)
	Exists(path string) (bool, error)
	CompareFile(left string, right string) (FileComparison, error)
	CompareFileFromRoots(leftRoot string, rightRoot string, relPath string) (FileComparison, error)
	FilesEqual(left string, right string) (bool, error)
	FileSize(path string) (int64, error)
	FileSizeFromRoot(root string, relPath string) (int64, error)
	CopyFile(sourcePath string, targetPath string) error
	CopyFileFromRoots(sourceRoot string, targetRoot string, relPath string) error
	EnsureDirectory(path string) error
	Remove(path string) error
	RemoveFromRoot(root string, relPath string) error
	RemoveEmptyParents(root string, start string) error
	RemoveEmptyParentsFromRoot(root string, relPath string) error
}

type FileComparison struct {
	Equal    bool
	LeftSize int64
}

type OSFileSystem struct{}

func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

func (fsys *OSFileSystem) ListRegularFiles(root string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && strings.EqualFold(entry.Name(), ".git") {
			return filepath.SkipDir
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	return files, err
}

func (fsys *OSFileSystem) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (fsys *OSFileSystem) CompareFile(left string, right string) (FileComparison, error) {
	leftInfo, err := os.Stat(left)
	if err != nil {
		return FileComparison{}, err
	}
	rightInfo, err := os.Stat(right)
	if err != nil {
		return FileComparison{}, err
	}
	comparison := FileComparison{LeftSize: leftInfo.Size()}
	if comparison.LeftSize != rightInfo.Size() {
		return comparison, nil
	}
	comparison.Equal, err = compareFileContents(left, right)
	return comparison, err
}

func (fsys *OSFileSystem) FilesEqual(left string, right string) (bool, error) {
	comparison, err := fsys.CompareFile(left, right)
	if err != nil {
		return false, err
	}
	return comparison.Equal, nil
}

func (fsys *OSFileSystem) FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (fsys *OSFileSystem) FileSizeFromRoot(root string, relPath string) (int64, error) {
	buffer := borrowPathBuffer(len(root) + len(relPath) + 1)
	defer releasePathBuffer(buffer)

	path := buildRootedPath(buffer, root, relPath)
	info, err := os.Stat(bytesToStringView(path))
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (fsys *OSFileSystem) CopyFile(sourcePath string, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	if err := fsys.EnsureDirectory(filepath.Dir(targetPath)); err != nil {
		return err
	}
	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode().Perm())
	if err != nil {
		return err
	}
	defer targetFile.Close()

	buffer := borrowFileBuffer()
	defer releaseFileBuffer(buffer)

	_, err = io.CopyBuffer(targetFile, sourceFile, buffer)
	return err
}

func (fsys *OSFileSystem) CopyFileFromRoots(sourceRoot string, targetRoot string, relPath string) error {
	sourceBuffer := borrowPathBuffer(len(sourceRoot) + len(relPath) + 1)
	targetBuffer := borrowPathBuffer(len(targetRoot) + len(relPath) + 1)
	defer releasePathBuffer(sourceBuffer)
	defer releasePathBuffer(targetBuffer)

	sourcePath := bytesToStringView(buildRootedPath(sourceBuffer, sourceRoot, relPath))
	targetPath := bytesToStringView(buildRootedPath(targetBuffer, targetRoot, relPath))
	return fsys.CopyFile(sourcePath, targetPath)
}

func (fsys *OSFileSystem) EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (fsys *OSFileSystem) Remove(path string) error {
	return os.Remove(path)
}

func (fsys *OSFileSystem) RemoveFromRoot(root string, relPath string) error {
	buffer := borrowPathBuffer(len(root) + len(relPath) + 1)
	defer releasePathBuffer(buffer)

	path := buildRootedPath(buffer, root, relPath)
	return fsys.Remove(bytesToStringView(path))
}

func (fsys *OSFileSystem) CompareFileFromRoots(leftRoot string, rightRoot string, relPath string) (FileComparison, error) {
	leftBuffer := borrowPathBuffer(len(leftRoot) + len(relPath) + 1)
	rightBuffer := borrowPathBuffer(len(rightRoot) + len(relPath) + 1)
	defer releasePathBuffer(leftBuffer)
	defer releasePathBuffer(rightBuffer)

	leftPath := bytesToStringView(buildRootedPath(leftBuffer, leftRoot, relPath))
	rightPath := bytesToStringView(buildRootedPath(rightBuffer, rightRoot, relPath))

	leftInfo, err := os.Stat(leftPath)
	if err != nil {
		return FileComparison{}, err
	}
	rightInfo, err := os.Stat(rightPath)
	if err != nil {
		return FileComparison{}, err
	}
	comparison := FileComparison{LeftSize: leftInfo.Size()}
	if comparison.LeftSize != rightInfo.Size() {
		return comparison, nil
	}
	comparison.Equal, err = compareFileContents(leftPath, rightPath)
	return comparison, err
}

func (fsys *OSFileSystem) RemoveEmptyParents(root string, start string) error {
	current := filepath.Dir(start)
	cleanRoot := filepath.Clean(root)
	for current != cleanRoot && current != "." {
		err := os.Remove(current)
		switch {
		case err == nil:
			current = filepath.Dir(current)
		case os.IsNotExist(err):
			current = filepath.Dir(current)
		case isDirectoryNotEmptyError(err):
			return nil
		default:
			return err
		}
	}
	return nil
}

func (fsys *OSFileSystem) RemoveEmptyParentsFromRoot(root string, relPath string) error {
	buffer := borrowPathBuffer(len(root) + len(relPath) + 1)
	defer releasePathBuffer(buffer)

	start := buildRootedPath(buffer, root, relPath)
	return fsys.RemoveEmptyParents(root, bytesToStringView(start))
}

func compareFileContents(left string, right string) (bool, error) {
	leftFile, err := os.Open(left)
	if err != nil {
		return false, err
	}
	defer leftFile.Close()

	rightFile, err := os.Open(right)
	if err != nil {
		return false, err
	}
	defer rightFile.Close()

	return compareReaders(leftFile, rightFile)
}

func compareReaders(left io.Reader, right io.Reader) (bool, error) {
	leftBuffer := borrowFileBuffer()
	rightBuffer := borrowFileBuffer()
	defer releaseFileBuffer(leftBuffer)
	defer releaseFileBuffer(rightBuffer)

	for {
		leftRead, leftErr := left.Read(leftBuffer)
		rightRead, rightErr := right.Read(rightBuffer)

		if leftRead != rightRead || !bytes.Equal(leftBuffer[:leftRead], rightBuffer[:rightRead]) {
			return false, nil
		}
		if leftErr == io.EOF || rightErr == io.EOF {
			return leftErr == io.EOF && rightErr == io.EOF, nil
		}
		if leftErr != nil {
			return false, leftErr
		}
		if rightErr != nil {
			return false, rightErr
		}
	}
}

func isDirectoryNotEmptyError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.Errno(145))
}

func borrowFileBuffer() []byte {
	return fileBufferPool.Get().([]byte)
}

func releaseFileBuffer(buffer []byte) {
	fileBufferPool.Put(buffer[:fileCompareBufferSize])
}

func borrowPathBuffer(targetCap int) []byte {
	buffer := pathBufferPool.Get().([]byte)
	if cap(buffer) < targetCap {
		return make([]byte, 0, targetCap)
	}
	return buffer[:0]
}

func releasePathBuffer(buffer []byte) {
	if cap(buffer) > maxRetainedPathCap {
		return
	}
	pathBufferPool.Put(buffer[:0])
}

func buildRootedPath(buffer []byte, root string, relPath string) []byte {
	if root == "" {
		return appendNativeRelativePath(buffer[:0], relPath)
	}
	buffer = append(buffer[:0], root...)
	if !isPathSeparator(root[len(root)-1]) {
		buffer = append(buffer, '\\')
	}
	return appendNativeRelativePath(buffer, relPath)
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

func isPathSeparator(char byte) bool {
	return char == '/' || char == '\\'
}

func bytesToStringView(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(raw), len(raw))
}
