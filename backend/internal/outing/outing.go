// Package outing contains Muster's core domain: outings, join
// requests, and the rules that govern them.
package outing

import (
	"time"

	"github.com/google/uuid"
)

// Difficulty is the physical difficulty a host assigns to an outing.
type Difficulty string

// Known difficulties, mirroring the outings.difficulty CHECK constraint.
const (
	DifficultyEasy     Difficulty = "easy"
	DifficultyModerate Difficulty = "moderate"
	DifficultyHard     Difficulty = "hard"
)

// Valid reports whether d is a known difficulty.
func (d Difficulty) Valid() bool {
	switch d {
	case DifficultyEasy, DifficultyModerate, DifficultyHard:
		return true
	}
	return false
}

// Pace is the group pace a host sets for an outing. Pace is a property
// of the trip, not the hiker: groups set pace, people adapt.
type Pace string

// Known paces, mirroring the outings.pace CHECK constraint.
const (
	PaceRelaxed  Pace = "relaxed"
	PaceModerate Pace = "moderate"
	PaceFast     Pace = "fast"
)

// Valid reports whether p is a known pace.
func (p Pace) Valid() bool {
	switch p {
	case PaceRelaxed, PaceModerate, PaceFast:
		return true
	}
	return false
}

// Status is the lifecycle state of an outing. "Past" is derived
// from StartsAt, never stored as a status.
type Status string

// Known outing statuses, mirroring the outings.status CHECK constraint.
const (
	StatusOpen      Status = "open"
	StatusCancelled Status = "cancelled"
)

// Valid reports whether o is a known outing status.
func (s Status) Valid() bool {
	switch s {
	case StatusOpen, StatusCancelled:
		return true
	}
	return false
}

// RequestStatus is the state of a hiker's request to join an outing.
// Declined is terminal; withdrawn may re-request.
type RequestStatus string

// Known request statuses, mirroring the join_requests.status CHECK
// constraint. (v1 adds waiting_list — see roadmap.)
const (
	RequestStatusRequested RequestStatus = "requested"
	RequestStatusAccepted  RequestStatus = "accepted"
	RequestStatusDeclined  RequestStatus = "declined"
	RequestStatusWithdrawn RequestStatus = "withdrawn"
)

// Valid reports whether rs is a known request status.
func (rs RequestStatus) Valid() bool {
	switch rs {
	case RequestStatusRequested, RequestStatusAccepted,
		RequestStatusDeclined, RequestStatusWithdrawn:
		return true
	}
	return false
}

// Outing is a planned group hike: a convergence of hikers on a meeting
// point at a start time. The host is not represented in join_requests;
// roster = host + accepted requests.
type Outing struct {
	ID          uuid.UUID  `json:"id"`
	HostID      uuid.UUID  `json:"host_id"`
	Title       string     `json:"title"`
	Destination string     `json:"destination"`
	MeetLabel   string     `json:"meet_label"`
	MeetLat     *float64   `json:"meet_lat,omitempty"`
	MeetLng     *float64   `json:"meet_lng,omitempty"`
	StartsAt    time.Time  `json:"starts_at"`
	MaxSize     int        `json:"max_size"` // total headcount, host included
	Difficulty  Difficulty `json:"difficulty"`
	Pace        Pace       `json:"pace"`
	Notes       *string    `json:"notes,omitempty"`
	Status      Status     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// JoinRequest links a hiker to an outing. It only exists as that link;
// all rules governing it are outing rules.
type JoinRequest struct {
	ID        uuid.UUID     `json:"id"`
	OutingID  uuid.UUID     `json:"outing_id"`
	HikerID   uuid.UUID     `json:"hiker_id"`
	Status    RequestStatus `json:"status"`
	Note      *string       `json:"note,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// Member is a roster read-model: the public slice of a hiker as seen
// on an outing (join_requests joined with hikers at the query level).
type Member struct {
	HikerID    uuid.UUID `json:"hiker_id"`
	Name       string    `json:"name"`
	Experience string    `json:"experience"`
}

// Detail is the full view of one outing for one viewer.
type Detail struct {
	Outing    Outing       `json:"outing"`
	Host      Member       `json:"host"`
	Roster    []Member     `json:"roster"` // accepted only; host NOT included
	MyRequest *JoinRequest `json:"my_request,omitempty"`
	SpotsLeft int          `json:"spots_left"` // max_size - (accepted + 1)
}

// MyOutings groups the outings a hiker hosts and the ones they joined.
type MyOutings struct {
	Hosting []Outing `json:"hosting"`
	Joined  []Outing `json:"joined"`
}
