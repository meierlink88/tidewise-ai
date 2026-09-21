package identity

import (
	"context"
	"time"
)

type LoginRequest struct {
	PhoneCode      string `json:"phone_code,omitempty"`
	PrivacyVersion string `json:"privacy_version,omitempty"`
	Code           string `json:"code"`
}
type AvatarResponse struct {
	Data        []byte `json:"data"`
	ContentType string `json:"content_type"`
}
type NicknameRequest struct {
	AvatarData []byte `json:"avatar_data,omitempty"`
	Nickname   string `json:"nickname"`
}
type Profile struct {
	UserID    string    `json:"user_id"`
	Nickname  string    `json:"nickname"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	Token     string    `json:"session_token,omitempty"`
}
type LogoutResult struct {
	Revoked bool `json:"revoked"`
}
type Service interface {
	Avatar(context.Context, string) (AvatarResponse, error)
	Login(context.Context, LoginRequest, string) (Profile, error)
	Me(context.Context, string) (Profile, error)
	Logout(context.Context, string) (LogoutResult, error)
	Nickname(context.Context, NicknameRequest, string) (Profile, error)
}
