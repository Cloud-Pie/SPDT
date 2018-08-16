package cmd

import (
	"github.com/spf13/cobra"
	"fmt"
	"github.com/Cloud-Pie/SPDT/spd"
)

// deriveCmd represents the derive policy command
var deriveCmd = &cobra.Command{
	Use:   "derive",
	Short: "Derive scaling policy",
	Long: `Derive scaling policy for the specified scaling horizon:
	It expects a configuration file with the settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("called derive")
		sysConfiguration := spd.ReadSysConfiguration()
		timeStart := sysConfiguration.ScalingHorizon.StartTime
		timeEnd := sysConfiguration.ScalingHorizon.EndTime
		spd.StartPolicyDerivation(timeStart,timeEnd)
	},
}
