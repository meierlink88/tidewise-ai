package conf

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Address         string `yaml:"address"`
	SessionTTLHours int    `yaml:"session_ttl_hours"`
	DatabaseURL     string `yaml:"-"`
	DatabaseName    string `yaml:"-"`
	ServiceToken    string `yaml:"-"`
	WechatAppID     string `yaml:"-"`
	WechatAppSecret string `yaml:"-"`
}

func Load(path string, server bool) (Config, error) { return load(path, server, os.Getenv) }
func load(path string, server bool, env func(string) string) (Config, error) {
	c := Config{Address: "127.0.0.1:9015", SessionTTLHours: 168}
	if path != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			return c, errors.New("cannot read user configuration")
		}
		decoder := yaml.NewDecoder(strings.NewReader(string(body)))
		decoder.KnownFields(true)
		if decoder.Decode(&c) != nil {
			return c, errors.New("invalid user configuration")
		}
	}
	c.DatabaseURL = env("USER_DATABASE_URL")
	c.DatabaseName = env("USER_DATABASE_NAME")
	parsed, err := url.Parse(c.DatabaseURL)
	if err != nil || parsed == nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || !strings.HasPrefix(c.DatabaseName, "tidewise_user_") || strings.TrimPrefix(parsed.Path, "/") != c.DatabaseName || parsed.Host == "" {
		return c, errors.New("USER_DATABASE_URL must target the explicit tidewise_user_* USER_DATABASE_NAME")
	}
	// libpq query parameters can override the URL database/host. Only permit
	// connection options that cannot redirect this service to another database.
	for key := range parsed.Query() {
		switch key {
		case "sslmode", "sslrootcert", "sslcert", "sslkey", "connect_timeout", "application_name":
		default:
			return c, errors.New("unsupported User database URL option")
		}
	}
	if !server {
		return c, nil
	}
	c.ServiceToken = env("USER_SERVICE_TOKEN")
	c.WechatAppID = env("WECHAT_APP_ID")
	c.WechatAppSecret = env("WECHAT_APP_SECRET")
	if address := env("USER_HTTP_ADDRESS"); address != "" {
		c.Address = address
	}
	if _, _, err = net.SplitHostPort(c.Address); err != nil || c.SessionTTLHours < 1 || c.SessionTTLHours > 720 {
		return c, errors.New("invalid user address or session TTL")
	}
	if len(c.ServiceToken) < 32 || strings.TrimSpace(c.ServiceToken) != c.ServiceToken || strings.ContainsAny(c.ServiceToken, "\r\n") || strings.TrimSpace(c.WechatAppID) == "" || strings.TrimSpace(c.WechatAppSecret) == "" {
		return c, errors.New("missing or invalid User Service credentials")
	}
	return c, nil
}
func (c Config) SessionTTL() time.Duration { return time.Duration(c.SessionTTLHours) * time.Hour }
