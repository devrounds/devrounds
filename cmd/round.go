package cmd

import (
"fmt"

"github.com/spf13/cobra"
)

var roundCmd = &cobra.Command{
Use:   "round",
Short: "Run check and env in one pass and print one verdict",
Run:   runRound,
}

func runRound(cmd *cobra.Command, args []string) {
fmt.Println("round: not implemented yet")
}

func init() {
rootCmd.AddCommand(roundCmd)
}
