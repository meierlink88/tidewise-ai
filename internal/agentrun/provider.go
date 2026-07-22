package agentrun

type ProviderConfig struct {
	Key     string
	BaseURL string
	Model   string
	APIKey  string
}

type ProviderConfigView struct {
	Key           string `json:"key"`
	BaseURL       string `json:"base_url"`
	Model         string `json:"model,omitempty"`
	KeyConfigured bool   `json:"key_configured"`
	MaskedKey     string `json:"masked_key,omitempty"`
}
