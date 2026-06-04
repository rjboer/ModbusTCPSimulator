package app

import (
	"fmt"
	"os"
	"path/filepath"
)

func DetectProjectRoot() (string, error) {
	candidates := []string{}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd, filepath.Dir(cwd))
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates, exeDir, filepath.Dir(exeDir))
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if looksLikeRuntimeRoot(candidate) {
			return candidate, nil
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		return cwd, nil
	}

	return "", fmt.Errorf("could not determine project root")
}

func ConfigRoot(projectRoot string) string {
	return filepath.Join(projectRoot, "configs")
}

func looksLikeRuntimeRoot(path string) bool {
	return isDir(filepath.Join(path, "configs"))
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
