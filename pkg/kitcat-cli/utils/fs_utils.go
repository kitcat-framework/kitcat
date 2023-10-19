package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindGoModPath finds the absolute path of the nearest go.mod file.
func FindGoModPath() (string, error) {
	// Start searching from the current directory.
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		_, err := os.Stat(goModPath)
		if err == nil {
			return currentDir, nil // Found go.mod, return the current directory
		}

		// Move up one directory.
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached the root directory, and go.mod is not found.
			return "", fmt.Errorf("go.mod not found in the current or parent directories")
		}

		currentDir = parentDir
	}
}

func CreateDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}

	return nil
}

func CreateFileWithDirsIfNotExist(path, data string) (*os.File, error) {
	if err := CreateDirIfNotExist(filepath.Dir(path)); err != nil {
		return nil, err
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	if data != "" {
		if _, err := f.WriteString(data); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func FileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
