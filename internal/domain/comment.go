package domain

import "time"

type Comment struct {
	ID        int64      `json:"id"`
	ParentID  *int64     `json:"parent_id"`
	Content   string     `json:"content"`
	AuthorID  string     `json:"author"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	Deleted   bool       `json:"deleted"`
}
