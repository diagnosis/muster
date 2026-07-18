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

// JoinInput carries a hiker's request to join an outing.
type JoinInput struct {
	Role         Role
	SeatsOffered int
	Guests       int
	Note         *string
}

// RequestJoin asks to join an outing. Declined is terminal; withdrawn may re-request; capacity is checked at acceptance, not here.
func (s *Service) RequestJoin(ctx context.Context, hikerID, outingID uuid.UUID, in JoinInput) (*JoinRequest, error) {
	if !in.Role.Valid() {
		return nil, apperr.BadRequest("invalid role", "invalid role")
	}
	if in.Role == RoleRider && in.SeatsOffered != 0 {
		return nil, apperr.BadRequest("rider cannot offer seats", "rider cannot offer seats")
	}
	if in.Role == RoleDriver && in.SeatsOffered < 1 {
		return nil, apperr.BadRequest("driver needs to offer at least one seat", "one seat needs to be offered as driver")
	}
	if in.Guests < 0 || in.Guests > 3 {
		return nil, apperr.BadRequest("number of guests should be between 0-3", "number guest violation")
	}
	o, err := s.store.GetOuting(ctx, outingID)
	if err != nil {
		return nil, err
	}
	if o.Status != StatusOpen {
		return nil, apperr.Conflict("outing is not open", "outing closed or cancelled")
	}
	if time.Until(o.StartsAt) <= 0 {
		return nil, apperr.Conflict("outing already started", "past outing")
	}
	if o.HostID == hikerID {
		return nil, apperr.BadRequest("cannot join the event you created", "host cannot join the event they created")
	}

	joinRequest, err := s.store.GetJoinRequestByHiker(ctx, o.ID, hikerID)
	if err != nil {
		if se, ok := apperr.AsStatusErr(err); ok && se.Status == apperr.CodeNotFound {
			joinRequest = &JoinRequest{
				ID:           uuid.New(),
				OutingID:     o.ID,
				HikerID:      hikerID,
				Status:       RequestStatusRequested,
				Role:         in.Role,
				SeatsOffered: in.SeatsOffered,
				Guests:       in.Guests,
				Note:         in.Note,
			}
			if err = s.store.CreateJoinRequest(ctx, joinRequest); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		switch joinRequest.Status {
		case RequestStatusWithdrawn:
			if err := s.store.SetJoinRequestStatus(ctx, joinRequest.ID, RequestStatusRequested); err != nil {
				return nil, err
			}
			joinRequest.Status = RequestStatusRequested
		case RequestStatusDeclined:
			return nil, apperr.Conflict("your request was declined", "declined is terminal")
		default: // requested or accepted
			return nil, apperr.Conflict("you already have an active request", "duplicate request")
		}
	}
	return joinRequest, nil
}
