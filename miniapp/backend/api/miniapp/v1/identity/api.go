package identity

import (
	"context"
	"time"
)

type LoginRequest struct {
	Code string `json:"code"`
}
type NicknameRequest struct {
	Nickname string `json:"nickname"`
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
	Login(context.Context, LoginRequest, string) (Profile, error)
	Me(context.Context, string) (Profile, error)
	Logout(context.Context, string) (LogoutResult, error)
	Nickname(context.Context, NicknameRequest, string) (Profile, error)
}
