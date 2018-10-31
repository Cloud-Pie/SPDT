package updatesHandler

import (
	Sservice "github.com/Cloud-Pie/SPDT/rest_clients/scheduler"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/storage"
	"time"
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("spdt")

func InvalidateOldPolicies(systemConfiguration util.SystemConfiguration, timeStart time.Time,timeEnd time.Time) bool{
	invalidated := false
	policyDAO := storage.GetPolicyDAO(systemConfiguration.MainServiceName)
	currentPolicies,err := policyDAO.FindAllByTimeWindow(timeStart,timeEnd)
	if len(currentPolicies) > 0 {
		err = InvalidateScalingStates(systemConfiguration, timeStart)
		if err != nil {
			log.Info("Deleted previous scheduled states")
		}
		//Delete all policies created previously for that period
		for _,p := range currentPolicies {
			err = policyDAO.DeleteById(p.ID.Hex())
			if err != nil {
				invalidated = false
				log.Fatalf("Error, policies could not be removed from db: %s",  err.Error())
			}
		}
		invalidated = true
	} else {
		log.Error("No policies found for the specified window")
	}

	return invalidated
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



func ValidateMSCThresholds(forecast types.Forecast,  policy types.Policy, sysConfiguration util.SystemConfiguration) bool{
	shouldScale := false
	predictions := forecast.ForecastedValues
	mainService := sysConfiguration.MainServiceName
	index := 0
	nPredictedValues := len(predictions)
	for _,c := range policy.ScalingActions {
		upperBoundCapacity := c.Metrics.RequestsCapacity
		lowerBoundCapacity := upperBoundCapacity - (upperBoundCapacity / float64(c.DesiredState.Services[mainService].Scale))

		for  index < nPredictedValues && c.TimeEnd.After(predictions[index].TimeStamp) {
			shouldScale = predictions[index].Requests > upperBoundCapacity || predictions[index].Requests < lowerBoundCapacity
			index=index+1
			if shouldScale {
				return shouldScale
			}
		}
	}
	return  shouldScale
}