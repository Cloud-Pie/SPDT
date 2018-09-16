package cmd

import (
	db "github.com/Cloud-Pie/SPDT/storage"
	"github.com/spf13/cobra"
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
	invalidateCmd.Flags().String("pId", "", "Policy ID")
	invalidateCmd.Flags().String("config-file", "config.yml", "Configuration file path")
	invalidateCmd.Flags().BoolVar(&forceInvalidation,"f", false, "Force the action")
}

func invalidate(cmd *cobra.Command, args []string) {
	if forceInvalidation {

		id := cmd.Flag("pId").Value.String()
		configFile := cmd.Flag("config-file").Value.String()
		systemConfiguration := server.ReadSysConfigurationFile(configFile)

		policyDAO := db.GetPolicyDAO(systemConfiguration.ServiceName)
		policy,err := policyDAO.FindByID(id)
		if err != nil {
			log.Fatalf("Error %s", err.Error())
		}

		//Request invalidation to scheduler
		err = server.InvalidateScalingStates(systemConfiguration, policy.TimeWindowStart)
		if err != nil {
			log.Fatalf("Error in request to invalidate states: %s", err.Error())
		} else {
			log.Info("Success invalidation request to scheduler")
		}

		//Recompute new set of policies
		timeStart := systemConfiguration.ScalingHorizon.StartTime
		timeEnd := systemConfiguration.ScalingHorizon.EndTime
		err = server.StartPolicyDerivation(timeStart,timeEnd,configFile)

		if err != nil {
			log.Fatalf("Error, new policy could not be derived %s", err.Error())

		} else {
			log.Info("New Policy derived")
		}

		//Delete all policies for that period
		err = policyDAO.DeleteById(id)
		if err != nil {
			log.Fatalf("Error, policy %s could not be removed from db: %s", id, err.Error())
		} else {
			log.Info("Policy %s invalidated successfully", id)
		}
	} else {
		log.Warning("Are you sure you want to invalidate this policy?, use the flag --f=true to force it")
	}
}
