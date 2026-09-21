package configuration

import (
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

const WechatCode = "wechat_miniapp"

var ErrUnavailable = errors.New("WeChat application configuration unavailable")

// Credentials are private server configuration, never a public wire DTO.
type Credentials struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

func (c Credentials) Validate() error {
	for _, s := range []string{c.AppID, c.AppSecret} {
		if len(s) == 0 || len(s) > 256 || strings.IndexFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
			return ErrUnavailable
		}
	}
	return nil
}

type Entry struct {
	ID          string
	Credentials Credentials
	UpdatedAt   time.Time
}

func NewEntry(c Credentials, now time.Time) (Entry, error) {
	if err := c.Validate(); err != nil {
		return Entry{}, err
	}
	return Entry{ID: uuid.NewString(), Credentials: c, UpdatedAt: now.UTC()}, nil
}
