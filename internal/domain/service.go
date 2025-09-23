package domain

import "context"

type CommentService interface {
	CreateComment(ctx context.Context, parentID *int64, author, content string) (*Comment, error)
	GetThread(ctx context.Context, parentID *int64, limit, offset int, sort string) ([]*Comment, error)
	DeleteThread(ctx context.Context, id int64) error
	SearchComment(ctx context.Context, query string, limit, offset int) ([]*Comment, error)
}
