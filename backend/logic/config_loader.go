package logic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func deepMergeJSON(dst map[string]any, src map[string]any) {
	for k, v := range src {
		if vMap, ok := v.(map[string]any); ok {
			if existing, ok := dst[k].(map[string]any); ok {
				deepMergeJSON(existing, vMap)
				dst[k] = existing
				continue
			}
			cloned := map[string]any{}
			deepMergeJSON(cloned, vMap)
			dst[k] = cloned
			continue
		}
		dst[k] = v
	}
}

func loadJSONMap(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

// LoadDefaultGameConfig loads the baseline server config.
//
// Priority:
//  1. <root>/config/*.json (split config, deep-merged in stable order)
//  2. <root>/game_config.json (legacy single-file)
func LoadDefaultGameConfig(rootDir string) (GameConfig, error) {
	var cfg GameConfig

	if rootDir == "" {
		return cfg, fmt.Errorf("rootDir is empty")
	}

	legacyPath := filepath.Join(rootDir, "game_config.json")
	base, err := loadJSONMap(legacyPath)
	loadedLegacy := true
	if err != nil {
		if os.IsNotExist(err) {
			base = map[string]any{}
			loadedLegacy = false
		} else {
			return cfg, fmt.Errorf("load legacy %s: %w", legacyPath, err)
		}
	}

	configDir := filepath.Join(rootDir, "config")
	order := []string{
		"server.json",
		"map.json",
		"gameplay.json",
		"items.json",
		"tactics.json",
		"combat.json",
		"phases.json",
	}

	merged := base
	loadedAny := loadedLegacy
	for _, name := range order {
		p := filepath.Join(configDir, name)
		m, err := loadJSONMap(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return cfg, fmt.Errorf("load %s: %w", p, err)
		}
		deepMergeJSON(merged, m)
		loadedAny = true
	}
	if !loadedAny {
		return cfg, fmt.Errorf("no config files found under %s (expected %s or %s)", rootDir, filepath.Join(rootDir, "config"), legacyPath)
	}

	b, err := json.Marshal(merged)
	if err != nil {
		return cfg, fmt.Errorf("marshal merged config: %w", err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse merged config: %w", err)
	}
	return cfg, nil
}
