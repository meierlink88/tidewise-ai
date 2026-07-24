package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
)

func (s *Store) UpsertModelProviderConfig(ctx context.Context, config agentrun.ModelProviderConfig) error {
	if strings.TrimSpace(config.ProviderKey) == "" {
		return fmt.Errorf("Model Provider Configuration key is required")
	}
	if strings.TrimSpace(config.BaseURL) == "" {
		return fmt.Errorf("Model Provider Base URL is required")
	}
	_, err := s.database.Exec(ctx, `
		INSERT INTO model_provider_configs (provider_key, base_url, model, api_key, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (provider_key) DO UPDATE
		SET base_url = EXCLUDED.base_url, model = EXCLUDED.model,
		    api_key = EXCLUDED.api_key, updated_at = now()
	`, strings.TrimSpace(config.ProviderKey), strings.TrimSpace(config.BaseURL), strings.TrimSpace(config.Model), config.APIKey)
	if err != nil {
		return fmt.Errorf("upsert Model Provider Configuration: %w", err)
	}
	return nil
}

func (s *Store) LoadModelProviderConfigs(ctx context.Context) (map[string]agentrun.ModelProviderConfig, error) {
	rows, err := s.database.Query(ctx, `
		SELECT provider_key, base_url, model, api_key
		FROM model_provider_configs
		ORDER BY provider_key
	`)
	if err != nil {
		return nil, fmt.Errorf("load Model Provider Configurations: %w", err)
	}
	defer rows.Close()
	loaded := make(map[string]agentrun.ModelProviderConfig)
	for rows.Next() {
		var config agentrun.ModelProviderConfig
		if err := rows.Scan(&config.ProviderKey, &config.BaseURL, &config.Model, &config.APIKey); err != nil {
			return nil, fmt.Errorf("scan Model Provider Configuration: %w", err)
		}
		loaded[config.ProviderKey] = config
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load Model Provider Configurations: %w", err)
	}
	return loaded, nil
}

func (s *Store) ListModelProviderConfigViews(ctx context.Context) ([]agentrun.ModelProviderConfigView, error) {
	rows, err := s.database.Query(ctx, `
		SELECT provider_key, base_url, model, api_key, updated_at
		FROM model_provider_configs
		ORDER BY provider_key
	`)
	if err != nil {
		return nil, fmt.Errorf("list Model Provider Configurations: %w", err)
	}
	defer rows.Close()
	var views []agentrun.ModelProviderConfigView
	for rows.Next() {
		var view agentrun.ModelProviderConfigView
		var key string
		var updatedAt time.Time
		if err := rows.Scan(&view.ProviderKey, &view.BaseURL, &view.Model, &key, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan Model Provider Configuration view: %w", err)
		}
		view.Configured = true
		view.UpdatedAt = &updatedAt
		view.KeyConfigured, view.MaskedKey = redactKey(key)
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list Model Provider Configurations: %w", err)
	}
	return views, nil
}

func (s *Store) UpsertConnectorConfig(ctx context.Context, config agentrun.ConnectorConfig) error {
	if strings.TrimSpace(config.ConnectorKey) == "" {
		return fmt.Errorf("Connector Configuration key is required")
	}
	if strings.TrimSpace(config.BaseURL) == "" {
		return fmt.Errorf("Connector Base URL is required")
	}
	_, err := s.database.Exec(ctx, `
		INSERT INTO connector_configs (connector_key, base_url, api_key, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (connector_key) DO UPDATE
		SET base_url = EXCLUDED.base_url, api_key = EXCLUDED.api_key, updated_at = now()
	`, strings.TrimSpace(config.ConnectorKey), strings.TrimSpace(config.BaseURL), config.APIKey)
	if err != nil {
		return fmt.Errorf("upsert Connector Configuration: %w", err)
	}
	return nil
}

func (s *Store) LoadConnectorConfigs(ctx context.Context) (map[string]agentrun.ConnectorConfig, error) {
	rows, err := s.database.Query(ctx, `
		SELECT connector_key, base_url, api_key
		FROM connector_configs
		ORDER BY connector_key
	`)
	if err != nil {
		return nil, fmt.Errorf("load Connector Configurations: %w", err)
	}
	defer rows.Close()
	loaded := make(map[string]agentrun.ConnectorConfig)
	for rows.Next() {
		var config agentrun.ConnectorConfig
		if err := rows.Scan(&config.ConnectorKey, &config.BaseURL, &config.APIKey); err != nil {
			return nil, fmt.Errorf("scan Connector Configuration: %w", err)
		}
		loaded[config.ConnectorKey] = config
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load Connector Configurations: %w", err)
	}
	return loaded, nil
}

func (s *Store) ListConnectorConfigViews(ctx context.Context) ([]agentrun.ConnectorConfigView, error) {
	rows, err := s.database.Query(ctx, `
		SELECT connector_key, base_url, api_key, updated_at
		FROM connector_configs
		ORDER BY connector_key
	`)
	if err != nil {
		return nil, fmt.Errorf("list Connector Configurations: %w", err)
	}
	defer rows.Close()
	var views []agentrun.ConnectorConfigView
	for rows.Next() {
		var view agentrun.ConnectorConfigView
		var key string
		var updatedAt time.Time
		if err := rows.Scan(&view.ConnectorKey, &view.BaseURL, &key, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan Connector Configuration view: %w", err)
		}
		view.Configured = true
		view.UpdatedAt = &updatedAt
		view.KeyConfigured, view.MaskedKey = redactKey(key)
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list Connector Configurations: %w", err)
	}
	return views, nil
}

func redactKey(key string) (bool, string) {
	if key == "" {
		return false, ""
	}
	if len(key) <= 4 {
		return true, "***"
	}
	return true, "***" + key[len(key)-4:]
}
