package search

import (
	"context"

	"github.com/yokitheyo/wb_level3_3/internal/domain"
)

type FullTextSearcher interface {
	SearchComments(ctx context.Context, query string, limit, offset int) ([]*domain.Comment, error)
}

type PostgresFullText struct {
	repo domain.CommentRepository
}

// NewPostgresFullText создаёт адаптер полнотекстового поиска
func NewPostgresFullText(repo domain.CommentRepository) *PostgresFullText {
	return &PostgresFullText{repo: repo}
}

func (f *PostgresFullText) SearchComments(ctx context.Context, query string, limit, offset int) ([]*domain.Comment, error) {
	return f.repo.Search(ctx, query, limit, offset)
}
