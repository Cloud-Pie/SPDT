package cmd

import (
	"github.com/spf13/cobra"
	"github.com/Cloud-Pie/SPDT/server"
)


// startCmd represents the start service command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start service",
	Long: "Start the scaling policy derivator",
	Run: startServer,
}

func init() {
	startCmd.Flags().String("http-port","8083", "Http Port")
	startCmd.Flags().String("config-file", "config.yml", "Configuration file path")
}

func startServer (cmd *cobra.Command, args []string) {
	port := cmd.Flag("http-port").Value.String()
	configFile := cmd.Flag("config-file").Value.String()
	server.Start(port,configFile)
}