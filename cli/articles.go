package cli

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/muesli/termenv"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/spy16/connote/article"
)

var (
	termEnvFunc = termenv.TemplateFuncs(termenv.ColorProfile())

	//go:embed articles.tpl
	articlesTplStr string
	articlesTpl    = template.Must(template.New("articles_tpl").Funcs(termEnvFunc).Parse(articlesTplStr))
)

func cmdShowArticle(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show [id-or-name]",
		Short:   "Show an article by id or name",
		Args:    cobra.MaximumNArgs(1),
		Aliases: []string{"display", "view", "get"},
	}

	var showMeta bool
	var wrap int
	var style string
	cmd.Flags().BoolVarP(&showMeta, "meta", "m", false, "Show metadata as well")
	cmd.Flags().StringVarP(&style, "style", "s", "dracula", "Output style for markdown")
	cmd.Flags().IntVarP(&wrap, "wrap", "w", 100, "Word wrap after")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		args = inferIdentifier(args)

		b, err := articles.Get(ctx, args[0])
		if err != nil {
			logrus.Fatalf("failed to get article with id/name '%s': %v", args[0], err)
		}

		if useJSON {
			jsonOut(b)
		} else {
			sc := glamour.DefaultStyles[style]
			if sc == nil {
				sc = &glamour.DarkStyleConfig
			}

			r, _ := glamour.NewTermRenderer(
				glamour.WithStyles(*sc),
				glamour.WithEmoji(),
				glamour.WithWordWrap(wrap),
			)
			md, _ := r.Render(b.Content)
			if showMeta {
				b.Content = md
				tplOut(articlesTpl, map[string]interface{}{
					"event":   "article_show",
					"article": b,
				})
			} else {
				fmt.Println(md)
			}
		}
	}
	return cmd
}

func cmdEditArticle(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "write [id-or-name]",
		Args:    cobra.MaximumNArgs(1),
		Short:   "Create/Edit an article",
		Aliases: []string{"edit", "note"},
	}

	var tags []string
	flags := cmd.Flags()
	flags.StringSliceVarP(&tags, "tags", "t", nil, "Tags to categorize")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		args = inferIdentifier(args)

		var ar article.Article
		existingAr, err := articles.Get(ctx, args[0])
		if err != nil {
			if errors.Is(err, article.ErrNotFound) {
				ar.Name = strings.TrimSpace(args[0])
				ar.Content = "# " + ar.Name

				newAr, err := articles.Create(ctx, ar)
				if err != nil {
					color.Red("failed to create new article: %v", err)
					os.Exit(1)
				}
				ar = *newAr
			} else {
				color.Red("failed to fetch article: %v", err)
				os.Exit(1)
			}
		} else {
			ar = *existingAr
		}
		ar.Tags = append(ar.Tags, tags...)

		edited, err := externalEditor(ar.ToMarkdown())
		if err != nil {
			logrus.Fatalf("failed to open editor: %v", err)
		} else if err := ar.FromMD(strings.NewReader(edited)); err != nil {
			logrus.Fatalf("failed to parse updated content: %v", err)
		} else if err := articles.Update(ctx, ar); err != nil {
			logrus.Fatalf("failed to save updated content: %v", err)
		}

		if useJSON {
			jsonOut(ar)
		} else {
			tplOut(articlesTpl, map[string]interface{}{
				"event":   "article_saved",
				"article": ar,
			})
		}
	}

	return cmd
}

