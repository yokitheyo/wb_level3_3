package usecase

import (
	"errors"
	"time"

	"github.com/yokitheyo/wb_level3_3/internal/domain"
)

type commentUsecase struct {
	repo domain.CommentRepository
}

func NewCommentUsecase(repo domain.CommentRepository) domain.CommentService {
	return &commentUsecase{repo: repo}
}

func (uc *commentUsecase) CreateComment(parentID *int64, author, content string) (*domain.Comment, error) {
	if content == "" {
		return nil, errors.New("content cannot be empty")
	}
	c := &domain.Comment{
		ParentID:  parentID,
		Author:    author,
		Content:   content,
		CreatedAt: time.Now(),
	}

	err := uc.repo.Save(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (uc *commentUsecase) GetThread(parentID *int64, limit, offset int, sort string) ([]*domain.Comment, error) {
	return uc.repo.FindChildren(parentID, limit, offset, sort)
}

func (uc *commentUsecase) DeleteThread(id int64) error {
	return uc.repo.Delete(id)
}

func (uc *commentUsecase) SearchComment(query string, limit, offset int) ([]*domain.Comment, error) {
	return uc.repo.Search(query, limit, offset)
}
