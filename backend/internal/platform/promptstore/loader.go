package promptstore

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Prompt struct {
	Ref     string
	Version string
	Text    string
}

type Loader struct {
	Root string
}

func (l Loader) Load(ref string, version string, variables map[string]any) (Prompt, error) {
	cleanRef, err := cleanPromptRef(ref)
	if err != nil {
		return Prompt{}, err
	}
	if strings.TrimSpace(version) == "" {
		return Prompt{}, fmt.Errorf("prompt version is required")
	}
	if !strings.Contains(filepath.Base(cleanRef), "."+version+".") {
		return Prompt{}, fmt.Errorf("prompt version mismatch")
	}
	if strings.TrimSpace(l.Root) == "" {
		return Prompt{}, fmt.Errorf("prompt root is required")
	}

	path := filepath.Join(l.Root, filepath.FromSlash(cleanRef))
	data, err := os.ReadFile(path)
	if err != nil {
		return Prompt{}, fmt.Errorf("read prompt: %w", err)
	}
	text, err := renderVariables(string(data), variables)
	if err != nil {
		return Prompt{}, err
	}
	return Prompt{Ref: cleanRef, Version: version, Text: text}, nil
}

func cleanPromptRef(ref string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(strings.TrimSpace(ref)))
	if cleaned == "." || filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned == ".." {
		return "", fmt.Errorf("invalid prompt ref")
	}
	return filepath.ToSlash(cleaned), nil
}

var promptVariablePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

func renderVariables(text string, variables map[string]any) (string, error) {
	var missing string
	rendered := promptVariablePattern.ReplaceAllStringFunc(text, func(match string) string {
		if missing != "" {
			return match
		}
		parts := promptVariablePattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		value, ok := variables[parts[1]]
		if !ok {
			missing = parts[1]
			return match
		}
		return strings.TrimSpace(fmt.Sprint(value))
	})
	if missing != "" {
		return "", fmt.Errorf("missing prompt variable %q", missing)
	}
	return rendered, nil
}
