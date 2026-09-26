package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/muster/internal/message"
	"github.com/diagnosis/muster/internal/outing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MessageStore is the Postgres implementation of message.Storage.
type MessageStore struct {
	pool *pgxpool.Pool
}

// NewMessageStore returns a MessageStore over pool.
func NewMessageStore(pool *pgxpool.Pool) *MessageStore {
	return &MessageStore{pool: pool}
}

// GetConversation loads one conversation by id; NotFound when absent.
func (s *MessageStore) GetConversation(ctx context.Context, id uuid.UUID) (*message.Conversation, error) {
	q := `
		SELECT id, kind, outing_id, dm_a, dm_b, dm_initiator, dm_status, dm_declined_by, created_at 
		FROM  conversations WHERE id = $1
`
	c, err := scanConversation(s.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("conversation not found", "no row for id")
		}
		return nil, apperr.Database("failed to get conversation", "scanning conversation failed", err)
	}
	return c, nil
}

// IsMember reports whether hikerID may read and post in conversationID:
// the outing's host, a hiker with an accepted join request, or a DM party.
func (s *MessageStore) IsMember(ctx context.Context, conversationID, hikerID uuid.UUID) (bool, error) {
	q := `SELECT EXISTS(
			SELECT 1
			FROM conversations c
			LEFT JOIN outings o ON o.id = c.outing_id
			WHERE c.id = $1
			AND (
			    o.host_id = $2
			    OR EXISTS(
			        SELECT 1 FROM join_requests jr
			        WHERE jr.outing_id = c.outing_id
                  	AND jr.hiker_id = $2
                  	AND jr.status = 'accepted'
			    )
			    OR $2 IN (c.dm_a, c.dm_b)
			)
	)`
	var member bool
	if err := s.pool.QueryRow(ctx, q, conversationID, hikerID).Scan(&member); err != nil {
		return false, apperr.Database("error occurred", "failed to check membership status of hiker", err)
	}
	return member, nil

}

// InsertMessage inserts m with created_at = now and fills ID, Seq and
// CreatedAt from RETURNING.
func (s *MessageStore) InsertMessage(ctx context.Context, m *message.Message, now time.Time) error {
	q := `
		INSERT INTO messages (conversation_id, hiker_id, body, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, seq, created_at
`
	row := s.pool.QueryRow(ctx, q, m.ConversationID, m.HikerID, m.Body, now)
	if err := row.Scan(&m.ID, &m.Seq, &m.CreatedAt); err != nil {
		return apperr.Database("failed to create message", "inserting message failed", err)
	}
	return nil

}

// MemberIDs returns every hiker who should receive a poke for the conversation:
// host and accepted requesters for an outing, both parties for a DM.
func (s *MessageStore) MemberIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	q := `
		SELECT o.host_id FROM conversations c JOIN outings o ON o.id = c.outing_id WHERE c.id = $1
		UNION
		SELECT jr.hiker_id FROM conversations c JOIN join_requests jr ON jr.outing_id = c.outing_id
    	WHERE c.id = $1 AND jr.status = 'accepted'
		UNION
		SELECT unnest(ARRAY[c.dm_a, c.dm_b]) FROM conversations c WHERE c.id = $1 AND c.kind = 'dm'
`
	rows, err := s.pool.Query(ctx, q, conversationID)
	if err != nil {
		return nil, apperr.Database("failed to list members", "select memberIDs failed", err)
	}
	defer rows.Close()
	var memberIDs []uuid.UUID
	for rows.Next() {
		var memberID uuid.UUID
		if err = rows.Scan(&memberID); err != nil {
			return nil, apperr.Database("failed to list members", "scan memberIDS failed", err)
		}
		memberIDs = append(memberIDs, memberID)
	}
	if err = rows.Err(); err != nil {
		return nil, apperr.Database("failed to list members", "iterate memberIDS failed", err)
	}
	return memberIDs, nil
}

