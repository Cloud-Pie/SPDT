package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
	"github.com/op/go-logging"
)

var (
	// VERSION is set during build
	VERSION string
	log = logging.MustGetLogger("spdt")
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "spd",
	Short: "Access spd from the command line",
	Long: `
   _____ ____  ____  ______
  / ___// __ \/ __ \/_  __/
  \__ \/ /_/ / / / / / /   
 ___/ / ____/ /_/ / / /    
/____/_/   /_____/ /_/     

	`,
}

// Execute adds all child commands to the root command
func Execute() {
	VERSION = "1.0"
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(startCmd)
	RootCmd.AddCommand(deriveCmd)
	RootCmd.AddCommand(deleteCmd)
	RootCmd.AddCommand(policiesCmd)
	RootCmd.AddCommand(invalidateCmd)

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func check(e error, msg string) {
	if e != nil {
		log.Fatalf(msg + " %s", e.Error())
		fmt.Println("An error has occurred")
		panic(e)
	}
}