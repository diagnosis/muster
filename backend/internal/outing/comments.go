// backend/internal/outing/comments.go
package outing

import (
	"time"

	"github.com/google/uuid"
)

// Comment is one comment on an outing. ParentID nil = top-level; set = a
// reply to that top-level comment (one level deep, enforced in the service).
// DeletedAt marks a soft delete — replies survive, the parent renders as a stub.
type Comment struct {
	ID        uuid.UUID  `json:"id"`
	OutingID  uuid.UUID  `json:"outing_id"`
	HikerID   uuid.UUID  `json:"hiker_id"`
	ParentID  *uuid.UUID `json:"parent_id"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"-"`
}

// CommentLike records that a hiker liked a comment. One per (comment, hiker)
// — the composite PK enforces it; a like is presence, an unlike is deletion.
type CommentLike struct {
	CommentID uuid.UUID `json:"comment_id"`
	HikerID   uuid.UUID `json:"hiker_id"`
	CreatedAt time.Time `json:"created_at"`
}

// CommentView is a comment as the list returns it: the row plus derived
// like data and the author's display name, joined at read time.
type CommentView struct {
	Comment
	AuthorName string `json:"author_name"`
	LikeCount  int    `json:"like_count"`
	LikedByMe  bool   `json:"liked_by_me"`
}
