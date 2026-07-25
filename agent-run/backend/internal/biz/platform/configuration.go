package agentrun

import "time"

type ModelProviderConfig struct {
	ProviderKey string
	BaseURL     string
	Model       string
	APIKey      string
}

type ModelProviderConfigView struct {
	ProviderKey   string     `json:"provider_key"`
	BaseURL       string     `json:"base_url"`
	Model         string     `json:"model"`
	Configured    bool       `json:"configured"`
	KeyConfigured bool       `json:"key_configured"`
	MaskedKey     string     `json:"masked_key,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type ConnectorConfig struct {
	ConnectorKey string
	BaseURL      string
	APIKey       string
}

type ConnectorConfigView struct {
	ConnectorKey  string     `json:"connector_key"`
	BaseURL       string     `json:"base_url"`
	Configured    bool       `json:"configured"`
	KeyConfigured bool       `json:"key_configured"`
	MaskedKey     string     `json:"masked_key,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}
