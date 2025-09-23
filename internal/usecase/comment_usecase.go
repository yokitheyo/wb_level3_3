package usecase

import (
	"context"
	"errors"
	"sync"

	"github.com/wb-go/wbf/zlog"

	"github.com/yokitheyo/wb_level3_3/internal/domain"
)

type CommentUsecase struct {
	repo domain.CommentRepository
}

func NewCommentUsecase(repo domain.CommentRepository) *CommentUsecase {
	return &CommentUsecase{repo: repo}
}

func (u *CommentUsecase) CreateComment(ctx context.Context, parentID *int64, author, content string) (*domain.Comment, error) {
	if author == "" {
		return nil, errors.New("author required")
	}
	if content == "" {
		return nil, errors.New("content required")
	}

	c := &domain.Comment{
		ParentID: parentID,
		Author:   author,
		Content:  content,
	}

	if err := u.repo.Save(ctx, c); err != nil {
		zlog.Logger.Error().Err(err).Msg("usecase: Save comment failed")
		return nil, err
	}

	zlog.Logger.Info().Msgf("comment created id=%d parent=%v", c.ID, c.ParentID)
	return c, nil
}

func (u *CommentUsecase) GetThread(ctx context.Context, parentID *int64, limit, offset int, sort string) ([]*domain.Comment, error) {
	roots, err := u.repo.FindChildren(ctx, parentID, limit, offset, sort)
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("usecase: FindChildren failed")
		return nil, err
	}

	if parentID == nil && len(roots) > 0 {
		out := make([]*domain.Comment, 0, len(roots))
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, root := range roots {
			wg.Add(1)
			go func(r *domain.Comment) {
				defer wg.Done()
				sub, err := u.repo.FindChildren(ctx, &r.ID, 0, 0, "asc")
				if err != nil {
					zlog.Logger.Error().Err(err).Msgf("usecase: FindChildren for root %d failed", r.ID)
					mu.Lock()
					out = append(out, r)
					mu.Unlock()
					return
				}
				if len(sub) > 0 {
					mu.Lock()
					out = append(out, sub[0])
					mu.Unlock()
					return
				}
				// fallback
				mu.Lock()
				out = append(out, r)
				mu.Unlock()
			}(root)
		}

		wg.Wait()
		return out, nil
	}

	return roots, nil
}

func (u *CommentUsecase) DeleteThread(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	if err := u.repo.Delete(ctx, id); err != nil {
		zlog.Logger.Error().Err(err).Msgf("usecase: Delete failed id=%d", id)
		return err
	}
	zlog.Logger.Info().Msgf("comment deleted id=%d", id)
	return nil
}

func (u *CommentUsecase) SearchComment(ctx context.Context, q string, limit, offset int) ([]*domain.Comment, error) {
	if q == "" {
		return nil, errors.New("empty query")
	}
	res, err := u.repo.Search(ctx, q, limit, offset)
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("usecase: Search failed")
		return nil, err
	}
	return res, nil
}
