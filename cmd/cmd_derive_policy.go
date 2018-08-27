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
	The configuration settings must be specified in a file config.yml.`,
	Run: derive,
}

func derive (cmd *cobra.Command, args []string) {
	fmt.Print("called derive")
	sysConfiguration := spd.ReadSysConfiguration()
	timeStart := sysConfiguration.ScalingHorizon.StartTime
	timeEnd := sysConfiguration.ScalingHorizon.EndTime
	spd.StartPolicyDerivation(timeStart,timeEnd)
}