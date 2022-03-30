package profile

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	emptyClone  = "cloned an empty repository"
	readFailed  = "Could not read from remote repository"
	nonEmptyDir = "exists and is not an empty directory"
)

var namePattern = regexp.MustCompile("^[A-Za-z][A-Za-z0-9-_]+$")

// Profile represents an isolated collection of notes, etc.
type Profile struct {
	LogFn     LogFn  `json:"-"`
	Dir       string `json:"dir" yaml:"dir"`
	Name      string `json:"name" yaml:"name"`
	GitRemote string `json:"git_remote" yaml:"git_remote"`

	notesIdx map[string]noteIdxEntry
}

// Init initialises the profile locally by cloning from git-remote.
func (p *Profile) Init() (isEmpty bool, err error) {
	if p.LogFn == nil {
		p.LogFn = func(lvl, format string, args ...interface{}) {
			lvl = strings.ToUpper(lvl)
			log.Printf("[%s] %s", lvl, fmt.Sprintf(format, args...))
		}
	}

	if err := p.Validate(); err != nil {
		return false, err
	}

	if err := os.MkdirAll(p.Dir, os.ModePerm); err != nil {
		return false, err
	}

	git := exec.Command("git", "clone", p.GitRemote, ".")
	git.Dir = p.Dir

	out, err := git.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		if bytes.Contains(out, []byte(readFailed)) {
			return false, fmt.Errorf("origin url is invalid or is not accessible")
		} else if bytes.Contains(out, []byte(nonEmptyDir)) {
			return false, fmt.Errorf("non-empty profile with same name exists")
		}
		return false, err
	}

	isEmpty = bytes.Contains(out, []byte(emptyClone))
	if !isEmpty {
		return false, p.loadIdx()
	}
	return true, nil
}

// Sync tries to synchronise local and remote copies of the profile.
func (p *Profile) Sync() error {
	// git pull origin master --rebase
	// git push origin master
	return nil
}

// Validate sets defaults if not overridden and validates the profile.
func (p *Profile) Validate() error {
	p.Name = strings.TrimSpace(p.Name)
	p.GitRemote = strings.TrimSpace(p.GitRemote)
	p.Dir = strings.TrimSpace(p.Dir)
	if p.Dir == "" {
		p.Dir = filepath.Join(os.Getenv("HOME"), ".connote", p.Name)
	}

	if p.Name == "" {
		return errors.New("name cannot be empty")
	} else if !namePattern.MatchString(p.Name) {
		return fmt.Errorf("invalid name, must match '%s'", namePattern)
	}

	if p.GitRemote == "" {
		return errors.New("git_remote must be specified")
	}
	return nil
}
