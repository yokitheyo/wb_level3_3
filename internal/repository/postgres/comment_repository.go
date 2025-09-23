package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wb-go/wbf/dbpg"
	"github.com/yokitheyo/wb_level3_3/internal/domain"
)

type CommentRepository struct {
	db *dbpg.DB
}

func NewCommentRepository(db *dbpg.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Save(ctx context.Context, c *domain.Comment) error {
	query := `INSERT INTO comments (parent_id, author, content) VALUES ($1, $2, $3) RETURNING id, created_at`
	row := r.db.Master.QueryRowContext(ctx, query, c.ParentID, c.Author, c.Content)
	var createdAt time.Time
	if err := row.Scan(&c.ID, &createdAt); err != nil {
		return err
	}
	c.CreatedAt = createdAt
	return nil
}

// scanner может быть *sql.Row или *sql.Rows — у обоих есть метод Scan(...)
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanCommentRowGeneric(s scanner) (*domain.Comment, error) {
	var (
		id         int64
		parentRaw  interface{}
		authorRaw  interface{}
		contentRaw interface{}
		created    time.Time
		updatedRaw interface{}
		deletedRaw interface{}
	)

	if err := s.Scan(&id, &parentRaw, &authorRaw, &contentRaw, &created, &updatedRaw, &deletedRaw); err != nil {
		return nil, err
	}

	// parent -> *int64
	var p *int64
	if parentRaw != nil {
		switch v := parentRaw.(type) {
		case int64:
			tmp := v
			p = &tmp
		case int32:
			tmp := int64(v)
			p = &tmp
		case int:
			tmp := int64(v)
			p = &tmp
		case []byte:
			if s := string(v); s != "" {
				if parsed, err := strconv.ParseInt(s, 10, 64); err == nil {
					tmp := parsed
					p = &tmp
				}
			}
		case string:
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				tmp := parsed
				p = &tmp
			}
		}
	}

	// author -> string
	var author string
	if authorRaw != nil {
		switch v := authorRaw.(type) {
		case string:
			author = v
		case []byte:
			author = string(v)
		default:
			author = fmt.Sprint(v)
		}
	}

	// content -> string
	var content string
	if contentRaw != nil {
		switch v := contentRaw.(type) {
		case string:
			content = v
		case []byte:
			content = string(v)
		default:
			content = fmt.Sprint(v)
		}
	}

	// updated -> *time.Time (nullable)
	var up *time.Time
	if updatedRaw != nil {
		switch v := updatedRaw.(type) {
		case time.Time:
			tmp := v
			up = &tmp
		case []byte:
			if s := string(v); s != "" {
				// пробуем несколько форматов
				if parsed, err := time.Parse(time.RFC3339Nano, s); err == nil {
					tmp := parsed
					up = &tmp
				} else if parsed, err := time.Parse(time.RFC3339, s); err == nil {
					tmp := parsed
					up = &tmp
				}
			}
		case string:
			if parsed, err := time.Parse(time.RFC3339Nano, v); err == nil {
				tmp := parsed
				up = &tmp
			} else if parsed, err := time.Parse(time.RFC3339, v); err == nil {
				tmp := parsed
				up = &tmp
			}
		}
	}

	// deleted -> bool
	var deleted bool
	if deletedRaw != nil {
		switch v := deletedRaw.(type) {
		case bool:
			deleted = v
		case int64:
			deleted = v != 0
		case []byte:
			s := string(v)
			deleted = s == "t" || s == "true" || s == "1"
		case string:
			deleted = v == "t" || v == "true" || v == "1"
		default:
			// fallback
			deleted = false
		}
	}

	return &domain.Comment{
		ID:        id,
		ParentID:  p,
		Author:    author,
		Content:   content,
		CreatedAt: created,
		UpdatedAt: up,
		Deleted:   deleted,
		Children:  make([]*domain.Comment, 0),
	}, nil
}

func (r *CommentRepository) FindByID(ctx context.Context, id int64) (*domain.Comment, error) {
	query := `SELECT id, parent_id, author, content, created_at, updated_at, deleted FROM comments WHERE id = $1`
	row := r.db.Master.QueryRowContext(ctx, query, id)
	return scanCommentRowGeneric(row)
}

func (r *CommentRepository) FindChildren(ctx context.Context, parentID *int64, limit, offset int, sort string) ([]*domain.Comment, error) {
	if parentID == nil {
		sortClause := "created_at DESC"
		if strings.ToLower(sort) == "asc" {
			sortClause = "created_at ASC"
		}
		query := fmt.Sprintf(`SELECT id, parent_id, author, content, created_at, updated_at, deleted
			FROM comments WHERE parent_id IS NULL ORDER BY %s LIMIT $1 OFFSET $2`, sortClause)

		rows, err := r.db.QueryContext(ctx, query, limit, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []*domain.Comment
		for rows.Next() {
			c, err := scanCommentRowGeneric(rows)
			if err != nil {
				return nil, err
			}
			out = append(out, c)
		}
		return out, nil
	}

	query := `
	WITH RECURSIVE subtree AS (
	  SELECT id, parent_id, author, content, created_at, updated_at, deleted FROM comments WHERE id = $1
	  UNION ALL
	  SELECT c.id, c.parent_id, c.author, c.content, c.created_at, c.updated_at, c.deleted
	  FROM comments c JOIN subtree s ON c.parent_id = s.id
	)
	SELECT id, parent_id, author, content, created_at, updated_at, deleted FROM subtree ORDER BY created_at ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, *parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := map[int64]*domain.Comment{}
	order := make([]int64, 0)
	for rows.Next() {
		c, err := scanCommentRowGeneric(rows)
		if err != nil {
			return nil, err
		}
		byID[c.ID] = c
		order = append(order, c.ID)
	}

	for _, id := range order {
		c := byID[id]
		if c.ParentID != nil {
			if p, ok := byID[*c.ParentID]; ok {
				p.Children = append(p.Children, c)
			}
		}
	}

	if root, ok := byID[*parentID]; ok {
		return []*domain.Comment{root}, nil
	}

	out := make([]*domain.Comment, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out, nil
}

func (r *CommentRepository) Delete(ctx context.Context, id int64) error {
	query := `
	WITH RECURSIVE to_delete AS (
	  SELECT id FROM comments WHERE id = $1
	  UNION ALL
	  SELECT c.id FROM comments c JOIN to_delete td ON c.parent_id = td.id
	)
	DELETE FROM comments WHERE id IN (SELECT id FROM to_delete);
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *CommentRepository) Search(ctx context.Context, q string, limit, offset int) ([]*domain.Comment, error) {
	query := `
	SELECT id, parent_id, author, content, created_at, updated_at, deleted
	FROM comments
	WHERE content_tsv @@ plainto_tsquery('russian', $1)
	ORDER BY ts_rank(content_tsv, plainto_tsquery('russian', $1)) DESC
	LIMIT $2 OFFSET $3;
	`
	rows, err := r.db.QueryContext(ctx, query, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*domain.Comment, 0)
	for rows.Next() {
		c, err := scanCommentRowGeneric(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}
