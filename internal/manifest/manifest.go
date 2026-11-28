package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// MaxSizeBytes caps the raw and canonical manifest payload sizes.
	MaxSizeBytes = 128 * 1024
)

// Normalize parses a JSON or YAML manifest and returns canonical JSON plus its digest.
func Normalize(raw string) ([]byte, string, error) {
	if raw == "" {
		return nil, "", fmt.Errorf("manifest is required")
	}
	if len(raw) > MaxSizeBytes {
		return nil, "", fmt.Errorf("manifest exceeds maximum size of %d bytes", MaxSizeBytes)
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, "", fmt.Errorf("manifest is required")
	}

	var value any
	if json.Valid([]byte(trimmed)) {
		if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
			return nil, "", fmt.Errorf("invalid JSON manifest: %w", err)
		}
	} else {
		if err := yaml.Unmarshal([]byte(trimmed), &value); err != nil {
			return nil, "", fmt.Errorf("invalid YAML manifest: %w", err)
		}
	}

	normalized, err := normalizeYAMLValue(value)
	if err != nil {
		return nil, "", err
	}

	canonical, err := json.Marshal(normalized)
	if err != nil {
		return nil, "", fmt.Errorf("failed to canonicalize manifest: %w", err)
	}
	if len(canonical) > MaxSizeBytes {
		return nil, "", fmt.Errorf("manifest exceeds maximum canonical size of %d bytes", MaxSizeBytes)
	}

	sum := sha256.Sum256(canonical)
	return canonical, hex.EncodeToString(sum[:]), nil
}

func normalizeYAMLValue(value any) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, item := range typed {
			next, err := normalizeYAMLValue(item)
			if err != nil {
				return nil, err
			}
			normalized[key] = next
		}
		return normalized, nil
	case map[any]any:
		normalized := make(map[string]any, len(typed))
		for key, item := range typed {
			keyString, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("manifest object keys must be strings")
			}
			next, err := normalizeYAMLValue(item)
			if err != nil {
				return nil, err
			}
			normalized[keyString] = next
		}
		return normalized, nil
	case []any:
		normalized := make([]any, len(typed))
		for i, item := range typed {
			next, err := normalizeYAMLValue(item)
			if err != nil {
				return nil, err
			}
			normalized[i] = next
		}
		return normalized, nil
	case nil, bool, string, float64, int, int64, uint64:
		return typed, nil
	default:
		return typed, nil
	}
}
