package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/devrounds/devrounds/internal/flow"
	"github.com/spf13/cobra"
)

var (
	recordDir  string
	recordExit int
	recordMs   int64
	recentN    int
)

func debugf(format string, a ...any) {
	if os.Getenv("DEVROUNDS_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "devrounds: "+format+"\n", a...)
	}
}

var flowRecordCmd = &cobra.Command{
	Use:    "record",
	Short:  "Record one command (called by the shell hook, command text on stdin)",
	Hidden: true,
	Args:   cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 64*1024))
		if err != nil {
			debugf("%v", err)
			return
		}
		path, err := flow.DefaultPath()
		if err != nil {
			debugf("%v", err)
			return
		}
		s, err := flow.Open(path)
		if err != nil {
			debugf("%v", err)
			return
		}
		defer s.Close()
		dir := recordDir
		if dir == "" {
			dir, _ = os.Getwd()
		}
		if _, err := flow.Capture(s, string(data), dir, recordExit, recordMs); err != nil {
			debugf("%v", err)
		}
	},
}

func hookHelp(shell string) string {
	switch strings.ToLower(shell) {
	case "bash":
		return "# Try it in this session only:  eval \"$(devrounds flow init bash 2>/dev/null)\"\n# Keep it:  devrounds flow init bash 2>/dev/null >> ~/.bashrc\n"
	case "zsh":
		return "# Try it in this session only:  eval \"$(devrounds flow init zsh 2>/dev/null)\"\n# Keep it:  devrounds flow init zsh 2>/dev/null >> ~/.zshrc\n"
	}
	return "# Try it in this session only:  devrounds flow init powershell 2>$null | Out-String | Invoke-Expression\n# Keep it: add the snippet above to your profile (the path is in $PROFILE).\n"
}

var flowInitCmd = &cobra.Command{
	Use:   "init <bash|zsh|powershell>",
	Short: "Print the shell hook that records your commands (it never edits your profile)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		snippet, err := flow.Hook(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "devrounds:", err)
			os.Exit(2)
		}
		fmt.Print(snippet)
		fmt.Fprint(os.Stderr, "\n"+hookHelp(args[0]))
	},
}

var oneLine = strings.NewReplacer("\t", " ", "\r", "", "\n", " ")

var flowRecentCmd = &cobra.Command{
	Use:   "recent",
	Short: "Show the most recently recorded commands",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		s := openFlowStore()
		defer s.Close()
		recs, err := s.Recent(recentN)
		if err != nil {
			fmt.Fprintln(os.Stderr, "devrounds:", err)
			os.Exit(1)
		}
		if len(recs) == 0 {
			fmt.Println("Nothing recorded yet. To start, see: devrounds flow init")
			return
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(tw, "TIME\tEXIT\tMS\tPROJECT\tCOMMAND")
		for _, r := range recs {
			fmt.Fprintf(tw, "%s\t%d\t%d\t%s\t%s\n", r.Time.Format("2006-01-02 15:04:05"), r.ExitCode, r.Duration.Milliseconds(), r.Project, oneLine.Replace(r.Command))
		}
		tw.Flush()
	},
}

func init() {
	flowRecordCmd.Flags().StringVar(&recordDir, "dir", "", "folder the command ran in")
	flowRecordCmd.Flags().IntVar(&recordExit, "exit", 0, "exit code of the command")
	flowRecordCmd.Flags().Int64Var(&recordMs, "ms", 0, "how long the command took, in milliseconds")
	flowRecentCmd.Flags().IntVarP(&recentN, "number", "n", 10, "how many commands to show")
	flowCmd.AddCommand(flowRecordCmd, flowInitCmd, flowRecentCmd)
}
