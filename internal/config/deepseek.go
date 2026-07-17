package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	defaultDeepSeekModel   = "deepseek-chat"
	defaultDeepSeekTimeout = 30 * time.Second
)

type DeepSeekConfig struct {
	APIKey  string
	Model   string
	BaseURL string
	Timeout time.Duration
}

func LoadDeepSeekConfig() (DeepSeekConfig, error) {
	config := DeepSeekConfig{
		APIKey:  strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")),
		Model:   strings.TrimSpace(os.Getenv("DEEPSEEK_MODEL")),
		BaseURL: strings.TrimSpace(os.Getenv("DEEPSEEK_BASE_URL")),
		Timeout: defaultDeepSeekTimeout,
	}
	if config.APIKey == "" {
		return DeepSeekConfig{}, fmt.Errorf("DEEPSEEK_API_KEY is missing")
	}
	if config.Model == "" {
		config.Model = defaultDeepSeekModel
	}
	if rawTimeout := strings.TrimSpace(os.Getenv("DEEPSEEK_TIMEOUT")); rawTimeout != "" {
		timeout, err := time.ParseDuration(rawTimeout)
		if err != nil || timeout <= 0 {
			return DeepSeekConfig{}, fmt.Errorf("DEEPSEEK_TIMEOUT must be a positive Go duration")
		}
		config.Timeout = timeout
	}
	return config, nil
}
