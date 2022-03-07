package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/spy16/connote/note"
)

var (
	root = &cobra.Command{
		Use:               "connote <command> [flags]",
		Short:             "📝 Console based note taking tool.",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
)

// Execute executes the CLI parser and invokes the command.
func Execute(ctx context.Context, version string) {
	var logLevel, profile string
	flags := root.PersistentFlags()
	flags.StringVarP(&profile, "profile", "p", "work", "Profile to load and use")
	flags.StringVarP(&logLevel, "log-level", "l", "warn", "Log level to use")
	flags.StringP("output", "o", "pretty", "Output format (json, yaml, markdown & pretty)")

	root.Version = version
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		lvl, err := logrus.ParseLevel(logLevel)
		if err != nil {
			return err
		}
		logrus.SetLevel(lvl)

		home, err := homedir.Dir()
		if err != nil {
			return err
		}
		configDir := filepath.Join(home, ".connote")
		if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
			return err
		}
		notesDir := filepath.Join(configDir, profile)

		notes, err = note.Open(notesDir, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// setup all commands
	root.AddCommand(
		cmdShowNote(ctx),
		cmdReindex(ctx),
		cmdEditNote(ctx),
		cmdSearch(ctx),
		cmdLoadNotes(ctx),
		cmdRemoveNote(ctx),
	)

	_ = root.Execute()
}

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
