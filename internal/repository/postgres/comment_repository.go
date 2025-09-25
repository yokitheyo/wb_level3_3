package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/wb-go/wbf/dbpg"
	"github.com/yokitheyo/wb_level3_3/internal/domain"
)

type commentRepository struct {
	db       *dbpg.DB
	strategy retry.Strategy
}

func NewCommentRepository(db *dbpg.DB, strategy retry.Strategy) domain.CommentRepository {
	return &commentRepository{db: db, strategy: strategy}
}

func (r *commentRepository) Save(ctx context.Context, c *domain.Comment) error {
	query := `
    INSERT INTO comments (parent_id, author, content, deleted)
    VALUES ($1, $2, $3, $4)
    RETURNING id, created_at, updated_at
`
	return r.db.Master.QueryRowContext(ctx, query,
		c.ParentID,
		c.Author,
		c.Content,
		c.Deleted,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *commentRepository) FindByID(ctx context.Context, id int64) (*domain.Comment, error) {
	query := `
		SELECT id, parent_id, author, content, created_at, updated_at, deleted
		FROM comments
		WHERE id = $1
	`

	c := &domain.Comment{}
	var parent sql.NullInt64
	var updated sql.NullTime

	row := r.db.Master.QueryRowContext(ctx, query, id)
	err := row.Scan(&c.ID, &parent, &c.Author, &c.Content, &c.CreatedAt, &updated, &c.Deleted)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		zlog.Logger.Error().Err(err).Msg("FindByID failed")
		return nil, err
	}

	if parent.Valid {
		c.ParentID = &parent.Int64
	}
	if updated.Valid {
		c.UpdatedAt = &updated.Time
	}

	return c, nil
}

func (r *commentRepository) FindChildren(ctx context.Context, parentID *int64, limit, offset int, sort string) ([]*domain.Comment, error) {
	order := "created_at ASC"
	if sort == "desc" {
		order = "created_at DESC"
	}

	query := fmt.Sprintf(`
		SELECT id, parent_id, author, content, created_at, updated_at, deleted
		FROM comments
		WHERE parent_id = $1
		ORDER BY %s
		LIMIT $2 OFFSET $3
	`, order)

	rows, err := r.db.QueryWithRetry(ctx, r.strategy, query, parentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*domain.Comment
	for rows.Next() {
		c := &domain.Comment{}
		if err := rows.Scan(
			&c.ID,
			&c.ParentID,
			&c.Author,
			&c.Content,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.Deleted,
		); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	return comments, nil
}

func (r *commentRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecWithRetry(ctx, r.strategy, `
		UPDATE comments
		SET deleted = true, updated_at = $2
		WHERE id = $1
	`, id, time.Now())
	return err
}

func (r *commentRepository) Search(ctx context.Context, q string, limit, offset int) ([]*domain.Comment, error) {
	query := `
		SELECT id, parent_id, author, content, created_at, updated_at, deleted
		FROM comments
		WHERE content_tsv @@ plainto_tsquery('russian', $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryWithRetry(ctx, r.strategy, query, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*domain.Comment
	for rows.Next() {
		c := &domain.Comment{}
		if err := rows.Scan(
			&c.ID,
			&c.ParentID,
			&c.Author,
			&c.Content,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.Deleted,
		); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	return comments, nil
}
