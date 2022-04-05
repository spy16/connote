package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/spy16/connote/pkg/note"
)

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

func cmdInitProfile() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init <profile> <git-remote>",
		Short:   "Initialise a new profile",
		Args:    cobra.ExactArgs(2),
		Aliases: []string{"start", "new", "initialise", "setup"},
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		profile, remote := strings.TrimSpace(args[0]), strings.TrimSpace(args[1])

		profileDir, err := getProfileDir(profile)
		if err != nil {
			exitErr("❗️ failed to infer profile dir: %v", err)
		}

		_, err = note.Open(profile, profileDir, remote, nil)
		if err != nil {
			exitErr("❗️ failed to init profile dir: %v", err)
		}

		exitOk("Profile '%s' initialised from '%s'", profile, remote)
	}
	return cmd
}

func cmdSyncProfile() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sync",
		Short:   "Sync local and remote versions of the profile",
		Args:    cobra.ExactArgs(2),
		Aliases: []string{"start", "new", "initialise", "setup"},
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		/*
			1. git pull origin master --rebase
			2. git push origin master
		*/
	}
	return cmd
}
