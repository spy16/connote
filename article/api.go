package article

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
)

var ErrNotFound = errors.New("not found")

// New returns an instance of API for managing articles.
func New(db *sql.DB) (*API, error) {
	api := &API{db: db}
	if err := api.init(); err != nil {
		return nil, err
	}
	return api, nil
}

// Query represents filtering options for articles.
type Query struct {
	NameLike       string       `json:"name_like"`
	HavingTags     []string     `json:"having_tags"`
	MatchAllTags   bool         `json:"match_all_tags"`
	CreatedBetween [2]time.Time `json:"created_between"`
}

// API provides functions for managing articles.
type API struct{ db *sql.DB }

// Get returns an article by id or name. Returns ErrNotFound if no article is
// found.
func (api *API) Get(ctx context.Context, idOrName string) (*Article, error) {
	var ar *Article

	txErr := withinTx(ctx, api.db, func(tx *sql.Tx) error {
		id, _ := strconv.ParseInt(idOrName, 10, 64)

		var err error
		ar, err = getArticle(ctx, tx, int(id), idOrName)
		if err != nil {
			return err
		}

		tags, err := getArticleTags(ctx, tx, ar.ID)
		if err != nil {
			return err
		}
		ar.Tags = tags

		return nil
	})

	return ar, txErr
}

// Create a new work article in db and return the identifier.
func (api *API) Create(ctx context.Context, ar Article) (*Article, error) {
	if err := ar.Validate(); err != nil {
		return nil, err
	}

	insertArticle := func(tx *sql.Tx) error {
		q := sq.Insert(articlesTable).
			Columns("name", "content", "created_at", "updated_at").
			Values(ar.Name, ar.Content, ar.CreatedAt, ar.UpdatedAt)

		res, err := q.RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		ar.ID = int(id)

		return nil
	}

	insertTags := func(tx *sql.Tx) error {
		if len(ar.Tags) == 0 {
			return nil
		}

		q := sq.Insert(articleTagsTable).
			Columns("article_id", "tag").
			Suffix("ON CONFLICT DO NOTHING")

		for _, tag := range ar.Tags {
			q = q.Values(ar.ID, tag)
		}
		_, err := q.RunWith(tx).ExecContext(ctx)
		return err
	}

	return &ar, withinTx(ctx, api.db, insertArticle, insertTags)
}

// List returns all articles matching the given query.
func (api *API) List(ctx context.Context, q Query) ([]Article, error) {
	q.NameLike = strings.TrimSpace(q.NameLike)

	sqlQ := sq.Select("article_id", "name", "content", "created_at", "updated_at").
		Distinct().From(articlesTable).OrderBy("created_at desc")

	if q.NameLike != "" {
		sqlQ = sqlQ.Where(sq.Like{"name": "%" + q.NameLike + "%"})
	}

	if len(q.HavingTags) > 0 {
		sqlQ = sqlQ.
			Join("article_tags USING (article_id)").
			Where(sq.Eq{"tag": q.HavingTags}).
			GroupBy("article_id")

		if q.MatchAllTags {
			sqlQ = sqlQ.Having("count(*) >= ?", len(q.HavingTags))
		} else {
			sqlQ = sqlQ.Having("count(*) >= 1")
		}
	}

	if createdAfter := q.CreatedBetween[0]; !createdAfter.IsZero() {
		sqlQ = sqlQ.Where(sq.GtOrEq{"created_at": createdAfter})
	}
	if createdBefore := q.CreatedBetween[1]; !createdBefore.IsZero() {
		sqlQ = sqlQ.Where(sq.LtOrEq{"created_at": createdBefore})
	}

	var articles []Article
	txErr := withinTx(ctx, api.db, func(tx *sql.Tx) error {
		rows, err := sqlQ.RunWith(api.db).QueryContext(ctx)
		if err != nil {
			return err
		}

		for rows.Next() {
			var ar Article

			if err := rows.Scan(&ar.ID, &ar.Name, &ar.Content, &ar.CreatedAt, &ar.UpdatedAt); err != nil {
				return err
			}

			ar.Tags, err = getArticleTags(ctx, tx, ar.ID)
			if err != nil {
				return err
			}

			articles = append(articles, ar)
		}

		return nil
	})

	return articles, txErr
}

// Delete removes an article by id from the db.
func (api *API) Delete(ctx context.Context, id int) error {
	dropTags := func(tx *sql.Tx) error {
		q := sq.Delete(articleTagsTable).Where(sq.Eq{"article_id": id})
		_, err := q.RunWith(tx).ExecContext(ctx)
		return err
	}

	dropArticle := func(tx *sql.Tx) error {
		q := sq.Delete(articlesTable).Where(sq.Eq{"article_id": id})
		_, err := q.RunWith(tx).ExecContext(ctx)
		return err
	}

	return withinTx(ctx, api.db, dropTags, dropArticle)
}

// Update updates the article by overriding fields in given partial
// object with non zero-values.
func (api *API) Update(ctx context.Context, ar Article) error {
	if err := ar.Validate(); err != nil {
		return err
	}

	updateArticle := func(tx *sql.Tx) error {
		q := sq.Update(articlesTable).Where(sq.Eq{"article_id": ar.ID})
		q = q.Set("content", strings.TrimSpace(ar.Content))
		q = q.Set("updated_at", time.Now())
		_, err := q.RunWith(tx).ExecContext(ctx)
		return err
	}

	removeAllTags := func(tx *sql.Tx) error {
		q := sq.Delete(articleTagsTable).Where(sq.Eq{"article_id": ar.ID})
		_, err := q.RunWith(tx).ExecContext(ctx)
		return err
	}

	addTags := func(tx *sql.Tx) error {
		q := sq.Insert(articleTagsTable).
			Columns("article_id", "tag").
			Suffix("ON CONFLICT DO NOTHING")

		count := 0
		for _, tag := range ar.Tags {
			count++
			q = q.Values(ar.ID, strings.TrimPrefix(tag, "+"))
		}
		if count == 0 {
			return nil
		}

		_, err := q.RunWith(tx).ExecContext(ctx)
		return err
	}

	return withinTx(ctx, api.db, updateArticle, removeAllTags, addTags)
}

func (api *API) init() error {
	if _, err := api.db.Exec(schema); err != nil {
		return err
	}
	return nil
}

func splitTag(tag string) (k, v string) {
	pair := strings.SplitN(tag, ":", 2)
	key := pair[0]
	if len(pair) == 1 {
		return key, ""
	}
	return key, pair[1]
}
