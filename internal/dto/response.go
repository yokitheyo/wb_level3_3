package dto

import "time"

type CommentResponse struct {
	ID        int64              `json:"id"`
	ParentID  *int64             `json:"parent_id,omitempty"`
	Content   string             `json:"content"`
	Author    string             `json:"author"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt *time.Time         `json:"updated_at,omitempty"`
	Deleted   bool               `json:"deleted"`
	Children  []*CommentResponse `json:"children,omitempty"`
}
