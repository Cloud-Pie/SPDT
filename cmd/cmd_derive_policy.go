package cmd

import (
	"github.com/spf13/cobra"
	"github.com/Cloud-Pie/SPDT/server"
	"github.com/Cloud-Pie/SPDT/util"
)

// deriveCmd represents the derive policy command
var deriveCmd = &cobra.Command{
	Use:   "derive",
	Short: "Derive scaling policy",
	Long: `Derive scaling policy for the specified scaling horizon:
	The configuration settings must be specified in a file config.yml.`,
	Run: derive,
}

func init() {
	deriveCmd.Flags().String("config-file", "config.yml", "Configuration file path")
	deriveCmd.Flags().String("vm-prices-file","vm_profiles.json", "VM prices file path")
}

func derive (cmd *cobra.Command, args []string) {
	configFile := cmd.Flag("config-file").Value.String()
	sysConfiguration,_ := util.ReadConfigFile(configFile)
	timeStart := sysConfiguration.ScalingHorizon.StartTime
	timeEnd := sysConfiguration.ScalingHorizon.EndTime
	selectedPolicy, predictionID, err := server.StartPolicyDerivation(timeStart,timeEnd,configFile)
	if err != nil {
		log.Error("An error has occurred and policies have been not derived. Please try again. Details: %s", err)
	}else{
		//Schedule scaling states
		server.ScheduleScaling(sysConfiguration, selectedPolicy)
		server.SubscribeForecastingUpdates(sysConfiguration, predictionID)
	}
}