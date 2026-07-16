package hiker

import (
	"time"

	"github.com/google/uuid"
)

// Experience is a hiker's self-reported skill level.
type Experience string

// Known experience levels, mirroring the hikers.experience CHECK constraint.
const (
	ExperienceBeginner     Experience = "beginner"
	ExperienceIntermediate Experience = "intermediate"
	ExperienceExperienced  Experience = "experienced"
)

// Valid reports whether e is one of the known experience levels.
func (e Experience) Valid() bool {
	switch e {
	case ExperienceBeginner, ExperienceIntermediate, ExperienceExperienced:
		return true
	}
	return false
}

// Hiker is a registered member of Muster. PasswordHash and DeletedAt
// never serialize to JSON.
type Hiker struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Name         string     `json:"name"`
	Experience   Experience `json:"experience"`
	HomeArea     *string    `json:"home_area,omitempty"`
	Bio          *string    `json:"bio,omitempty"`
	Gender       *string    `json:"gender,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"-"`
}

// RefreshToken is a stored refresh-token record. Only the SHA-256 hash
// of the token is persisted; the raw value lives in the client's cookie.
type RefreshToken struct {
	ID        uuid.UUID
	HikerID   uuid.UUID
	TokenHash string
	Platform  Platform
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Session is the result of a successful authentication: the hiker plus
// a signed access JWT and a raw refresh token, both destined for
// cookies. The raw refresh token is never persisted.
type Session struct {
	Hiker        *Hiker
	AccessToken  string
	RefreshToken string
}
