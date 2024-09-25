package utils

import (
	"os"
	"path/filepath"
	"strings"
)

func ListFilesWithPrefix(dir, prefix string) ([]string, error) {
	var filesWithPrefix []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasPrefix(filepath.Base(path), prefix) {
			filesWithPrefix = append(filesWithPrefix, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return filesWithPrefix, nil
}
