package collector

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

const maxCollectorPromptBytes = 64 * 1024

var (
	ErrCollectorPromptLoad    = errors.New("load collector prompt failed")
	ErrCollectorPromptInvalid = errors.New("collector prompt is invalid")
)

// LoadCollectorPrompt reads one immutable collector objective from path.
// Relative paths follow the process current working directory.
func LoadCollectorPrompt(path string) (string, error) {
	return loadCollectorPrompt(path, os.Open)
}

func loadCollectorPrompt(path string, openFile func(string) (*os.File, error)) (string, error) {
	file, err := openFile(path)
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrCollectorPromptLoad, path)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrCollectorPromptLoad, path)
	}
	if !info.Mode().IsRegular() || info.Size() > maxCollectorPromptBytes {
		return "", fmt.Errorf("%w: %q", ErrCollectorPromptInvalid, path)
	}

	content, err := io.ReadAll(io.LimitReader(file, maxCollectorPromptBytes+1))
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrCollectorPromptLoad, path)
	}
	if len(content) > maxCollectorPromptBytes || !utf8.Valid(content) || strings.TrimSpace(string(content)) == "" {
		return "", fmt.Errorf("%w: %q", ErrCollectorPromptInvalid, path)
	}
	return string(content), nil
}
