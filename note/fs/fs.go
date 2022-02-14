package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/spy16/connote/note"
)

const idxFileName = "idx_notes.json"

// Store implements note.Store using file-system.
type Store struct {
	Dir string `json:"-"`
	idx struct {
		CurID int               `json:"cur_id"`
		Notes map[int]indexNode `json:"notes"`
		dirty bool
	}
}

func (st *Store) Search(ctx context.Context, q note.Query) ([]note.Note, error) {
	// TODO implement me
	panic("implement me")
}

func (st *Store) Get(ctx context.Context, byID int, byName string) (*note.Note, error) {
	// TODO implement me
	panic("implement me")
}

func (st *Store) Put(ctx context.Context, note note.Note) (int, error) {
	filePath := filepath.Join(st.Dir, fmt.Sprintf("%s.md", note.Name))
	if err := ioutil.WriteFile(filePath, note.ToMarkdown(), os.ModePerm); err != nil {
		return 0, err
	}

	node := indexNode{
		ID:        st.idx.CurID + 1,
		Name:      note.Name,
		Tags:      arrToSet(note.Tags),
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
	}

	st.idx.Notes[note.ID] = node
	st.idx.CurID++
	st.idx.dirty = true

	return node.ID, st.syncIndex()
}

func (st *Store) Del(ctx context.Context, id int) error {
	// TODO: delete the md file.

	if _, found := st.idx.Notes[id]; found {
		delete(st.idx.Notes, id)
		st.idx.dirty = true
		return st.syncIndex()
	}

	return nil
}

func (st *Store) search(q note.Query, reverse bool) []indexNode {
	var nodes []indexNode
	for _, n := range st.idx.Notes {
		if n.isMatch(q) {
			nodes = append(nodes, n)
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		if reverse {
			return nodes[i].CreatedAt.After(nodes[j].CreatedAt)
		}
		return nodes[i].CreatedAt.Before(nodes[j].CreatedAt)
	})

	return nodes
}

func (st *Store) syncIndex() error {
	idxFile := filepath.Join(st.Dir, idxFileName)

	f, err := os.OpenFile(idxFile, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(st.idx)
}

func (st *Store) loadIndex() error {
	idxFile := filepath.Join(st.Dir, idxFileName)

	f, err := os.Open(idxFile)
	if err != nil && os.IsNotExist(err) {
		return err
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(&st)
}
