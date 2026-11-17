package cmd

import (
"fmt"

"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
Use:   "check",
Short: "Readiness checks: runtime, binaries, ports, services",
Run: func(cmd *cobra.Command, args []string) {
fmt.Println("check: not implemented yet")
},
}

func init() {
rootCmd.AddCommand(checkCmd)
}
