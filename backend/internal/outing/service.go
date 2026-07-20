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
	SetOutingStatus(ctx context.Context, id uuid.UUID, s Status) error
	ListUpcoming(ctx context.Context, now time.Time) ([]Outing, error)

	CreateJoinRequest(ctx context.Context, r *JoinRequest) error
	GetJoinRequest(ctx context.Context, id uuid.UUID) (*JoinRequest, error)
	GetJoinRequestByHiker(ctx context.Context, outingID, hikerID uuid.UUID) (*JoinRequest, error)
	SetJoinRequestStatus(ctx context.Context, id uuid.UUID, s RequestStatus) error
	ListJoinRequests(ctx context.Context, outingID uuid.UUID, status RequestStatus) ([]JoinRequest, error)
	// AcceptIfCapacity flips a request from requested to accepted only if
	// it fits, atomically: a driver needs cap room (people + 1 + guests
	// <= max_size); a rider needs cap room AND seat room (people + 1 +
	// guests <= min(seat capacity, max_size)). Full => apperr.Conflict.
	AcceptIfCapacity(ctx context.Context, requestID uuid.UUID) error
}

// Service implements outing business rules over a Storage.
type Service struct {
	store Storage
}

// NewService returns a Service backed by store.
func NewService(store Storage) *Service {
	return &Service{store: store}
}

// minLeadTime is how far in advance an outing must be scheduled —
// people need time to muster. leadGrace absorbs clock skew and
// form-filling time.
const (
	minLeadTime = 24 * time.Hour
	leadGrace   = 5 * time.Minute
)

// maxGuests caps the unregistered +1s one member may bring.
const maxGuests = 3

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

// Create schedules a new outing hosted by hostID. Outings must start at
// least 24 hours in the future so people have time to muster. MaxSize
// caps total headcount including the host; HostSeats may be 0 (the
// host needs a ride); cost is display-only in v0.
func (s *Service) Create(ctx context.Context, hostID uuid.UUID, in CreateInput) (*Outing, error) {
	title := strings.TrimSpace(in.Title)
	destination := strings.TrimSpace(in.Destination)
	meetLabel := strings.TrimSpace(in.MeetLabel)

	v := validator.New()
	v.Required("title", title)
	v.Required("destination", destination)
	v.Required("meet_label", meetLabel)
	if verr := v.Errors(); verr != nil {
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
		HostSeats:        in.HostSeats,
		CostPerSeatCents: in.CostPerSeatCents,
		Difficulty:       in.Difficulty,
		Pace:             in.Pace,
		Notes:            in.Notes,
		Status:           StatusOpen,
	}

	return o, s.store.CreateOuting(ctx, o)
}

// Cancel marks an open, future outing as cancelled. Host-only. Past
// outings cannot be cancelled — they're history, not state.
func (s *Service) Cancel(ctx context.Context, hostID, outingID uuid.UUID) error {
	o, err := s.store.GetOuting(ctx, outingID)
	if err != nil {
		return err
	}
	if o.HostID != hostID {
		return apperr.Forbidden("only the host can cancel this outing", "forbidden: host required")
	}
	if o.Status == StatusCancelled {
		return apperr.Conflict("outing is already cancelled", "already cancelled")
	}
	if o.StartsAt.Before(time.Now()) {
		return apperr.BadRequest("cannot cancel a past outing", "outing already started")
	}
	return s.store.SetOutingStatus(ctx, outingID, StatusCancelled)
}

// JoinInput carries a hiker's request to join an outing.
type JoinInput struct {
	Role         Role    `json:"role"`
	SeatsOffered int     `json:"seats_offered"`
	Guests       int     `json:"guests"`
	Note         *string `json:"note"`
}

// RequestJoin asks to join an outing. A first request creates a row in
// requested; a withdrawn request may re-request; declined is terminal;
// an active request conflicts. Hosts cannot join their own outing.
// Capacity is deliberately NOT checked here — it gates acceptance,
// not asking.
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
	if in.Guests < 0 || in.Guests > maxGuests {
		return nil, apperr.BadRequest("number of guests should be between 0-3", "guest count violation")
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
		return nil, apperr.BadRequest("cannot join the event you created", "host self-join")
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
			return joinRequest, nil
		}
		return nil, err
	}

	switch joinRequest.Status {
	case RequestStatusWithdrawn:
		// Re-requesting is legal. v0 keeps the original role/seats/guests.
		if err = s.store.SetJoinRequestStatus(ctx, joinRequest.ID, RequestStatusRequested); err != nil {
			return nil, err
		}
		joinRequest.Status = RequestStatusRequested
		return joinRequest, nil
	case RequestStatusDeclined:
		return nil, apperr.Conflict("your request to this outing was declined", "declined is terminal")
	default: // requested or accepted
		return nil, apperr.Conflict("you already have an active request", "duplicate request")
	}
}

