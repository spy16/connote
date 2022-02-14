package note

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Store implementation should provide efficient storage for notes.
type Store interface {
	Search(ctx context.Context, q Query) ([]Note, error)
	Get(ctx context.Context, byID int, byName string) (*Note, error)
	Put(ctx context.Context, note Note) (int, error)
	Del(ctx context.Context, id int) error
}

// API provides functions to manage notes in the given directory.
type API struct{ Store Store }

// Query represents filtering options for articles.
type Query struct {
	Tags           []string     `json:"tags"`
	NameRegex      string       `json:"name_like"`
	CreatedBetween [2]time.Time `json:"created_between"`
}

// Search returns all notes matching the query.
func (api *API) Search(ctx context.Context, q Query) ([]Note, error) {
	res, err := api.Store.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	var notes []Note
	for _, n := range res {
		if n.IsMatch(q) {
			notes = append(notes, n)
		}
	}

	return notes, nil
}

// Get returns the note by its numeric ID or name.
func (api *API) Get(ctx context.Context, idOrName string) (*Note, error) {
	idOrName = strings.TrimSpace(idOrName)

	if !nameExp.MatchString(idOrName) {
		if !idExp.MatchString(idOrName) {
			return nil, fmt.Errorf("'%s' is not a valid name or id", idOrName)
		}

		id, _ := strconv.ParseInt(idOrName, 10, 64)
		return api.Store.Get(ctx, int(id), "")
	}

	return api.Store.Get(ctx, 0, idOrName)
}

// Upsert creates/updates a note to the collection.
func (api *API) Upsert(ctx context.Context, note Note, createOnly bool) (*Note, error) {
	if err := note.Validate(); err != nil {
		return nil, err
	}
	note.CreatedAt = time.Now().UTC()
	note.UpdatedAt = note.CreatedAt

	if createOnly {
		if _, err := api.Store.Get(ctx, 0, note.Name); err == nil {
			return nil, fmt.Errorf("conflict: note with name '%s' already exists", note.Name)
		}
	}

	id, err := api.Store.Put(ctx, note)
	if err != nil {
		return nil, err
	}
	note.ID = id

	return &note, nil
}

// Delete deletes a note by its id.
func (api *API) Delete(ctx context.Context, id int) error {
	return api.Store.Del(ctx, id)
}
