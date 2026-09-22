package identity

import (
	"context"
	v1 "github.com/meierlink88/tidewise-ai/user-service/backend/api/user/v1"
	"time"
)

type LoginRequest struct {
	PhoneCode            string `json:"phone_code,omitempty"`
	PrivacyVersion       string `json:"privacy_version,omitempty"`
	Code                 string `json:"code"`
	PreviousSessionToken string `json:"previous_session_token,omitempty"`
}
type SessionRequest struct {
	SessionToken string `json:"session_token"`
}
type AvatarResponse struct {
	Data        []byte `json:"data"`
	ContentType string `json:"content_type"`
}
type NicknameRequest struct {
	AvatarData   []byte `json:"avatar_data,omitempty"`
	SessionToken string `json:"session_token"`
	Nickname     string `json:"nickname"`
}
type UserResponse struct {
	Nickname     string    `json:"nickname"`
	UserID       string    `json:"user_id"`
	Status       string    `json:"status"`
	ExpiresAt    time.Time `json:"expires_at"`
	SessionToken string    `json:"session_token,omitempty"`
}
type RevokeResponse struct {
	Revoked bool `json:"revoked"`
}
type Error = v1.Error

type Service interface {
	Avatar(context.Context, SessionRequest) (AvatarResponse, error)
	UpdateNickname(context.Context, NicknameRequest) (UserResponse, error)
	Login(context.Context, LoginRequest) (UserResponse, error)
	Verify(context.Context, SessionRequest) (UserResponse, error)
	Revoke(context.Context, SessionRequest) (RevokeResponse, error)
}
