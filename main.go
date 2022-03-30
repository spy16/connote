package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	var logLevel, profile string
	flags := rootCmd.PersistentFlags()
	flags.StringVarP(&profile, "profile", "p", "work", "Profile to load and use")
	flags.StringVarP(&logLevel, "log-level", "l", "warn", "Log level to use")
	flags.StringP("output", "o", "pretty", "Output format (json, yaml, markdown & pretty)")
	flags.StringP("config", "c", "", "override configuration file")

	rootCmd.Version = fmt.Sprintf("Version %s (commit %s built on %s)", Version, Commit, BuildTime)
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		err := config.CobraPreRunHook("", "connote")(cmd, args)
		if err != nil {
			return err
		}

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

		notes, err = note.Open(profile, notesDir, true, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// setup all commands
	rootCmd.AddCommand(
		cmdShowNote(),
		cmdReindex(),
		cmdEditNote(),
		cmdSearch(),
		cmdLoadNotes(),
		cmdRemoveNote(),
		cmdInfo(),
	)

	_ = rootCmd.Execute()
}

func cmdInfo() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show statistics and profile information",
		Run: func(cmd *cobra.Command, args []string) {
			profile, dir, count := notes.Stats()
			m := map[string]interface{}{
				"count":     count,
				"profile":   profile,
				"directory": dir,
			}
			writeOut(cmd, m, func(_ string) string {
				var s = "-------------------------------\n"
				s += fmt.Sprintf("👤 Profile  : %s\n", profile)
				s += fmt.Sprintf("📂 Location : %s\n", dir)
				s += fmt.Sprintf("❕ Notes    : %d\n", count)
				s += "-------------------------------\n"
				return strings.TrimSpace(s)
			})
		},
	}
}
