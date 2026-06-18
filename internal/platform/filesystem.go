package platform

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FileSystem interface {
	ListRegularFiles(root string) ([]string, error)
	Exists(path string) (bool, error)
	FilesEqual(left string, right string) (bool, error)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	FileMode(path string) (fs.FileMode, error)
	EnsureDirectory(path string) error
	Remove(path string) error
	RemoveEmptyParents(root string, start string) error
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
	sort.Strings(files)
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

func (fsys *OSFileSystem) FilesEqual(left string, right string) (bool, error) {
	leftInfo, err := os.Stat(left)
	if err != nil {
		return false, err
	}
	rightInfo, err := os.Stat(right)
	if err != nil {
		return false, err
	}
	if leftInfo.Size() != rightInfo.Size() {
		return false, nil
	}
	return compareFileContents(left, right)
}

func (fsys *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fsys *OSFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm.Perm())
}

func (fsys *OSFileSystem) FileMode(path string) (fs.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Mode(), nil
}

func (fsys *OSFileSystem) EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (fsys *OSFileSystem) Remove(path string) error {
	return os.Remove(path)
}

func (fsys *OSFileSystem) RemoveEmptyParents(root string, start string) error {
	current := filepath.Dir(start)
	cleanRoot := filepath.Clean(root)
	for current != cleanRoot && current != "." {
		entries, err := os.ReadDir(current)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return nil
		}
		if err := os.Remove(current); err != nil {
			return err
		}
		current = filepath.Dir(current)
	}
	return nil
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

	leftContent, err := io.ReadAll(leftFile)
	if err != nil {
		return false, err
	}
	rightContent, err := io.ReadAll(rightFile)
	if err != nil {
		return false, err
	}
	return bytes.Equal(leftContent, rightContent), nil
}
