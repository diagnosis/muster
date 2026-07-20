package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/diagnosis/go-toolkit/v2/apperr"
	"github.com/diagnosis/muster/internal/outing"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutingStore implements outing.Storage against Postgres.
type OutingStore struct {
	pool *pgxpool.Pool
}

// NewOutingStore returns a store backed by the given pool.
func NewOutingStore(pool *pgxpool.Pool) *OutingStore {
	return &OutingStore{pool: pool}
}

var _ outing.Storage = (*OutingStore)(nil)

// CreateOuting inserts a new outing, scanning DB-generated timestamps
// back into Outings
func (s *OutingStore) CreateOuting(ctx context.Context, o *outing.Outing) error {
	q := `
	INSERT INTO outings
		(id, host_id, title, destination, meet_label, meet_lat,
		 meet_lng, starts_at, max_size, host_seats, cost_per_seat_cents,
		 difficulty, pace, notes, status)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	RETURNING created_at, updated_at`

	err := s.pool.QueryRow(ctx, q,
		o.ID, o.HostID, o.Title, o.Destination, o.MeetLabel,
		o.MeetLat, o.MeetLng, o.StartsAt, o.MaxSize, o.HostSeats,
		o.CostPerSeatCents, o.Difficulty, o.Pace, o.Notes, o.Status,
	).Scan(&o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return apperr.Database("could not create outing", "insert outings failed", err)
	}
	return nil
}

// GetOuting returns the outing with given id
func (s *OutingStore) GetOuting(ctx context.Context, id uuid.UUID) (*outing.Outing, error) {
	q := `
		SELECT id, host_id, title, destination, meet_label, meet_lat, meet_lng,
			starts_at, max_size, host_seats, cost_per_seat_cents, difficulty, pace, notes, status,
			created_at, updated_at
		FROM outings 
		WHERE id = $1
`
	o, err := scanOuting(s.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("outing not found", "no row for id")
		}
		return nil, apperr.Database("could not load outing", "select outings by id failed", err)
	}
	return o, nil
}

// ListUpcoming returns open outings starting after now, soonest first.
func (s *OutingStore) ListUpcoming(ctx context.Context, now time.Time) ([]outing.Outing, error) {
	q := `
	SELECT id, host_id, title, destination, meet_label, meet_lat, meet_lng,
			starts_at, max_size, host_seats, cost_per_seat_cents, difficulty, pace, notes, status,
			created_at, updated_at
		FROM outings
	WHERE status = 'open' AND starts_at > $1
	ORDER BY starts_at
`
	rows, err := s.pool.Query(ctx, q, now)
	if err != nil {
		return nil, apperr.Database("failed to list outings", "select upcoming outings failed", err)
	}
	defer rows.Close()

	outings := []outing.Outing{}
	for rows.Next() {
		o, err := scanOuting(rows)
		if err != nil {
			return nil, apperr.Database("failed to list outings", "scan upcoming outing failed", err)
		}
		outings = append(outings, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Database("failed to list outings", "iterate upcoming outings failed", err)
	}
	return outings, nil
}

// ListJoinRequests returns the outing's requests with the given status, oldest first.
func (s *OutingStore) ListJoinRequests(ctx context.Context, outingID uuid.UUID, status outing.RequestStatus) ([]outing.JoinRequest, error) {
	q := `
	SELECT id, outing_id, hiker_id, status, role, seats_offered, guests, note, created_at, updated_at	
		FROM join_requests 
		WHERE outing_id = $1 AND status = $2
	ORDER BY created_at
`
	rows, err := s.pool.Query(ctx, q, outingID, status)
	if err != nil {
		return nil, apperr.Database("failed to list requests", "select join requests failed", err)
	}
	defer rows.Close()

	jrs := []outing.JoinRequest{}

	for rows.Next() {
		jr, err := scanJoinRequest(rows)
		if err != nil {
			return nil, apperr.Database("failed to list requests", "scan join requests failed", err)
		}
		jrs = append(jrs, *jr)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Database("failed to list requests", "iterate join requests failed", err)
	}
	return jrs, nil
}

// CreateJoinRequest inserts new join requests to table.
func (s *OutingStore) CreateJoinRequest(ctx context.Context, r *outing.JoinRequest) error {
	q := `
		INSERT INTO join_requests 
		(id, outing_id, hiker_id, status, role, seats_offered, guests, note)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING created_at, updated_at
`
	row := s.pool.QueryRow(ctx, q, r.ID, r.OutingID, r.HikerID, r.Status, r.Role, r.SeatsOffered, r.Guests, r.Note)
	err := row.Scan(&r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return apperr.Conflict("request already exists for this outing", "unique violation on uq_outing_hiker")
		}
		return apperr.Database("could not create join request", "insert join_requests failed", err)
	}
	return nil
}

