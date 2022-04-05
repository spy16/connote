package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/spy16/connote/pkg/config"
	"github.com/spy16/connote/pkg/note"
)

var (
	Commit    = "N/A"
	Version   = "N/A"
	BuildTime = "N/A"

	rootCmd = &cobra.Command{
		Use:               "connote <command> [flags]",
		Short:             "📝 Console based note taking tool.",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
)

func main() {
	rand.Seed(time.Now().UnixNano())

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	runCLI(ctx)
}

func runCLI(ctx context.Context) {
	flags := rootCmd.PersistentFlags()
	flags.StringP("output", "o", "pretty", "Output format (json, yaml, markdown & pretty)")
	flags.StringP("config", "c", "", "override configuration file")

	var logLevel, profile string
	flags.StringVarP(&profile, "profile", "p", "work", "Profile to load and use")
	flags.StringVarP(&logLevel, "log-level", "l", "warn", "Log level to use")

	rootCmd.Version = fmt.Sprintf("Version %s (commit %s built on %s)", Version, Commit, BuildTime)
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		err := config.CobraPreRunHook("", "connote")(cmd, args)
		if err != nil {
			exitErr("❗%v", err)
		}

		lvl, err := logrus.ParseLevel(logLevel)
		if err != nil {
			lvl = logrus.InfoLevel
		}
		logrus.SetLevel(lvl)

		profileDir, err := getProfileDir(profile)
		if err != nil {
			exitErr("❗️failed to infer profile dir: %v", err)
		}

		notes, err = note.Open(profile, profileDir, "", nil)
		if err != nil {
			exitErr("❗️ failed to open: %v", err)
		}
	}

	// setup all commands
	rootCmd.AddCommand(
		// Notes management.
		cmdShowNote(),
		cmdReindex(),
		cmdEditNote(),
		cmdSearch(),
		cmdLoadNotes(),
		cmdRemoveNote(),

		// Profile management.
		cmdInfo(),
		cmdInitProfile(),
	)

	_ = rootCmd.Execute()
}

func getProfileDir(profile string) (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".connote")
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return "", err
	}
	notesDir := filepath.Join(configDir, profile)

	return notesDir, nil
}
