package domain

type CommentService interface {
	CreateComment(parentID *int64, author, content string) (*Comment, error)
	GetThread(parentID *int64, limit, offset int, sort string) ([]*Comment, error)
	DeleteThread(id int64) error
	SearchComment(query string, limit, offset int) ([]*Comment, error)
}
