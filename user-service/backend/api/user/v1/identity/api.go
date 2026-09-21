package identity

import (
	"context"
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
type NicknameRequest struct {
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
type Error struct {
	Status int
	Code   string
}

func (e *Error) Error() string { return e.Code }

type Service interface {
	UpdateNickname(context.Context, NicknameRequest) (UserResponse, error)
	Login(context.Context, LoginRequest) (UserResponse, error)
	Verify(context.Context, SessionRequest) (UserResponse, error)
	Revoke(context.Context, SessionRequest) (RevokeResponse, error)
}
