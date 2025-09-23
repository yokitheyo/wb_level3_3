package dto

type CreateCommentRequest struct {
	ParentID *int64 `json:"parent_id,omitempty"`
	Author   string `json:"author"`
	Content  string `json:"content"`
}
