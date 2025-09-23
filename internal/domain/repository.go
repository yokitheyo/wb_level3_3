package domain

import "context"

type CommentRepository interface {
	Save(ctx context.Context, comment *Comment) error
	FindByID(ctx context.Context, id int64) (*Comment, error)
	FindChildren(ctx context.Context, parentID *int64, limit, offset int, sort string) ([]*Comment, error)
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, query string, limit, offset int) ([]*Comment, error)
}