// OutingStatus returns the outing's status; NotFound when absent.
func (s *MessageStore) OutingStatus(ctx context.Context, outingID uuid.UUID) (outing.Status, error) {
	q := `SELECT status from outings WHERE id = $1`
	var status outing.Status
	if err := s.pool.QueryRow(ctx, q, outingID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", apperr.NotFound("outing not found", "outing not found")
		}
		return "", apperr.Database("failed to fetch outing status", "failed to scan outing status", err)
	}
	return status, nil

}

// CountMessagesSince counts messages by hikerID in conversationID created
// strictly after since (the rate-limit window).
func (s *MessageStore) CountMessagesSince(ctx context.Context, conversationID, hikerID uuid.UUID, since time.Time) (int, error) {
	q := `SELECT COUNT(*) 
	      FROM messages 
	      WHERE conversation_id = $1 
	        AND hiker_id = $2 
	        AND created_at > $3;`

	var count int
	err := s.pool.QueryRow(ctx, q, conversationID, hikerID, since).Scan(&count)
	if err != nil {
		return 0, apperr.Database("failed to count messages", "failed to count messages", err)
	}
	return count, nil

}

// ListMessages returns every message in the conversation ordered by seq (D6).
func (s *MessageStore) ListMessages(ctx context.Context, conversationID uuid.UUID) ([]*message.Message, error) {
	q := `SELECT id, conversation_id, hiker_id, seq, body, created_at 
		FROM messages 
		WHERE conversation_id = $1 ORDER BY seq
`
	rows, err := s.pool.Query(ctx, q, conversationID)
	if err != nil {
		return nil, apperr.Database("failed to list messages", "failed to select messages", err)
	}
	defer rows.Close()
	ms := []*message.Message{}
	for rows.Next() {
		m, nerr := scanMessage(rows)
		if nerr != nil {
			return nil, apperr.Database("failed to list messages", "scan messages failed", nerr)
		}
		ms = append(ms, m)
	}
	if err = rows.Err(); err != nil {
		return nil, apperr.Database("failed to list messages", "scan messages failed", err)
	}
	return ms, nil
}

// GetMessage loads one message by id; NotFound when absent.
func (s *MessageStore) GetMessage(ctx context.Context, messageID uuid.UUID) (*message.Message, error) {
	q := `SELECT id, conversation_id, hiker_id, seq, body, created_at 
		FROM messages 
		WHERE id = $1
`
	m, err := scanMessage(s.pool.QueryRow(ctx, q, messageID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("message not found", "message not found")
		}
		return nil, apperr.Database("failed to get message", "scanning message failed", err)
	}
	return m, nil
}

// DeleteMessage hard-deletes a message; deleting a missing id is a no-op.
func (s *MessageStore) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	q := `DELETE FROM messages WHERE id = $1`
	_, err := s.pool.Exec(ctx, q, messageID)
	if err != nil {
		return apperr.Database("failed to delete", "failed to exec delete", err)
	}
	return nil
}

// OutingHost returns the outing's host id; NotFound when absent.
func (s *MessageStore) OutingHost(ctx context.Context, outingID uuid.UUID) (uuid.UUID, error) {
	q := `SELECT host_id from outings WHERE id = $1`
	var hostID uuid.UUID
	if err := s.pool.QueryRow(ctx, q, outingID).Scan(&hostID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, apperr.NotFound("outing not found", "outing not found")
		}
		return uuid.Nil, apperr.Database("failed to fetch outing host id", "failed to scan host id", err)
	}
	return hostID, nil
}

var _ message.Storage = (*MessageStore)(nil)

// helper

func scanConversation(row pgx.Row) (*message.Conversation, error) {
	c := message.Conversation{}
	if err := row.Scan(
		&c.ID,
		&c.Kind,
		&c.OutingID,
		&c.DmA,
		&c.DmB,
		&c.DmInitiator,
		&c.DmStatus,
		&c.DmDeclinedBy,
		&c.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &c, nil

}

func scanMessage(row pgx.Row) (*message.Message, error) {
	m := message.Message{}
	if err := row.Scan(
		&m.ID,
		&m.ConversationID,
		&m.HikerID,
		&m.Seq,
		&m.Body,
		&m.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &m, nil
}
