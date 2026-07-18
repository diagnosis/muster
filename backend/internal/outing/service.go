package outing

import (
	"context"
	"strings"
	"time"

	"github.com/diagnosis/go-toolkit/v2/apperr"
	"github.com/diagnosis/go-toolkit/v2/validator"
	"github.com/google/uuid"
)

// Storage is what the outing service needs from persistence.
// Implemented by postgres.OutingStore.
type Storage interface {
	CreateOuting(ctx context.Context, o *Outing) error
	GetOuting(ctx context.Context, id uuid.UUID) (*Outing, error)
	GetJoinRequestByHiker(ctx context.Context, outingID, hikerID uuid.UUID) (*JoinRequest, error)

	CreateJoinRequest(ctx context.Context, r *JoinRequest) error
	SetJoinRequestStatus(ctx context.Context, id uuid.UUID, s RequestStatus) error // for the withdrawn→requested flip
}

// Service implements outing business rules over a Storage.
type Service struct {
	store Storage
}

// NewService returns a Service backed by store.
func NewService(store Storage) *Service {
	return &Service{
		store: store,
	}
}

// CreateInput carries the fields required to schedule an outing.
type CreateInput struct {
	Title            string     `json:"title"`
	Destination      string     `json:"destination"`
	MeetLabel        string     `json:"meet_label"`
	HostSeats        int        `json:"host_seats"`
	CostPerSeatCents int        `json:"cost_per_seat_cents"`
	MeetLat          *float64   `json:"meet_lat"`
	MeetLng          *float64   `json:"meet_lng"`
	StartsAt         time.Time  `json:"starts_at"`
	MaxSize          int        `json:"max_size"`
	Difficulty       Difficulty `json:"difficulty"`
	Pace             Pace       `json:"pace"`
	Notes            *string    `json:"notes,omitempty"`
}

// minLeadTime is how far in advance an outing must be scheduled —
// people need time to muster. leadGrace absorbs clock skew and
// form-filling time.
const (
	minLeadTime = 24 * time.Hour
	leadGrace   = 5 * time.Minute
)

// Create schedules a new outing hosted by hostID. Outings must start
// at least 24 hours in the future so people have time to muster, and
// max_size counts the host.
func (s *Service) Create(ctx context.Context, hostID uuid.UUID, in CreateInput) (*Outing, error) {

	title := strings.TrimSpace(in.Title)
	destination := strings.TrimSpace(in.Destination)
	meetLabel := strings.TrimSpace(in.MeetLabel)

	v := validator.New()
	v.Required("title", title)
	v.Required("destination", destination)
	v.Required("meet_label", meetLabel)
	verr := v.Errors()
	if verr != nil {
		return nil, verr
	}

	if time.Until(in.StartsAt) < minLeadTime-leadGrace {
		return nil, apperr.BadRequest("outing has to be at least 24 hours in advance", "under 24h lead time")
	}
	if in.MaxSize < 2 {
		return nil, apperr.BadRequest("outing size has to be at least 2", "min members violation")
	}
	if in.HostSeats < 0 {
		return nil, apperr.BadRequest("host seats cannot be negative", "negative host seats")
	}
	if !in.Pace.Valid() {
		return nil, apperr.BadRequest("invalid pace input", "invalid pace input")
	}
	if !in.Difficulty.Valid() {
		return nil, apperr.BadRequest("invalid difficulty input", "invalid difficulty")
	}
	if in.CostPerSeatCents < 0 {
		return nil, apperr.BadRequest("invalid cost per seat input", "invalid seat cost")
	}

	o := &Outing{
		ID:               uuid.New(),
		HostID:           hostID,
		Title:            title,
		Destination:      destination,
		MeetLabel:        meetLabel,
		MeetLat:          in.MeetLat,
		MeetLng:          in.MeetLng,
		StartsAt:         in.StartsAt,
		MaxSize:          in.MaxSize,
		Difficulty:       in.Difficulty,
		Pace:             in.Pace,
		Notes:            in.Notes,
		Status:           StatusOpen,
		HostSeats:        in.HostSeats,
		CostPerSeatCents: in.CostPerSeatCents,
	}

	return o, s.store.CreateOuting(ctx, o)

}
