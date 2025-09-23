package domain

type CommentRepository interface {
	Save(comment *Comment) error
	FindByID(id int64) (*Comment, error)
	FindChildren(parentID *int64, limit, offset int, sort string) ([]*Comment, error)
	Delete(id int64) error
	Search(query string, limit, offset int) ([]*Comment, error)
}
