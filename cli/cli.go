package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/spy16/connote/article"
)

var (
	useJSON  bool
	articles *article.API

	root = &cobra.Command{
		Use:   "connote <command> [flags]",
		Short: "Console based note taking tool.",
	}
)

// Execute executes the CLI parser and invokes the command.
func Execute(ctx context.Context, version string) {
	var logLevel, profile string
	flags := root.PersistentFlags()
	flags.StringVarP(&profile, "profile", "p", "work", "Profile to load and use")
	flags.StringVarP(&logLevel, "log-level", "l", "warn", "Log level to use")
	flags.BoolVarP(&useJSON, "json", "j", false, "Use JSON output instead of pretty")

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
		profileDB := filepath.Join(configDir, profile+".db")
		logrus.Infof("using profile db '%s'", profileDB)

		db, err := sql.Open("sqlite3", profileDB)
		if err != nil {
			return err
		}

		articles, err = article.New(db)
		if err != nil {
			return err
		}
		return nil
	}

	// setup all commands
	root.AddCommand(
		cmdEditArticle(ctx),
		cmdListArticles(ctx),
		cmdShowArticle(ctx),
		cmdDeleteArticle(ctx),
		cmdLoadFrom(ctx),
	)

	_ = root.Execute()
}

func jsonOut(v interface{}) {
	_ = json.NewEncoder(os.Stdout).Encode(v)
}

func externalEditor(d []byte) (string, error) {
	f, err := os.CreateTemp(os.TempDir(), "wt-article*.md")
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.Write(d); err != nil {
		return "", err
	}
	_ = f.Sync()

	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err = cmd.Start(); err != nil {
		return "", err
	}
	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	_, _ = f.Seek(0, 0)
	buf, err := ioutil.ReadAll(f)
	return string(buf), err
}
