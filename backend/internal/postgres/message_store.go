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

// GetConversationView loads one conversation with everything the chat page needs:
// the row itself, the outing's title and start time when kind is outing (NULL for
// DMs), and the participants with their names — host and accepted members for an
// outing, both parties for a DM. The participant query is viewer-independent on
// purpose; the caller (message.Service) enforces membership with IsMember, and the
// client picks "the other party" for a DM by excluding itself. NotFound when the
// conversation doesn't exist.
func (s *MessageStore) GetConversationView(ctx context.Context, convID uuid.UUID) (*message.ConversationView, error) {
	q1 := ` 
	SELECT c.id, c.kind, o.title, o.starts_at, c.outing_id, o.host_id, c.dm_a, c.dm_b, c.dm_initiator, c.dm_status, c.dm_declined_by, c.created_at 
	FROM conversations c LEFT JOIN outings o ON o.id = c.outing_id 
	WHERE c.id = $1
`
	q2 := `
		SELECT h.id, h.name 
		FROM conversations c 
		JOIN outings o ON o.id = c.outing_id
		JOIN hikers h ON h.id = o.host_id
		WHERE c.id = $1
		UNION 
		SELECT h.id, h.name
		FROM conversations c
		JOIN join_requests jr ON jr.outing_id = c.outing_id
		JOIN hikers h ON h.id = jr.hiker_id
		WHERE c.id = $1 and jr.status = 'accepted'
		UNION 
		SELECT h.id, h.name
		FROM conversations c 
		JOIN hikers h ON h.id IN (c.dm_a, c.dm_b)
		WHERE c.id = $1
`
	cv := &message.ConversationView{}
	if err := s.pool.QueryRow(ctx, q1, convID).Scan(
		&cv.ID,
		&cv.Kind,
		&cv.OutingTitle,
		&cv.OutingStartsAt,
		&cv.OutingID,
		&cv.OutingHostID,
		&cv.DmA,
		&cv.DmB,
		&cv.DmInitiator,
		&cv.DmStatus,
		&cv.DmDeclinedBy,
		&cv.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("conversation not found", "no row for id")
		}
		return nil, apperr.Database("failed to get conversation", "failed to scan conversation view", err)
	}
	participants := []message.Participant{}
	rows, err := s.pool.Query(ctx, q2, convID)
	if err != nil {
		return nil, apperr.Database("failed to get conversation", "failed to get participants for conversation", err)
	}
	defer rows.Close()
	for rows.Next() {
		p := message.Participant{}
		if verr := rows.Scan(&p.HikerID, &p.Name); verr != nil {
			return nil, apperr.Database("failed to get conversation", "failed to scan participant row", verr)
		}
		participants = append(participants, p)
	}
	if err = rows.Err(); err != nil {
		return nil, apperr.Database("failed to get conversation", "failed to iterate participants", err)
	}
	cv.Participants = participants

	return cv, nil

}

