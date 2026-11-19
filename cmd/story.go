package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devrounds/devrounds/internal/story"
	"github.com/spf13/cobra"
)

var (
	storySince  string
	storyUntil  string
	storyAuthor string
	storyAll    bool
	storyBy     string
	storyFormat string
)

func storyFail(msg string) {
	fmt.Fprintln(os.Stderr, "devrounds:", msg)
	os.Exit(2)
}

var storyCmd = &cobra.Command{
	Use:   "story [path]",
	Short: "Work log built from git history",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := projectDir(args)
		if storyBy != "area" && storyBy != "day" {
			storyFail("--by must be area or day")
		}
		if storyFormat != "text" && storyFormat != "markdown" && storyFormat != "standup" {
			storyFail("--format must be text, markdown or standup")
		}

		author, label := storyAuthor, storyAuthor
		if author == "" && !storyAll {
			name, email := story.CurrentUser(dir)
			if email != "" {
				author = email
				label = name
				if label == "" {
					label = "you"
				}
			}
		}

		commits, err := story.Commits(dir, storySince, storyUntil, author)
		if err != nil {
			fmt.Fprintln(os.Stderr, "devrounds:", err)
			os.Exit(1)
		}
		log := story.Log{
			Project: filepath.Base(dir),
			Period:  story.Period(storySince, storyUntil),
			By:      storyBy,
			Author:  label,
			Total:   len(commits),
			Groups:  story.GroupBy(commits, storyBy),
		}
		if log.Groups == nil {
			log.Groups = []story.Group{}
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			_ = enc.Encode(log)
			return
		}
		if len(commits) == 0 {
			who := ""
			if label != "" {
				who = " for " + label
			}
			fmt.Printf("No commits found%s in %s. Try --all, --author or a longer --since.\n", who, log.Period)
			return
		}
		switch storyFormat {
		case "markdown":
			story.PrintMarkdown(os.Stdout, log)
		case "standup":
			story.PrintStandup(os.Stdout, log)
		default:
			story.PrintText(os.Stdout, log)
		}
	},
}

func init() {
	f := storyCmd.Flags()
	f.StringVar(&storySince, "since", "7d", "start of the range: 7d, 2w or a date")
	f.StringVar(&storyUntil, "until", "", "end of the range (optional)")
	f.StringVar(&storyAuthor, "author", "", "only commits whose author matches this text")
	f.BoolVar(&storyAll, "all", false, "include every author, not just you")
	f.StringVar(&storyBy, "by", "area", "group by area or day")
	f.StringVar(&storyFormat, "format", "text", "text, markdown or standup")
	rootCmd.AddCommand(storyCmd)
}