// GetJoinRequest returns the join request with the given id, regardless of status.
func (s *OutingStore) GetJoinRequest(ctx context.Context, id uuid.UUID) (*outing.JoinRequest, error) {
	q := `
		SELECT id, outing_id, hiker_id, status, role, seats_offered, guests, note, created_at, updated_at	
		FROM join_requests WHERE id = $1
`
	row := s.pool.QueryRow(ctx, q, id)
	r, err := scanJoinRequest(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("request not found", "no row for id")
		}
		return nil, apperr.Database("could not load join request", "select join_requests by id failed", err)
	}
	return r, nil
}

// GetJoinRequestByHiker returns the hiker's join request on the given
// outing, regardless of status. At most one row exists per pair
// (uq_outing_hiker).
func (s *OutingStore) GetJoinRequestByHiker(ctx context.Context, outingID, hikerID uuid.UUID) (*outing.JoinRequest, error) {
	q := `
		SELECT id, outing_id, hiker_id, status, role, seats_offered, guests, note, created_at, updated_at	
		FROM join_requests WHERE outing_id = $1 AND hiker_id = $2
`
	row := s.pool.QueryRow(ctx, q, outingID, hikerID)
	r, err := scanJoinRequest(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("request not found", "no row for outing+hiker")
		}
		return nil, apperr.Database("could not load join request", "select join_requests by outing+hiker failed", err)
	}
	return r, nil
}

// SetJoinRequestStatus sets the request's status unconditionally; transition validity is the service's responsibility.
func (s *OutingStore) SetJoinRequestStatus(ctx context.Context, id uuid.UUID, status outing.RequestStatus) error {
	q := `
		UPDATE join_requests 
		SET status = $2,
		    updated_at = now()
		WHERE id = $1
		RETURNING updated_at
`
	var updatedAt time.Time
	err := s.pool.QueryRow(ctx, q, id, status).Scan(&updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("request not found", "no row for id")
		}
		return apperr.Database("could not update join request", "update join_requests by id failed", err)
	}
	return nil
}

// AcceptIfCapacity atomically flips a pending request to accepted iff
// the outing has room: headcount (host + accepted + candidate + guests)
// must fit max_size, and riders must additionally fit seat capacity
// (host_seats + accepted drivers' seats_offered). The count and flip
// happen in one statement so concurrent accepts can't oversell.
// Missing request → NotFound; full or not pending → Conflict.
func (s *OutingStore) AcceptIfCapacity(ctx context.Context, requestID uuid.UUID) error {
	q := `
		UPDATE join_requests r
		SET status = 'accepted', 
    		updated_at = now()
		FROM outings o
		WHERE r.id = $1
  			AND r.status = 'requested'
  			AND o.id = r.outing_id
  		-- 1. Headcount constraint: Current accepted headcount + 1 (the host) + this request (1 + guests) <= max_size
  			AND (
   			 1 + 
    		COALESCE((
       			SELECT SUM(1 + jr.guests) 
        		FROM join_requests jr 
        		WHERE jr.outing_id = r.outing_id 
          			AND jr.status = 'accepted'
    		), 0) + 
    		1 + r.guests
  		) <= o.max_size
 		 -- 2. Vehicle seat capacity constraint: Only applies if the applicant is a passenger ('non-driver')
  		AND (
    		r.role = 'driver' 
    		OR 
    		(
        	1 + 
        	COALESCE((
            	SELECT SUM(1 + jr.guests) 
            	FROM join_requests jr 
            	WHERE jr.outing_id = r.outing_id 
              		AND jr.status = 'accepted'
        	), 0) + 
        1 + r.guests
    	) <= (
        o.host_seats + 
        	COALESCE((
            	SELECT SUM(jr.seats_offered) 
            	FROM join_requests jr 
            	WHERE jr.outing_id = r.outing_id 
              		AND jr.status = 'accepted' 
              		AND jr.role = 'driver'
        ), 0)
    )
  )
RETURNING r.updated_at;
`
	var updatedAt time.Time
	err := s.pool.QueryRow(ctx, q, requestID).Scan(&updatedAt)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return apperr.Database("could not accept request", "conditional update join_requests failed", err)
	}
	// Zero rows: either the request doesn't exist, or it exists but
	// isn't pending / capacity check failed. Distinguish for the caller.
	var exists bool
	checkQ := `SELECT EXISTS (SELECT 1 FROM join_requests WHERE id = $1)`
	if cerr := s.pool.QueryRow(ctx, checkQ, requestID).Scan(&exists); cerr != nil {
		return apperr.Database("could not accept request", "existence check after conditional update failed", cerr)
	}
	if !exists {
		return apperr.NotFound("request not found", "no row for id")
	}
	return apperr.Conflict("outing is full or request is not pending", "capacity or status condition failed")
}

