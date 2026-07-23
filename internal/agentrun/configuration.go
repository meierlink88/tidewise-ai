package agentrun

type ModelProviderConfig struct {
	ProviderKey string
	BaseURL     string
	Model       string
	APIKey      string
}

type ModelProviderConfigView struct {
	ProviderKey   string `json:"provider_key"`
	BaseURL       string `json:"base_url"`
	Model         string `json:"model"`
	KeyConfigured bool   `json:"key_configured"`
	MaskedKey     string `json:"masked_key,omitempty"`
}

type ConnectorConfig struct {
	ConnectorKey string
	BaseURL      string
	APIKey       string
}

type ConnectorConfigView struct {
	ConnectorKey  string `json:"connector_key"`
	BaseURL       string `json:"base_url"`
	KeyConfigured bool   `json:"key_configured"`
	MaskedKey     string `json:"masked_key,omitempty"`
}
