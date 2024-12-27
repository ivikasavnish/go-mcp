package ide

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// FileManager handles file operations
type FileManager struct {
	rootDir string
}

func NewFileManager(rootDir string) *FileManager {
	return &FileManager{rootDir: rootDir}
}

func (fm *FileManager) CreateFile(path string, content []byte) error {
	fullPath := filepath.Join(fm.rootDir, path)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	return ioutil.WriteFile(fullPath, content, 0644)
}

func (fm *FileManager) ReadFile(path string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(fm.rootDir, path))
}

func (fm *FileManager) DeleteFile(path string) error {
	return os.Remove(filepath.Join(fm.rootDir, path))
}

func (fm *FileManager) ListFiles(path string) ([]FileInfo, error) {
	var files []FileInfo
	fullPath := filepath.Join(fm.rootDir, path)

	entries, err := ioutil.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		info := FileInfo{
			Name:        entry.Name(),
			Path:        filepath.Join(path, entry.Name()),
			Size:        entry.Size(),
			IsDir:       entry.IsDir(),
			ModTime:     entry.ModTime(),
			Permissions: entry.Mode().String(),
		}
		files = append(files, info)
	}

	return files, nil
}
