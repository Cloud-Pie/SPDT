package cmd

import (
	"github.com/spf13/cobra"
	"fmt"
)

// policiesCmd represents the policies command
var policiesCmd = &cobra.Command{
	Use:   "policies",
	Short: "List policies",
	Long: `List stored policies.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("called policies")
	},
}
