package usecase

import (
	"context"
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

func (uc *commentUsecase) CreateComment(ctx context.Context, parentID *int64, author, content string) (*domain.Comment, error) {
	if content == "" {
		return nil, errors.New("content cannot be empty")
	}
	c := &domain.Comment{
		ParentID:  parentID,
		Author:    author,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := uc.repo.Save(ctx, c); err != nil {
		return nil, err
	}

	return c, nil
}

func (uc *commentUsecase) GetThread(ctx context.Context, parentID *int64, limit, offset int, sort string) ([]*domain.Comment, error) {
	return uc.repo.FindChildren(ctx, parentID, limit, offset, sort)
}

func (uc *commentUsecase) DeleteThread(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *commentUsecase) SearchComment(ctx context.Context, query string, limit, offset int) ([]*domain.Comment, error) {
	return uc.repo.Search(ctx, query, limit, offset)
}
