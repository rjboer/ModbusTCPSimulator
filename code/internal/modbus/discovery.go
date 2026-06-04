package modbus

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type DiscoveredConfig struct {
	Path     string
	RelPath  string
	Config   Config
	Rendered string
}

type InvalidConfig struct {
	Path    string
	RelPath string
	Err     error
}

type DiscoveryResult struct {
	Root    string
	Valid   []DiscoveredConfig
	Invalid []InvalidConfig
}

func DiscoverConfigs(root string) (DiscoveryResult, error) {
	result := DiscoveryResult{Root: root}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), "c.json") {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			relPath = path
		}

		cfg, err := LoadConfig(path)
		if err != nil {
			result.Invalid = append(result.Invalid, InvalidConfig{
				Path:    path,
				RelPath: relPath,
				Err:     err,
			})
			return nil
		}

		rendered, err := MarshalConfig(cfg)
		if err != nil {
			result.Invalid = append(result.Invalid, InvalidConfig{
				Path:    path,
				RelPath: relPath,
				Err:     fmt.Errorf("render config: %w", err),
			})
			return nil
		}

		result.Valid = append(result.Valid, DiscoveredConfig{
			Path:     path,
			RelPath:  relPath,
			Config:   cfg,
			Rendered: rendered,
		})
		return nil
	})
	if err != nil {
		return DiscoveryResult{}, err
	}

	sort.Slice(result.Valid, func(i, j int) bool {
		return strings.ToLower(result.Valid[i].RelPath) < strings.ToLower(result.Valid[j].RelPath)
	})
	sort.Slice(result.Invalid, func(i, j int) bool {
		return strings.ToLower(result.Invalid[i].RelPath) < strings.ToLower(result.Invalid[j].RelPath)
	})

	return result, nil
}
