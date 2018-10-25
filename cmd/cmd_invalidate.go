package cmd

import (
	db "github.com/Cloud-Pie/SPDT/storage"
	"github.com/spf13/cobra"
	"github.com/Cloud-Pie/SPDT/server"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
	"fmt"
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

		policyDAO := db.GetPolicyDAO(systemConfiguration.MainServiceName)
		currentPolicies,err := policyDAO.FindAllByTimeWindow(timeStart,timeEnd)
		check(err, "No policies found for the specified window")
		if len(currentPolicies) == 0 {
			log.Fatalf("No policies found for the specified window")
			fmt.Println("No policies found for the specified window")
		}

		//Recompute new set of policies
		selectedPolicy,predictionID, err2 := server.StartPolicyDerivation(timeStart,timeEnd,systemConfiguration)
		check(err2, "New policy could not be derived")

		log.Info("New Policy derived")
		err = server.InvalidateScalingStates(systemConfiguration, timeStart)
		check(err, "States could not be invalidated")
		log.Info("Invalidated previous scheduled states")
		server.ScheduleScaling(systemConfiguration, selectedPolicy)
		log.Info("Schedule new states")
		server.SubscribeForecastingUpdates(systemConfiguration, predictionID)
		log.Info("Subscribed for notifications")

		//Delete all policies created previously for that period
		for _,p := range currentPolicies {
			err = policyDAO.DeleteById(p.ID.Hex())
			if err != nil {
				log.Fatalf("Error, policies could not be removed from db: %s",  err.Error())
			}
		}


	} else {
		log.Warning("Are you sure you want to invalidate this policy?, use the flag --f=true to force it")
	}
}
