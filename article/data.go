package article

import (
	"context"
	"database/sql"
	_ "embed"

	sq "github.com/Masterminds/squirrel"
)

const (
	articlesTable    = "articles"
	articleTagsTable = "article_tags"
)

//go:embed schema.sql
var schema string

func getArticle(ctx context.Context, tx *sql.Tx, id int, name string) (*Article, error) {
	q := sq.Select("article_id", "name", "content", "created_at", "updated_at").
		From(articlesTable)

	if id > 0 {
		q = q.Where(sq.Eq{"article_id": id})
	} else {
		q = q.Where(sq.Eq{"name": name})
	}

	var ar Article
	row := q.RunWith(tx).QueryRowContext(ctx)
	if err := row.Scan(&ar.ID, &ar.Name, &ar.Content, &ar.CreatedAt, &ar.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &ar, nil
}

func getArticleTags(ctx context.Context, tx *sql.Tx, id int) ([]string, error) {
	query := sq.Select("tag").From(articleTagsTable).Where(sq.Eq{"article_id": id})

	rows, err := query.RunWith(tx).QueryContext(ctx)
	if err != nil {
		return nil, err
	}

	var res []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		res = append(res, tag)
	}
	return res, rows.Err()
}

func withinTx(ctx context.Context, db *sql.DB, fns ...txnFn) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, fn := range fns {
		if err := fn(tx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

type txnFn func(tx *sql.Tx) error
