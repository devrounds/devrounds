package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/devrounds/devrounds/internal/flow"
	"github.com/spf13/cobra"
)

var (
	suggestMin     int
	suggestProject string
)

var flowSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Find command sequences you repeat (it only lists them, nothing is saved or run)",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		s := openFlowStore()
		defer s.Close()
		recs, err := s.All()
		if err != nil {
			fmt.Fprintln(os.Stderr, "devrounds:", err)
			os.Exit(1)
		}
		opts := flow.DefaultOptions()
		opts.MinCount = suggestMin
		opts.Project = suggestProject
		found := flow.Suggest(recs, opts)

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			_ = enc.Encode(found)
			return
		}
		if len(found) == 0 {
			fmt.Printf("No command sequence has repeated %d times yet (%d commands recorded). Keep working, or lower --min.\n", opts.MinCount, len(recs))
			return
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(tw, "#\tCOUNT\tPROJECT\tWORKFLOW\tLAST SEEN")
		for i, f := range found {
			fmt.Fprintf(tw, "%d\t%d\t%s\t%s\t%s\n", i+1, f.Count, f.Project, strings.Join(f.Steps, "  ->  "), f.LastSeen.Local().Format("2006-01-02"))
		}
		tw.Flush()
	},
}

func init() {
	flowSuggestCmd.Flags().IntVar(&suggestMin, "min", 3, "how many times a sequence must repeat")
	flowSuggestCmd.Flags().StringVar(&suggestProject, "project", "", "only look at this project")
	flowCmd.AddCommand(flowSuggestCmd)
}
