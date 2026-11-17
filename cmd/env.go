package cmd

import (
"fmt"

"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
Use:   "env",
Short: "Environment-variable drift between code, .env.example, Docker and CI",
Run: func(cmd *cobra.Command, args []string) {
fmt.Println("env: not implemented yet")
},
}

func init() {
rootCmd.AddCommand(envCmd)
}
