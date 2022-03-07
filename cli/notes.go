package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/davecgh/go-spew/spew"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/spy16/connote/note"
)

var (
	notes *note.API
)

func cmdEditNote(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "write [name]",
		Args:    cobra.MaximumNArgs(1),
		Short:   "Create/Edit a note",
		Aliases: []string{"edit", "note"},
	}

	var tags []string
	flags := cmd.Flags()
	flags.StringSliceVarP(&tags, "tags", "t", nil, "Tags to categorize")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		args = inferName(args)

		var nt note.Note
		existing, err := notes.Get(args[0])
		if err != nil {
			if errors.Is(err, note.ErrNotFound) {
				nt.Name = strings.TrimSpace(args[0])
				nt.Content = "# " + nt.Name

				newAr, err := notes.Put(nt, true)
				if err != nil {
					exitErr("❗️ failed to create: %v", err)
				}
				nt = *newAr
			} else {
				exitErr("❗️ failed to fet: %v", err)
			}
		} else {
			nt = *existing
		}
		nt.Tags = append(nt.Tags, tags...)

		edited, err := externalEditor(nt.ToMarkdown())
		if err != nil {
			logrus.Fatalf("failed to open editor: %v", err)
		} else if err := nt.FromMD(edited); err != nil {
			logrus.Fatalf("failed to parse updated content: %v", err)
		} else if _, err = notes.Put(nt, false); err != nil {
			logrus.Fatalf("failed to save updated content: %v", err)
		}

		writeOut(cmd, nt, func(_ string) string {
			return fmt.Sprintf("✅ Note '%s' saved", nt.Name)
		})
	}

	return cmd
}

func cmdShowNote(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show [id-or-name]",
		Short:   "Show a note by name",
		Args:    cobra.MaximumNArgs(1),
		Aliases: []string{"display", "view", "get"},
	}

	var wrap int
	var style string
	var fzf, loadNote bool
	cmd.Flags().BoolVar(&fzf, "fzf", false, "Select from fzf")
	cmd.Flags().StringVarP(&style, "style", "s", "dracula", "Output style for markdown")
	cmd.Flags().IntVarP(&wrap, "wrap", "w", 100, "Word wrap after (for Markdown output)")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if fzf {
			all, err := notes.Search(note.Query{}, loadNote)
			if err != nil {
				exitErr("❗️ Failed to list: %v", err)
			}

			sel, err := fzfSearch(all)
			if err != nil {
				exitErr("❗ Failed to search using fzf: %v", err)
			}
			args = []string{sel}
		} else {
			args = inferName(args)
		}

		nt, err := notes.Get(args[0])
		if err != nil {
			exitErr("❗️ %s", err)
		}

		mdFormat := func(_ string) string {
			sc := glamour.DefaultStyles[style]
			if sc == nil {
				sc = &glamour.DarkStyleConfig
			}

			r, _ := glamour.NewTermRenderer(
				glamour.WithStyles(*sc),
				glamour.WithEmoji(),
				glamour.WithWordWrap(wrap),
			)

			md, err := r.Render(nt.Content)
			if err != nil {
				exitErr("❗ render failed: %v", err)
			}
			return md
		}

		writeOut(cmd, nt, mdFormat)
	}
	return cmd
}

func cmdReindex(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:     "reindex",
		Short:   "Force re-index current profile directory",
		Aliases: []string{"ri", "idx"},
		Run: func(cmd *cobra.Command, args []string) {
			err := notes.Index()
			if err != nil {
				exitErr("💣 %s", err)
			}
			exitOk("✅ Successfully re-indexed")
		},
	}
}

func cmdSearch(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search <name-pattern>",
		Args:    cobra.MaximumNArgs(1),
		Short:   "List all notes",
		Aliases: []string{"li", "ls", "query", "q", "list", "notes"},
	}

	var q note.Query
	var after, before string
	var loadFull bool
	flags := cmd.Flags()
	flags.BoolVar(&loadFull, "full", false, "Load note from file instead of partial data from index")
	flags.StringVarP(&after, "after", "a", "", "Created After")
	flags.StringVarP(&before, "before", "b", "", "Created Before")
	flags.StringSliceVarP(&q.IncludeTags, "include", "i", nil, "Include notes with this tag")
	flags.StringSliceVarP(&q.ExcludeTags, "exclude", "e", nil, "Exclude notes with this tag")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			q.NameLike = strings.TrimSpace(args[0])
		}

		after = strings.TrimSpace(after)
		before = strings.TrimSpace(before)
		if after != "" {
			afterT, err := note.ParseTime(after)
			if err != nil {
				exitErr("❓ Sorry, '%s' is not valid time-string: %v", after, err)
			}

			if after == before {
				dayStart := time.Date(afterT.Year(), afterT.Month(), afterT.Day(), 0, 0, 0, 0, afterT.Location())
				dayEnd := time.Date(afterT.Year(), afterT.Month(), afterT.Day(), 23, 59, 59, 0, afterT.Location())
				q.CreatedRange = [2]int64{dayStart.Unix(), dayEnd.Unix()}
			} else {
				q.CreatedRange = [2]int64{afterT.Unix(), time.Now().Unix()}
			}
		}

		notesList, err := notes.Search(q, false)
		if err != nil {
			exitErr("❗️Search failed: %v", err)
		}
		if notesList == nil {
			notesList = []note.Note{}
		}

		writeOut(cmd, notesList, func(format string) string {
			if len(notesList) == 0 {
				return "❕ No notes matched the query."
			}

			res := strings.Builder{}
			table := tablewriter.NewWriter(&res)
			table.SetHeader([]string{"Name", "Tags", "Created On"})
			for _, s := range notesList {
				tags := "-"
				if len(s.Tags) > 0 {
					tags = strings.Join(s.Tags, ", ")
				}
				table.Append([]string{s.Name, tags, s.CreatedAt.Format("2006-01-02")})
			}
			table.Render()

			return strings.TrimSpace(res.String())
		})
	}
	return cmd
}

