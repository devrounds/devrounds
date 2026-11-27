package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/devrounds/devrounds/internal/flow"
	"github.com/spf13/cobra"
)

var (
	saveFrom    int
	saveMin     int
	saveProject string
	saveForce   bool
	saveYes     bool
	deleteYes   bool
)

func flowFail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "devrounds: "+format+"\n", a...)
	os.Exit(1)
}

// ask prints a question and returns true only for y or yes. No answer means no.
func ask(prompt string) bool {
	fmt.Print(prompt)
	ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	a := strings.ToLower(strings.TrimSpace(ans))
	return a == "y" || a == "yes"
}

func printWorkflow(w flow.Workflow, path string) {
	fmt.Printf("Workflow: %s\n", w.Name)
	if w.Description != "" {
		fmt.Printf("About:    %s\n", w.Description)
	}
	if w.Project != "" {
		fmt.Printf("Project:  %s\n", w.Project)
	}
	if w.Dir != "" {
		fmt.Printf("Folder:   %s\n", w.Dir)
	}
	if w.Shell != "" {
		fmt.Printf("Shell:    %s\n", w.Shell)
	}
	fmt.Println("Steps:")
	for i, s := range w.Steps {
		fmt.Printf("  %d. %s\n", i+1, s)
	}
	fmt.Printf("File:     %s\n", path)
}

var flowSaveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Save a suggested sequence as a workflow (asks first)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if err := flow.ValidName(name); err != nil {
			flowFail("%v", err)
		}
		if saveFrom < 1 {
			flowFail("--from <#> is required; run devrounds flow suggest to see the numbers")
		}
		s := openFlowStore()
		defer s.Close()
		recs, err := s.All()
		if err != nil {
			flowFail("%v", err)
		}
		opts := flow.DefaultOptions()
		opts.MinCount = saveMin
		opts.Project = saveProject
		found := flow.Suggest(recs, opts)
		if saveFrom > len(found) {
			flowFail("there is no suggestion #%d (devrounds flow suggest finds %d with these settings)", saveFrom, len(found))
		}
		w := flow.FromSuggestion(name, found[saveFrom-1])
		if err := w.Validate(); err != nil {
			flowFail("%v", err)
		}
		path, err := flow.WorkflowPath(name)
		if err != nil {
			flowFail("%v", err)
		}
		if _, err := os.Stat(path); err == nil && !saveForce {
			flowFail("a workflow named %q already exists; use --force to replace it", name)
		}
		printWorkflow(w, path)
		if !saveYes && !ask("Save this workflow? [y/N] ") {
			fmt.Println("Cancelled.")
			return
		}
		if _, err := flow.SaveWorkflow(w, saveForce); err != nil {
			flowFail("%v", err)
		}
		fmt.Printf("Saved to %s\n", path)
	},
}

var flowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved workflows",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		list, err := flow.ListWorkflows()
		if err != nil {
			flowFail("%v", err)
		}
		if len(list) == 0 {
			fmt.Println("No saved workflows yet. Find candidates with: devrounds flow suggest")
			return
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(tw, "NAME\tPROJECT\tSTEPS\tABOUT")
		for _, w := range list {
			fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n", w.Name, w.Project, len(w.Steps), w.Description)
		}
		tw.Flush()
	},
}

var flowShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a saved workflow",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		w, err := flow.LoadWorkflow(args[0])
		if err != nil {
			flowFail("%v", err)
		}
		path, _ := flow.WorkflowPath(args[0])
		printWorkflow(w, path)
	},
}

var flowDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a saved workflow (asks first)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := flow.LoadWorkflow(args[0]); err != nil {
			flowFail("%v", err)
		}
		if !deleteYes && !ask(fmt.Sprintf("Delete workflow %q? [y/N] ", args[0])) {
			fmt.Println("Cancelled.")
			return
		}
		if err := flow.DeleteWorkflow(args[0]); err != nil {
			flowFail("%v", err)
		}
		fmt.Printf("Deleted workflow %s.\n", args[0])
	},
}

func init() {
	f := flowSaveCmd.Flags()
	f.IntVar(&saveFrom, "from", 0, "number of the suggestion to save (see devrounds flow suggest)")
	f.IntVar(&saveMin, "min", 3, "use the same --min as flow suggest")
	f.StringVar(&saveProject, "project", "", "use the same --project as flow suggest")
	f.BoolVar(&saveForce, "force", false, "replace a workflow that already exists")
	f.BoolVar(&saveYes, "yes", false, "save without asking")
	flowDeleteCmd.Flags().BoolVar(&deleteYes, "yes", false, "delete without asking")
	flowCmd.AddCommand(flowSaveCmd, flowListCmd, flowShowCmd, flowDeleteCmd)
}
