package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/devrounds/devrounds/internal/flow"
	"github.com/spf13/cobra"
)

var flowForgetYes bool

func openFlowStore() *flow.Store {
	path, err := flow.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "devrounds:", err)
		os.Exit(1)
	}
	s, err := flow.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "devrounds:", err)
		os.Exit(1)
	}
	return s
}

var flowCmd = &cobra.Command{
	Use:   "flow",
	Short: "Learn and save repeated command workflows",
	Long:  "flow records the commands you run in your shell (only after you add its hook), finds the sequences you repeat, and lets you save them as workflows. Everything stays on this machine.",
}

var flowStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether recording is on and how much is stored",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		s := openFlowStore()
		defer s.Close()
		state := "on"
		if !s.Enabled() {
			state = "off"
		}
		n, _ := s.Count()
		fmt.Printf("Recording:         %s\n", state)
		fmt.Printf("Commands recorded: %d (kept for %d days)\n", n, int(flow.Retention().Hours()/24))
		fmt.Printf("Database:          %s\n", s.Path)
	},
}

var flowOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Resume recording",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		s := openFlowStore()
		defer s.Close()
		if err := s.SetEnabled(true); err != nil {
			fmt.Fprintln(os.Stderr, "devrounds:", err)
			os.Exit(1)
		}
		fmt.Println("Recording is on.")
	},
}

var flowOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Pause recording",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		s := openFlowStore()
		defer s.Close()
		if err := s.SetEnabled(false); err != nil {
			fmt.Fprintln(os.Stderr, "devrounds:", err)
			os.Exit(1)
		}
		fmt.Println("Recording is off. Nothing new will be stored.")
	},
}

var flowForgetCmd = &cobra.Command{
	Use:   "forget",
	Short: "Delete every recorded command",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		s := openFlowStore()
		defer s.Close()
		n, _ := s.Count()
		if n == 0 {
			fmt.Println("Nothing to forget.")
			return
		}
		if !flowForgetYes {
			fmt.Printf("Delete all %d recorded commands? [y/N] ", n)
			ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			if strings.ToLower(strings.TrimSpace(ans)) != "y" {
				fmt.Println("Cancelled.")
				return
			}
		}
		if err := s.Forget(); err != nil {
			fmt.Fprintln(os.Stderr, "devrounds:", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted %d recorded commands.\n", n)
	},
}

func init() {
	flowForgetCmd.Flags().BoolVar(&flowForgetYes, "yes", false, "delete without asking")
	flowCmd.AddCommand(flowStatusCmd, flowOnCmd, flowOffCmd, flowForgetCmd)
	rootCmd.AddCommand(flowCmd)
}
