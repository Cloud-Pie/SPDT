package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

var (
	// VERSION is set during build
	VERSION string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "spd",
	Short: "Access spd from the command line",
}

// Execute adds all child commands to the root command
func Execute() {
	VERSION = "1.0"
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(startCmd)
	RootCmd.AddCommand(deriveCmd)
	RootCmd.AddCommand(deleteCmd)
	RootCmd.AddCommand(policiesCmd)

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