// SetOutingStatus sets the outing's status unconditionally; transition
// validity is the service's responsibility.
func (s *OutingStore) SetOutingStatus(ctx context.Context, id uuid.UUID, status outing.Status) error {
	q := `
		UPDATE outings
		SET status = $2,
		    updated_at = now()
		WHERE id = $1
		RETURNING updated_at`

	var updatedAt time.Time
	err := s.pool.QueryRow(ctx, q, id, status).Scan(&updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("outing not found", "no row for id")
		}
		return apperr.Database("could not update outing", "update outings by id failed", err)
	}
	return nil
}

// ListForHiker returns the outings the hiker hosts and the ones they've joined (accepted requests only), each soonest first.
func (s *OutingStore) ListForHiker(ctx context.Context, hikerID uuid.UUID) (*outing.MyOutings, error) {
	hostingQuery := `
	SELECT id, host_id, title, destination, meet_label, meet_lat, meet_lng,
			starts_at, max_size, host_seats, cost_per_seat_cents, difficulty, pace, notes, status,
			created_at, updated_at
		FROM outings
		WHERE host_id = $1 
		ORDER BY starts_at
`
	joinedQuery := `
	SELECT o.id, o.host_id, o.title, o.destination, o.meet_label, o.meet_lat,
			o.meet_lng, o.starts_at, o.max_size, o.host_seats, o.cost_per_seat_cents,
			o.difficulty, o.pace, o.notes, o.status, o.created_at, o.updated_at
		FROM outings o 
		JOIN join_requests jr ON jr.outing_id = o.id
		WHERE jr.hiker_id = $1 AND jr.status = 'accepted'
		ORDER BY o.starts_at
`
	myOutings := &outing.MyOutings{
		Hosting: []outing.Outing{},
		Joined:  []outing.Outing{},
	}

	rowsHosting, err := s.pool.Query(ctx, hostingQuery, hikerID)
	if err != nil {
		return nil, apperr.Database("failed to list hosting outings", "my_outings: select hosting outings failed", err)
	}
	defer rowsHosting.Close()
	for rowsHosting.Next() {
		hostingOuting, scanErr := scanOuting(rowsHosting)
		if scanErr != nil {
			return nil, apperr.Database("failed to list hosting outings", "my_outings: scan hosting outings failed", scanErr)
		}
		myOutings.Hosting = append(myOutings.Hosting, *hostingOuting)
	}
	if err = rowsHosting.Err(); err != nil {
		return nil, apperr.Database("failed to list hosting outings", "my_outings: iterate hosting outings failed", err)
	}

	rowsJoined, err := s.pool.Query(ctx, joinedQuery, hikerID)
	if err != nil {
		return nil, apperr.Database("failed to list joined outings", "my_outings: select joined outings failed", err)
	}
	defer rowsJoined.Close()

	for rowsJoined.Next() {
		joinedOuting, scanErr := scanOuting(rowsJoined)
		if scanErr != nil {
			return nil, apperr.Database("failed to list joined outings", "my_outings: scan joined outings failed", scanErr)
		}
		myOutings.Joined = append(myOutings.Joined, *joinedOuting)
	}
	if err = rowsJoined.Err(); err != nil {
		return nil, apperr.Database("failed to list joined outings", "my_outings: iterate joined outings failed", err)
	}

	return myOutings, nil
}

func scanOuting(row pgx.Row) (*outing.Outing, error) {
	o := &outing.Outing{}
	err := row.Scan(
		&o.ID,
		&o.HostID,
		&o.Title,
		&o.Destination,
		&o.MeetLabel,
		&o.MeetLat,
		&o.MeetLng,
		&o.StartsAt,
		&o.MaxSize,
		&o.HostSeats,
		&o.CostPerSeatCents,
		&o.Difficulty,
		&o.Pace,
		&o.Notes,
		&o.Status,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	return o, err
}

func scanJoinRequest(row pgx.Row) (*outing.JoinRequest, error) {
	r := &outing.JoinRequest{}

	err := row.Scan(
		&r.ID,
		&r.OutingID,
		&r.HikerID,
		&r.Status,
		&r.Role,
		&r.SeatsOffered,
		&r.Guests,
		&r.Note,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	return r, err
}
