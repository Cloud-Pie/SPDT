package server

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/pkg/forecast_processing"
	"sort"
	"time"
	"github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/pkg/policy_management"
	"github.com/Cloud-Pie/SPDT/pkg/reconfiguration"
	"github.com/Cloud-Pie/SPDT/util"
)

func updatePolicyDerivation(forecastChannel chan types.Forecast) {
	for forecast := range forecastChannel {
		shouldUpdate, newForecast, timeConflict := forecast_processing.UpdateForecast(forecast)
		if(shouldUpdate) {
			//Read Configuration File
			ReadSysConfiguration()
			//Request VM Profiles
			getVMProfiles()
			//Request Performance Profiles
			getServiceProfile()

			log.Info("Start points of interest search in time serie")
			poiList, values, times, err:= forecast_processing.PointsOfInterest(forecast)
			if err != nil {
				log.Error("The request failed with error %s\n", err)
				return
			} else {
				log.Info("Finish points of interest search in time serie")
			}

			sort.Slice(vmProfiles, func(i, j int) bool {
				return vmProfiles[i].Pricing.Price <= vmProfiles[j].Pricing.Price
			})

			var timeInvalidation time.Time
			var oldPolicy types.Policy
			var indexConflict int

			policyDAO := storage.GetPolicyDAO()


			//verify if current time is greater than start window
			if time.Now().After(forecast.TimeWindowStart) {
				setNewPolicy(newForecast,poiList,values,times)
				oldPolicy, indexConflict = policy_management.ConflictTimeOldPolicy(forecast,timeConflict)
				timeInvalidation = oldPolicy.ScalingActions[indexConflict].TimeEnd
				selectedPolicy.ScalingActions[0].TimeStart = timeInvalidation
				//update policy
				oldPolicy.ScalingActions = append(oldPolicy.ScalingActions[:indexConflict], selectedPolicy.ScalingActions...)

			}else{
				//Discart completely old policy and create new one
				setNewPolicy(forecast,poiList,values,times)
				po, _ := policyDAO.FindOneByTimeWindow(forecast.TimeWindowStart, forecast.TimeWindowEnd)
				selectedPolicy.ID = po.ID
				oldPolicy = selectedPolicy
				timeInvalidation = forecast.TimeWindowStart
			}


			err = policyDAO.UpdateById(oldPolicy.ID,oldPolicy)
			if err != nil {
				log.Error("The policy could not be updated. Error %s\n", err)
			}

			log.Info("Start request Scheduler to invalidate states")
			err = reconfiguration.InvalidateStates(timeInvalidation, sysConfiguration.SchedulerComponent.Endpoint+util.ENDPOINT_INVALIDATE_STATES)
			if err != nil {
				log.Error("The scheduler request failed with error %s\n", err)
			} else {
				log.Info("Finish request Scheduler to invalidate states")
			}

			log.Info("Start request Scheduler to create states")
			err = reconfiguration.TriggerScheduler(selectedPolicy, sysConfiguration.SchedulerComponent.Endpoint+util.ENDPOINT_STATES)
			if err != nil {
				log.Error("The scheduler request failed with error %s\n", err)
			} else {
				log.Info("Finish request Scheduler to create states")
			}
		}
	}
}