// ListConversations loads all conversations for a hiker
func (s *MessageStore) ListConversations(ctx context.Context, hikerID uuid.UUID) ([]*message.ConversationSummary, error) {
	q := `
	SELECT * FROM(
		SELECT c.id, c.kind, o.title, c.dm_status, c.dm_initiator, c.dm_declined_by,
			lm.created_at AS last_message_at, COALESCE(LEFT(lm.body, 80), ''), c.created_at AS created_at
		FROM conversations c
		JOIN outings o ON o.id = c.outing_id
		LEFT JOIN LATERAL (
			SELECT body, created_at FROM messages m
			WHERE m.conversation_id = c.id ORDER BY seq DESC LIMIT 1
		) lm ON true
		WHERE c.kind = 'outing'
			AND (o.host_id = $1 OR EXISTS (SELECT 1 FROM join_requests jr
				WHERE jr.outing_id = o.id AND jr.hiker_id = $1 AND jr.status = 'accepted'))
	
		UNION ALL
	
		SELECT c.id, c.kind, h.name, c.dm_status, c.dm_initiator, c.dm_declined_by,
		   lm.created_at, COALESCE(LEFT(lm.body, 80), ''), c.created_at
		FROM conversations c
		JOIN hikers h ON h.id = CASE WHEN c.dm_a = $1 THEN c.dm_b ELSE c.dm_a END
		LEFT JOIN LATERAL (
		SELECT body, created_at FROM messages m
		WHERE m.conversation_id = c.id ORDER BY seq DESC LIMIT 1
		) lm ON true
		WHERE c.kind = 'dm' AND $1 IN (c.dm_a, c.dm_b)
	) x ORDER BY COALESCE(x.last_message_at, x.created_at) DESC
`
	convSums := []*message.ConversationSummary{}
	rows, err := s.pool.Query(ctx, q, hikerID)
	if err != nil {
		return nil, apperr.Database("failed to get conversations summary", "scanning summary failed", err)
	}
	defer rows.Close()
	for rows.Next() {
		cs := message.ConversationSummary{}
		err = rows.Scan(&cs.ID, &cs.Kind, &cs.Title, &cs.DmStatus, &cs.DmInitiator, &cs.DmDeclinedBy, &cs.LastMessageAt, &cs.LastPreview, &cs.CreatedAt)
		if err != nil {
			return nil, apperr.Database("failed to get conversations summary", "failed to scan conversations", err)
		}
		convSums = append(convSums, &cs)
	}
	if err = rows.Err(); err != nil {
		return nil, apperr.Database("failed to get comversations summary", "failed to iterate  conversations", err)
	}
	return convSums, nil
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

// CanDM reports whether initiator may open a DM with other: both are host or
// accepted members of a common outing, or initiator hosts an outing other has a
// pending request on. The pending door is one-way.
func (s *MessageStore) CanDM(ctx context.Context, initiator, other uuid.UUID) (bool, error) {
	q := `SELECT EXISTS (
    SELECT 1
    FROM outings o
    WHERE
        -- door 1: both hikers belong to this outing (host or accepted)
        (
            (o.host_id = $1 OR EXISTS (
                SELECT 1 FROM join_requests jr
                WHERE jr.outing_id = o.id AND jr.hiker_id = $1 AND jr.status = 'accepted'))
            AND
            (o.host_id = $2 OR EXISTS (
                SELECT 1 FROM join_requests jr
                WHERE jr.outing_id = o.id AND jr.hiker_id = $2 AND jr.status = 'accepted'))
        )
        OR
        -- door 2: $1 hosts this outing and $2 has a pending request on it (one-way)
        (
            o.host_id = $1 AND EXISTS (
                SELECT 1 FROM join_requests jr
                WHERE jr.outing_id = o.id AND jr.hiker_id = $2 AND jr.status = 'requested')
        )
)`
	var can bool
	if err := s.pool.QueryRow(ctx, q, initiator, other).Scan(&can); err != nil {
		return false, apperr.Database("starting DM failed", "failed to scan if user can dm.", err)
	}
	return can, nil
}

// GetOrCreateDM inserts the DM for the sorted pair (lo < hi) as pending with the
// given initiator, or returns the existing one. The bool reports whether this call
// created it; the UNIQUE (dm_a, dm_b) constraint serializes concurrent creates.
func (s *MessageStore) GetOrCreateDM(ctx context.Context, lo, hi, initiator uuid.UUID) (*message.Conversation, bool, error) {
	q := `
		INSERT INTO conversations (kind, dm_a, dm_b, dm_initiator, dm_status)
		VALUES ('dm', $1, $2, $3, 'pending')
		ON CONFLICT (dm_a, dm_b) DO NOTHING
		RETURNING id, kind, outing_id, dm_a, dm_b, dm_initiator, dm_status, dm_declined_by, created_at
		`
	conv, err := scanConversation(s.pool.QueryRow(ctx, q, lo, hi, initiator))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			q = `
				SELECT id, kind, outing_id, dm_a, dm_b, dm_initiator, dm_status, dm_declined_by, created_at
				FROM conversations WHERE dm_a = $1 AND dm_b = $2
				`
			conv, err = scanConversation(s.pool.QueryRow(ctx, q, lo, hi))
			if err != nil {
				return nil, false, apperr.Database("failed to get dm", "failed to scan existing outing", err)
			}
			return conv, false, nil
		}
		return nil, false, apperr.Database("failed to create dm", "failed to insert dm", err)
	}
	return conv, true, nil

}

// UpdateDMStatus sets dm_status and dm_declined_by on a DM conversation;
// NotFound when no such conversation exists.
func (s *MessageStore) UpdateDMStatus(ctx context.Context, convID uuid.UUID, status message.DMStatus, declinedBy *uuid.UUID) error {
	q := `
		UPDATE conversations
		SET dm_status = $2, dm_declined_by = $3
		WHERE id = $1 AND kind = 'dm'
`
	ct, err := s.pool.Exec(ctx, q, convID, status, declinedBy)
	if err != nil {
		return apperr.Database("failed to update dm status", "failed to exec command for updating dm status", err)
	}
	if ct.RowsAffected() == 0 {
		return apperr.NotFound("dm is not found", "row not found for dm")
	}
	return nil
}

// HikerName returns the hiker's display name, used to label notifications
// ("X wants to message you"). NotFound when no such hiker exists.
func (s *MessageStore) HikerName(ctx context.Context, hikerID uuid.UUID) (string, error) {
	q := `SELECT name FROM hikers where id = $1`
	var hikerName string
	if err := s.pool.QueryRow(ctx, q, hikerID).Scan(&hikerName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", apperr.NotFound("hiker not found", "row not found")
		}
		return "", apperr.Database("failed to get hiker name", "scanning hiker name failed", err)
	}
	return hikerName, nil

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
