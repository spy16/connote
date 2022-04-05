package note

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	emptyClone  = "cloned an empty repository"
	readFailed  = "Could not read from remote repository"
	nonEmptyDir = "exists and is not an empty directory"
)

var (
	errEmptyClone     = errors.New("cloned repository is empty")
	errInvalidRemote  = errors.New("invalid remote url or is not accessible")
	errNonEmptyTarget = errors.New("non-empty target dir for clone")
)

func gitClone(dir, gitRemote string) (err error) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	git := exec.Command("git", "clone", gitRemote, ".")
	git.Dir = dir

	out, err := git.CombinedOutput()
	if err != nil {
		if bytes.Contains(out, []byte(readFailed)) {
			return errInvalidRemote
		} else if bytes.Contains(out, []byte(nonEmptyDir)) {
			return errNonEmptyTarget
		}
		return err
	} else if bytes.Contains(out, []byte(emptyClone)) {
		return errEmptyClone
	}

	return nil
}

func gitSync(remoteSpec, dir string) error {
	remoteSpec = strings.TrimSpace(remoteSpec)

	if remoteSpec == "" {
		// dir must already be a git repo.
		if err := ensureGitRepo(dir); err != nil {
			return fmt.Errorf("dir is not a git repository")
		}
	} else {
		err := gitClone(dir, remoteSpec)
		if err != nil && !errors.Is(err, errEmptyClone) {
			return err
		}
	}

	if err := gitPull(true, dir); err != nil {
		return err
	}

	return gitPush(dir)
}

func ensureGitRepo(dir string) error {
	return nil
}

func gitPull(rebase bool, dir string) error {
	git := exec.Command("git", "pull", "origin", "master", "--rebase")
	git.Dir = dir

	out, err := git.CombinedOutput()
	if err != nil {
		if bytes.Contains(out, []byte(readFailed)) {
			return errInvalidRemote
		} else if bytes.Contains(out, []byte(nonEmptyDir)) {
			return errNonEmptyTarget
		}
		return err
	} else if bytes.Contains(out, []byte(emptyClone)) {
		return errEmptyClone
	}
	return nil
}

func gitPush(dir string) error {
	return nil
}
