package identity

import (
	"context"
	"time"
)

type LoginRequest struct {
	Code                 string `json:"code"`
	PreviousSessionToken string `json:"previous_session_token,omitempty"`
}
type SessionRequest struct {
	SessionToken string `json:"session_token"`
}
type UserResponse struct {
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
	Login(context.Context, LoginRequest) (UserResponse, error)
	Verify(context.Context, SessionRequest) (UserResponse, error)
	Revoke(context.Context, SessionRequest) (RevokeResponse, error)
}
