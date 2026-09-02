package authtoken

import (
	"time"

	"github.com/google/uuid"
)

// Purpose will be used to determine what kind of auth token will be minted and consumed
type Purpose string

// PurposeEmailVerification, PurposeForgotPurpose, PurposeLoginCode will determine what kind of token will be minted and consumed
const (
	PurposeEmailVerification Purpose = "email_verification"
	PurposeForgotPassword    Purpose = "forgot_password"
	PurposeLoginCode         Purpose = "login_code"
)

// Valid will check if input purpose is valid
func (p Purpose) Valid() bool {
	switch p {
	case PurposeEmailVerification, PurposeForgotPassword, PurposeLoginCode:
		return true
	}
	return false
}

// Token is authToken that will be used for PurposeEmailVerification PurposeForgotPassword and PurposeLoginCode
type Token struct {
	ID        uuid.UUID
	HikerID   uuid.UUID
	Hash      string
	Purpose   Purpose
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}
