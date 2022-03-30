package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const idxName = "notes_idx.json"

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// Search finds names of all notes that match the given query.
func (p *Profile) Search(q Query, loadNote bool) ([]Note, error) {
	var nameRE *regexp.Regexp
	q.NameLike = strings.TrimSpace(q.NameLike)
	if q.NameLike != "" {
		np, err := regexp.Compile(q.NameLike)
		if err != nil {
			return nil, fmt.Errorf("invalid search regex '%s': %v", q.NameLike, err)
		}
		nameRE = np
	}

	var res []Note
	for name, node := range p.notesIdx {
		if nameRE != nil && !nameRE.MatchString(name) {
			continue
		} else if !q.isMatch(node) {
			continue
		}

		if loadNote {
			n, err := p.Get(name)
			if err != nil {
				return nil, err
			}
			res = append(res, *n)
		} else {
			res = append(res, Note{
				Name:      name,
				Tags:      setToArray(node.Tags),
				CreatedAt: time.Unix(node.CreatedAt, 0),
			})
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.After(res[j].CreatedAt)
	})
	return res, nil
}

// Get returns a note by its unique name.
func (p *Profile) Get(name string) (*Note, error) {
	name = strings.TrimSpace(name)
	if _, found := p.notesIdx[name]; !found {
		return nil, fmt.Errorf("%w: note with name '%s'", ErrNotFound, name)
	}

	d, err := os.ReadFile(p.getPath(name))
	if err != nil {
		return nil, err
	}

	return Parse(d)
}

// Put saves a new note. If a note with same name exists and this is not
// an update, returns ErrConflict.
func (p *Profile) Put(note Note, createOnly bool) (*Note, error) {
	if err := note.Validate(); err != nil {
		return nil, err
	}
	note.CreatedAt = time.Now()
	note.UpdatedAt = time.Now()

	if _, found := p.notesIdx[note.Name]; found && createOnly {
		return nil, fmt.Errorf("%w: note with name '%s' already exists", ErrConflict, note.Name)
	}

	path := p.getPath(note.Name)
	if err := ioutil.WriteFile(path, note.ToMarkdown(), 0644); err != nil {
		return nil, err
	}

	p.notesIdx[note.Name] = noteIdxEntry{
		Tags:      arrToSet(note.Tags),
		CreatedAt: note.CreatedAt.Unix(),
	}
	return &note, p.syncIdx()
}

// Del deletes a note with given name. If not found, returns ErrNotFound.
func (p *Profile) Del(name string) error {
	path := p.getPath(name)

	if _, found := p.notesIdx[name]; !found {
		return fmt.Errorf("%w: note with name '%s'", ErrNotFound, name)
	}

	delete(p.notesIdx, name)
	if err := p.syncIdx(); err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// Index walks the directory and re-builds the index.
func (p *Profile) Index() error {
	p.notesIdx = map[string]noteIdxEntry{}

	walkErr := filepath.Walk(p.Dir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() && p.Dir != path {
			p.LogFn("debug", "skipping dir '%s'", path)
			return filepath.SkipDir
		} else if err != nil {
			return err
		} else if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		p.LogFn("debug", "reading file '%s'", path)

		d, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		n, err := Parse(d)
		if err != nil {
			return err
		}

		p.notesIdx[n.Name] = noteIdxEntry{
			Tags:      arrToSet(n.Tags),
			CreatedAt: n.CreatedAt.Unix(),
		}
		return nil
	})
	if walkErr != nil {
		p.notesIdx = nil
		return walkErr
	}

	return p.syncIdx()
}

// Stats returns statistics of this note storage.
func (p *Profile) Stats() (profile, dir string, count int) {
	return p.Name, p.Dir, len(p.notesIdx)
}

func (p *Profile) loadIdx() error {
	idxPath := filepath.Join(p.Dir, idxName)
	if fi, err := os.Stat(idxPath); err != nil {
		if os.IsNotExist(err) {
			return p.Index()
		}
		return err
	} else if fi.IsDir() {
		return fmt.Errorf("'%s' is a directory, not an index file", idxPath)
	}

	d, err := os.ReadFile(idxPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(d, &p.notesIdx)
}

func (p *Profile) syncIdx() error {
	idxPath := filepath.Join(p.Dir, idxName)
	d, err := json.Marshal(p.notesIdx)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(idxPath, d, 0644)
}

func (p *Profile) getPath(name string) string {
	name = strings.TrimSpace(name)
	return filepath.Join(p.Dir, fmt.Sprintf("%s.md", name))
}

// Query represents filtering options for articles.
type Query struct {
	NameLike     string   `json:"name_like"`
	IncludeTags  []string `json:"include_tags"`
	ExcludeTags  []string `json:"exclude_tags"`
	CreatedRange [2]int64 `json:"created_range"`
}

type noteIdxEntry struct {
	Tags      map[string]struct{} `json:"tags"`
	CreatedAt int64               `json:"created_at"`
}

func (q Query) isMatch(node noteIdxEntry) bool {
	for _, tag := range q.IncludeTags {
		if _, found := node.Tags[tag]; !found {
			return false
		}
	}

	for _, tag := range q.ExcludeTags {
		if _, found := node.Tags[tag]; found {
			return false
		}
	}

	after, before := q.CreatedRange[0], q.CreatedRange[1]
	if before == 0 {
		before = time.Now().Unix()
	}
	return node.CreatedAt >= after && node.CreatedAt <= before
}

func setToArray(set map[string]struct{}) []string {
	var arr []string
	for v := range set {
		arr = append(arr, v)
	}
	return arr
}
