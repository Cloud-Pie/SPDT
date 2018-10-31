package cmd

import (
	"github.com/spf13/cobra"
	"github.com/Cloud-Pie/SPDT/planner/updatesHandler"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
	"github.com/Cloud-Pie/SPDT/server"
)

// invalidateCmd represents the invalidate policies command
var invalidateCmd = &cobra.Command{
	Use:   "invalidate",
	Short: "Invalidate policy",
	Long: "Invalidate a policy",
	Run: invalidate,
}

var forceInvalidation bool

func init() {
	invalidateCmd.Flags().String("start-time", "", "Start time of the horizon span")
	invalidateCmd.Flags().String("end-time", "", "End time of the horizon span")
	invalidateCmd.Flags().String("config-file", "config.yml", "Configuration file path")
	invalidateCmd.Flags().BoolVar(&forceInvalidation,"f", false, "Force the action")
}

func invalidate(cmd *cobra.Command, args []string) {
	var(
		timeStart time.Time
		timeEnd   time.Time
		err       error
	)
	if forceInvalidation {
		timeStart,err = time.Parse(util.UTC_TIME_LAYOUT,cmd.Flag("start-time").Value.String())
		check(err, "Time start window no valid")
		timeEnd,err = time.Parse(util.UTC_TIME_LAYOUT,cmd.Flag("end-time").Value.String())
		check(err, "Time end window no valid")
		configFile := cmd.Flag("config-file").Value.String()
		systemConfiguration,_ := util.ReadConfigFile(configFile)

		invalidated := updatesHandler.InvalidateOldPolicies(systemConfiguration, timeStart, timeEnd )
		if invalidated {
			//Recompute new set of policies
			_, err2 := server.StartPolicyDerivation(timeStart,timeEnd,systemConfiguration)
			check(err2, "New policy could not be derived")
		}

	} else {
		log.Warning("Are you sure you want to invalidate this policy?, use the flag --f=true to force it")
	}
}