func cmdListArticles(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search <name-pattern>",
		Args:    cobra.MaximumNArgs(1),
		Short:   "List all articles",
		Aliases: []string{"li", "ls", "query", "q", "list", "articles"},
	}

	var q article.Query
	var tableView bool
	var after, before string
	flags := cmd.Flags()
	flags.BoolVarP(&tableView, "table", "T", false, "Use table view")
	flags.StringVarP(&after, "after", "a", "", "Created After")
	flags.StringVarP(&before, "before", "b", "", "Created Before")
	flags.StringSliceVarP(&q.HavingTags, "tags", "t", nil, "Tags to filter by")
	flags.BoolVarP(&q.MatchAllTags, "all", "A", true, "Should match all tags")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			q.NameLike = strings.TrimSpace(args[0])
		}

		after = strings.TrimSpace(after)
		before = strings.TrimSpace(before)
		if after != "" {
			afterT, err := article.ParseTime(after)
			if err != nil {
				logrus.Fatalf("invalid time-string '%s': %v", after, err)
			}

			if after == before {
				q.CreatedBetween = [2]time.Time{
					time.Date(afterT.Year(), afterT.Month(), afterT.Day(), 0, 0, 0, 0, afterT.Location()),
					time.Date(afterT.Year(), afterT.Month(), afterT.Day(), 23, 59, 59, 0, afterT.Location()),
				}
			} else {
				q.CreatedBetween = [2]time.Time{afterT, {}}
			}
		}

		articlesList, err := articles.List(ctx, q)
		if err != nil {
			logrus.Fatalf("failed to list articles: %v", err)
		}

		if useJSON {
			jsonOut(articlesList)
		} else if tableView {
			data := make([][]string, len(articlesList), len(articlesList))
			for i, ar := range articlesList {
				data[i] = []string{fmt.Sprintf("%d", ar.ID), ar.Name, strings.Join(ar.Tags, ",")}
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Name", "Tags"})
			table.SetBorder(true)
			table.AppendBulk(data)
			table.Render()
		} else {
			tplOut(articlesTpl, map[string]interface{}{
				"event":    "article_list",
				"articles": articlesList,
			})
		}
	}
	return cmd
}

func cmdDeleteArticle(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm [id]",
		Short:   "Delete an article",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"remove", "delete"},
		Run: func(cmd *cobra.Command, args []string) {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				logrus.Fatalf("'%s' is not valid article id", args[0])
			}

			b := article.Article{ID: int(id)}
			err = articles.Delete(ctx, b.ID)
			if err != nil {
				logrus.Fatalf("failed to delete article %d: %v", id, err)
			}

			if useJSON {
				jsonOut(b)
			} else {
				tplOut(articlesTpl, map[string]interface{}{
					"event":   "article_deleted",
					"article": b,
				})
			}
		},
	}
	return cmd
}

func cmdLoadFrom(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "from <dir-or-file>",
		Short:   "Load articles from markdown files in given directory or file",
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
			logrus.Infof("loading '%s'", path)
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			ar := article.Article{}
			if err := ar.FromMD(f); err != nil {
				return err
			}
			ar.Name = strings.TrimSuffix(f.Name(), ".md")
			ar.Tags = append(ar.Tags, tags...)

			_, err = articles.Create(ctx, ar)
			return err
		}

		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				logrus.Fatalf("path '%s' does not exist", path)
			}
			logrus.Fatalf("unexpected error: %v", err)
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
				logrus.Fatalf("failed to walk '%s': %v", path, walkErr)
			}
		} else {
			if err := addOne(path); err != nil {
				logrus.Fatalf("failed to load from '%s': %v", path, err)
			}
		}
	}
	return cmd
}

func inferIdentifier(args []string) []string {
	const expander = "@"

	if len(args) == 0 {
		return []string{makeDayID(time.Now())}
	} else if strings.HasPrefix(args[0], expander) {
		spec := strings.TrimPrefix(args[0], expander)

		t, err := article.ParseTime(spec)
		if err == nil {
			return []string{makeDayID(t)}
		}
	}

	return args
}

func makeDayID(t time.Time) string {
	return fmt.Sprintf("day:%d-%s-%d", t.Day(), t.Month().String()[0:3], t.Year())
}

func tplOut(tpl *template.Template, v interface{}) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, v); err != nil {
		logrus.Errorf("failed to execute template: %v", err)
	}
	fmt.Println(strings.TrimSpace(buf.String()))
}
