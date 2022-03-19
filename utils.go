package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spy16/connote/note"
)

func externalEditor(d []byte) ([]byte, error) {
	f, err := os.CreateTemp(os.TempDir(), "wt-article*.md")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()

	if _, err := f.Write(d); err != nil {
		return nil, err
	}
	_ = f.Sync()

	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		editor = "vi"
	}

	editorPath, err := exec.LookPath(editor)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(editorPath, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err = cmd.Run(); err != nil {
		return nil, err
	}

	_, _ = f.Seek(0, 0)
	return ioutil.ReadAll(f)
}

func fzfSearch(notes []note.Note) (string, error) {
	items := make([]string, len(notes), len(notes))
	for i := range notes {
		items[i] = notes[i].Name
	}

	p, err := exec.LookPath("fzf")
	if err != nil {
		return "", errors.New("fzf not found")
	}

	cmd := exec.Command(p)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	cmd.Stderr = os.Stderr

	buf := bytes.Buffer{}
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return "", err
	}

	sel := strings.TrimSpace(buf.String())
	return sel, nil
}

func makeDayID(t time.Time) string {
	return fmt.Sprintf("day:%d-%s-%d", t.Day(), t.Month().String()[0:3], t.Year())
}
