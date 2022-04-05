package note

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const idxName = "notes_idx.json"

var (
	namePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-_]+$`)

	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// Open returns a new API instance for given directory. If directory is not found
// it will be created automatically.
func Open(profileName, dir, gitRemote string, logFn LogFn) (*API, error) {
	dir = strings.TrimSpace(dir)
	gitRemote = strings.TrimSpace(gitRemote)
	profileName = strings.TrimSpace(profileName)

	if !namePattern.MatchString(profileName) {
		return nil, fmt.Errorf("invalid profile name, must match '%s'", namePattern)
	}

	if logFn == nil {
		logFn = func(lvl, format string, args ...interface{}) {
			lvl = strings.ToUpper(lvl)
			log.Printf("[%s] %s", lvl, fmt.Sprintf(format, args...))
		}
	}

	info, err := os.Stat(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if os.IsNotExist(err) && gitRemote == "" {
		return nil, fmt.Errorf("profile directory non-existent")
	} else if info != nil && !info.IsDir() {
		return nil, fmt.Errorf("profile path is not a directory")
	}

	api := &API{dir: dir, log: logFn, profile: profileName}
	if err := api.initDir(gitRemote); err != nil {
		return nil, err
	}
	return api, api.loadIdx()
}

// API provides functions to manage notes in a given directory.
type API struct {
	dir     string
	log     LogFn
	idx     map[string]indexNode
	profile string
}

// Search finds names of all notes that match the given query.
func (api *API) Search(q Query, loadNote bool) ([]Note, error) {
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
	for name, node := range api.idx {
		if nameRE != nil && !nameRE.MatchString(name) {
			continue
		} else if !q.isMatch(node) {
			continue
		}

		if loadNote {
			n, err := api.Get(name)
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
func (api *API) Get(name string) (*Note, error) {
	name = strings.TrimSpace(name)
	if _, found := api.idx[name]; !found {
		return nil, fmt.Errorf("%w: note with name '%s'", ErrNotFound, name)
	}

	d, err := os.ReadFile(api.getPath(name))
	if err != nil {
		return nil, err
	}

	return Parse(d)
}

// Put saves a new note. If a note with same name exists and this is not
// an update, returns ErrConflict.
func (api *API) Put(note Note, createOnly bool) (*Note, error) {
	if err := note.Validate(); err != nil {
		return nil, err
	}
	note.CreatedAt = time.Now()
	note.UpdatedAt = time.Now()

	if _, found := api.idx[note.Name]; found && createOnly {
		return nil, fmt.Errorf("%w: note with name '%s' already exists", ErrConflict, note.Name)
	}

	path := api.getPath(note.Name)
	if err := ioutil.WriteFile(path, note.ToMarkdown(), 0644); err != nil {
		return nil, err
	}

	api.idx[note.Name] = indexNode{
		Tags:      arrToSet(note.Tags),
		CreatedAt: note.CreatedAt.Unix(),
	}
	return &note, api.syncIdx()
}

// Del deletes a note with given name. If not found, returns ErrNotFound.
func (api *API) Del(name string) error {
	path := api.getPath(name)

	if _, found := api.idx[name]; !found {
		return fmt.Errorf("%w: note with name '%s'", ErrNotFound, name)
	}

	delete(api.idx, name)
	if err := api.syncIdx(); err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// Index walks the directory and re-builds the index.
func (api *API) Index() error {
	api.idx = map[string]indexNode{}

	walkErr := filepath.Walk(api.dir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() && api.dir != path {
			api.log("debug", "skipping dir '%s'", path)
			return filepath.SkipDir
		} else if err != nil {
			return err
		} else if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		api.log("debug", "reading file '%s'", path)

		d, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		n, err := Parse(d)
		if err != nil {
			return err
		}

		api.idx[n.Name] = indexNode{
			Tags:      arrToSet(n.Tags),
			CreatedAt: n.CreatedAt.Unix(),
		}
		return nil
	})
	if walkErr != nil {
		api.idx = nil
		return walkErr
	}

	return api.syncIdx()
}

// Stats returns statistics of this note storage.
func (api *API) Stats() (profile, dir string, count int) {
	return api.profile, api.dir, len(api.idx)
}

func (api *API) loadIdx() error {
	idxPath := filepath.Join(api.dir, idxName)
	if fi, err := os.Stat(idxPath); err != nil {
		if os.IsNotExist(err) {
			return api.Index()
		}
		return err
	} else if fi.IsDir() {
		return fmt.Errorf("'%s' is a directory, not an index file", idxPath)
	}

	d, err := os.ReadFile(idxPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(d, &api.idx)
}

func (api *API) syncIdx() error {
	idxPath := filepath.Join(api.dir, idxName)
	d, err := json.Marshal(api.idx)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(idxPath, d, 0644)
}

func (api *API) initDir(gitRemote string) error {
	if err := os.MkdirAll(api.dir, os.ModePerm); err != nil {
		return err
	}

	if gitRemote != "" {
		if err := gitClone(api.dir, gitRemote); err != nil && !errors.Is(err, errEmptyClone) {
			return err
		}
	}
	return nil
}

func (api *API) getPath(name string) string {
	name = strings.TrimSpace(name)
	return filepath.Join(api.dir, fmt.Sprintf("%s.md", name))
}

// Query represents filtering options for articles.
type Query struct {
	NameLike     string   `json:"name_like"`
	IncludeTags  []string `json:"include_tags"`
	ExcludeTags  []string `json:"exclude_tags"`
	CreatedRange [2]int64 `json:"created_range"`
}

type indexNode struct {
	Tags      map[string]struct{} `json:"tags"`
	CreatedAt int64               `json:"created_at"`
}

func (q Query) isMatch(node indexNode) bool {
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
