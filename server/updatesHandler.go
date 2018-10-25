package server

import (
	Sservice "github.com/Cloud-Pie/SPDT/rest_clients/scheduler"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/storage"
	"time"
)

func invalidateOldPolicies(systemConfiguration util.SystemConfiguration){

	policyDAO := storage.GetPolicyDAO(systemConfiguration.MainServiceName)
	currentPolicies,err := policyDAO.FindAllByTimeWindow(timeStart,timeEnd)
	if len(currentPolicies) == 0 {
		log.Error("No policies found for the specified window")
	}
	err = InvalidateScalingStates(systemConfiguration, timeStart)

	//Delete all policies created previously for that period
	for _,p := range currentPolicies {
		err = policyDAO.DeleteById(p.ID.Hex())
		if err != nil {
			log.Fatalf("Error, policies could not be removed from db: %s",  err.Error())
		}
	}
	log.Info("Deleted previous scheduled states")
}

func InvalidateScalingStates(sysConfiguration util.SystemConfiguration, timeInvalidation time.Time) error {
	log.Info("Start request Scheduler to invalidate states")
	statesInvalidationURL := sysConfiguration.SchedulerComponent.Endpoint+util.ENDPOINT_INVALIDATE_STATES
	err := Sservice.InvalidateStates(timeInvalidation, statesInvalidationURL)
	if err != nil {
		log.Error("The scheduler request failed with error %s\n", err)
	} else {
		log.Info("Finish request Scheduler to invalidate states")
	}
	return err
}