func cmdLoadNotes(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "from <dir-or-file>",
		Short:   "Load notes from markdown files in given directory or file",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"load"},
	}

	var tags []string
	var recurse bool
	cmd.Flags().StringSliceVarP(&tags, "tag", "t", nil, "Add these tags to loaded articles")
	cmd.Flags().BoolVarP(&recurse, "recursive", "r", false, "Traverse directory recursively")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		path := strings.TrimSpace(args[0])

		addOne := func(path string) error {
			d, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			nt, err := note.Parse(d)
			if err != nil {
				return err
			} else if nt.Name == "" {
				nt.Name = strings.TrimSuffix(filepath.Base(path), ".md")
			}
			nt.Tags = append(nt.Tags, tags...)

			_, err = notes.Put(*nt, true)
			return err
		}

		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				exitErr("❓ Path '%s' does not exist", path)
			}
			exitErr("❓ Unexpected error: %v", err)
		}

		if fi.IsDir() {
			// recursively load files.
			walkErr := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				} else if info.IsDir() {
					if !recurse {
						return filepath.SkipDir
					}
					return err
				} else if !strings.HasSuffix(info.Name(), ".md") {
					return nil
				}

				return addOne(path)
			})
			if walkErr != nil {
				exitErr("❓ Failed to walk '%s': %v", path, walkErr)
			}
		} else {
			if err := addOne(path); err != nil {
				exitErr("❓ Failed to load from '%s': %v", path, err)
			}
		}
	}
	return cmd
}

func cmdRemoveNote(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm <name>",
		Short:   "Delete a note by name",
		Args:    cobra.MaximumNArgs(1),
		Aliases: []string{"del", "remove", "delete"},
	}

	var autoConfirm bool
	cmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Do not ask confirmation")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			all, err := notes.Search(note.Query{}, false)
			if err != nil {
				exitErr("❗️ Failed to list: %v", err)
			}

			sel, err := fzfSearch(all)
			if err != nil {
				exitErr("❗ Failed to search using fzf: %v", err)
			}
			args = []string{sel}
		}
		name := strings.TrimSpace(args[0])

		confirmed := autoConfirm || confirm("⚠️ You are about to delete '%s', continue? [y/N]: ", name)
		if !confirmed {
			exitOk("❕ Aborted deletion.")
		} else {
			if err := notes.Del(name); err != nil {
				exitErr("❗️Deletion failed: %v", err)
			} else {
				exitOk("✅ Note '%s' has been deleted", name)
			}
		}
	}

	return cmd
}

func inferName(args []string) []string {
	const expander = "@"

	if len(args) == 0 {
		return []string{makeDayID(time.Now())}
	} else if strings.HasPrefix(args[0], expander) {
		spec := strings.TrimPrefix(args[0], expander)

		t, err := note.ParseTime(spec)
		if err == nil {
			return []string{makeDayID(t)}
		}
	}

	return args
}

func writeOut(cmd *cobra.Command, v interface{}, customFmt func(format string) string) {
	format, err := cmd.Flags().GetString("output")
	if err != nil {
		format = "pretty"
	}

	switch format {
	case "json":
		_ = json.NewEncoder(os.Stdout).Encode(v)

	case "yaml":
		_ = yaml.NewEncoder(os.Stdout).Encode(v)

	case "spew":
		spew.Fdump(os.Stdout, v)

	default:
		fmt.Println(customFmt(format))
	}
}

func exitErr(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func exitOk(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(0)
}

func confirm(format string, args ...interface{}) bool {
	fmt.Printf(format, args...)

	sc := bufio.NewScanner(os.Stdin)
	if sc.Scan() {
		ans := strings.TrimSpace(strings.ToLower(sc.Text()))
		return ans == "yes" || ans == "y" || ans == "ok"
	}
	return false
}
