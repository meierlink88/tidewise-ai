package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
)

func (s *Store) UpsertProviderConfig(ctx context.Context, config agentrun.ProviderConfig) error {
	if strings.TrimSpace(config.Key) == "" {
		return fmt.Errorf("Provider configuration key is required")
	}
	if strings.TrimSpace(config.BaseURL) == "" {
		return fmt.Errorf("Provider Base URL is required")
	}
	_, err := s.database.Exec(ctx, `
		INSERT INTO provider_configs (provider_key, base_url, model, api_key, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (provider_key) DO UPDATE
		SET base_url = EXCLUDED.base_url, model = EXCLUDED.model,
		    api_key = EXCLUDED.api_key, updated_at = now()
	`, config.Key, strings.TrimSpace(config.BaseURL), strings.TrimSpace(config.Model), config.APIKey)
	if err != nil {
		return fmt.Errorf("upsert Provider configuration: %w", err)
	}
	return nil
}

func (s *Store) LoadProviderConfigs(ctx context.Context) (map[string]agentrun.ProviderConfig, error) {
	rows, err := s.database.Query(ctx, `SELECT provider_key, base_url, model, api_key FROM provider_configs ORDER BY provider_key`)
	if err != nil {
		return nil, fmt.Errorf("load Provider configurations: %w", err)
	}
	defer rows.Close()
	loaded := make(map[string]agentrun.ProviderConfig)
	for rows.Next() {
		var config agentrun.ProviderConfig
		if err := rows.Scan(&config.Key, &config.BaseURL, &config.Model, &config.APIKey); err != nil {
			return nil, fmt.Errorf("scan Provider configuration: %w", err)
		}
		loaded[config.Key] = config
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load Provider configurations: %w", err)
	}
	return loaded, nil
}

func (s *Store) ListProviderConfigViews(ctx context.Context) ([]agentrun.ProviderConfigView, error) {
	rows, err := s.database.Query(ctx, `SELECT provider_key, base_url, model, api_key FROM provider_configs ORDER BY provider_key`)
	if err != nil {
		return nil, fmt.Errorf("list Provider configurations: %w", err)
	}
	defer rows.Close()
	var views []agentrun.ProviderConfigView
	for rows.Next() {
		var view agentrun.ProviderConfigView
		var key string
		if err := rows.Scan(&view.Key, &view.BaseURL, &view.Model, &key); err != nil {
			return nil, fmt.Errorf("scan Provider configuration view: %w", err)
		}
		view.KeyConfigured = key != ""
		if len(key) >= 4 {
			view.MaskedKey = "***" + key[len(key)-4:]
		} else if key != "" {
			view.MaskedKey = "***"
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list Provider configurations: %w", err)
	}
	return views, nil
}