// loadForHostAction fetches a request and its outing, verifying the
// caller is the host and the outing is open.
func (s *Service) loadForHostAction(ctx context.Context, hostID, requestID uuid.UUID) (*JoinRequest, *Outing, error) {
	joinRequest, err := s.store.GetJoinRequest(ctx, requestID)
	if err != nil {
		return nil, nil, err
	}
	o, err := s.store.GetOuting(ctx, joinRequest.OutingID)
	if err != nil {
		return nil, nil, err
	}
	if hostID != o.HostID {
		return nil, nil, apperr.Forbidden("only the event host can do this", "forbidden: host required")
	}
	if o.Status != StatusOpen {
		return nil, nil, apperr.Conflict("event is closed", "event is closed")
	}
	return joinRequest, o, nil
}

// Accept approves a pending join request. Host-only; capacity is
// checked atomically in the store — a rider needs a seat and cap room,
// a driver only cap room.
func (s *Service) Accept(ctx context.Context, hostID, requestID uuid.UUID) error {
	joinRequest, _, err := s.loadForHostAction(ctx, hostID, requestID)
	if err != nil {
		return err
	}
	if joinRequest.Status != RequestStatusRequested {
		return apperr.Conflict("request is not pending", "accept requires requested status")
	}
	return s.store.AcceptIfCapacity(ctx, requestID)
}

// Decline rejects a pending join request. Host-only; declined is
// terminal for this outing.
func (s *Service) Decline(ctx context.Context, hostID, requestID uuid.UUID) error {
	joinRequest, _, err := s.loadForHostAction(ctx, hostID, requestID)
	if err != nil {
		return err
	}
	if joinRequest.Status != RequestStatusRequested {
		return apperr.Conflict("request is not pending", "decline requires requested status")
	}
	return s.store.SetJoinRequestStatus(ctx, requestID, RequestStatusDeclined)
}

// Withdraw pulls the caller's own request, whether pending or already
// accepted. A withdrawing driver takes their seats — the shortage
// shows in Detail; the host resolves it.
func (s *Service) Withdraw(ctx context.Context, hikerID, outingID uuid.UUID) error {
	joinRequest, err := s.store.GetJoinRequestByHiker(ctx, outingID, hikerID)
	if err != nil {
		return err
	}
	if joinRequest.Status != RequestStatusRequested && joinRequest.Status != RequestStatusAccepted {
		return apperr.Conflict("nothing to withdraw", "withdraw requires requested or accepted")
	}
	return s.store.SetJoinRequestStatus(ctx, joinRequest.ID, RequestStatusWithdrawn)
}

// RemoveMember removes an accepted member from the roster. Host-only;
// v0 reuses the withdrawn status for host removals.
func (s *Service) RemoveMember(ctx context.Context, hostID, requestID uuid.UUID) error {
	joinRequest, _, err := s.loadForHostAction(ctx, hostID, requestID)
	if err != nil {
		return err
	}
	if joinRequest.Status != RequestStatusAccepted {
		return apperr.Conflict("member is not on the roster", "remove requires accepted status")
	}
	return s.store.SetJoinRequestStatus(ctx, requestID, RequestStatusWithdrawn)
}

// ListUpcoming returns open outings that start in the future, soonest
// first. The service owns the clock; the store just filters against
// the time it's given.
func (s *Service) ListUpcoming(ctx context.Context) ([]Outing, error) {
	return s.store.ListUpcoming(ctx, time.Now())
}

// PendingRequests returns the outing's pending join requests, oldest
// first — the host's inbox, host-only. No outing-state gate: reads
// return history regardless of cancelled/past status.
func (s *Service) PendingRequests(ctx context.Context, hostID, outingID uuid.UUID) ([]JoinRequest, error) {
	o, err := s.store.GetOuting(ctx, outingID)
	if err != nil {
		return nil, err
	}
	if o.HostID != hostID {
		return nil, apperr.Forbidden("only the host can view pending requests", "forbidden: non-host listing pending requests")
	}
	return s.store.ListJoinRequests(ctx, outingID, RequestStatusRequested)
}
