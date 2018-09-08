package cmd

import (
	"github.com/spf13/cobra"
	"fmt"
	"github.com/Cloud-Pie/SPDT/server"
)

// startCmd represents the start service command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start service",
	Long: "Start the scaling policy derivator",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("called start")
		server.Start()
	},
}
