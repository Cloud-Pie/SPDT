package server

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/pkg/forecast_processing"
	"time"
	"github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/pkg/policy_management"
)

func updatePolicyDerivation(forecastChannel chan types.Forecast) {
	for forecast := range forecastChannel {
		shouldUpdate, newForecast, timeConflict := forecast_processing.UpdateForecast(forecast)
		if shouldUpdate {
			//Read Configuration File
			sysConfiguration := ReadSysConfiguration()

			//Request Performance Profiles
			getServiceProfile(sysConfiguration)

			log.Info("Start points of interest search in time serie")
			poiList, values, times, err:= forecast_processing.PointsOfInterest(forecast)
			if err != nil {
				log.Error("The request failed with error %s\n", err)
				return
			} else {
				log.Info("Finish points of interest search in time serie")
			}

			var timeInvalidation time.Time
			var oldPolicy types.Policy
			var indexConflict int
			var selectedPolicy types.Policy
			policyDAO := storage.GetPolicyDAO(sysConfiguration.ServiceName)

			//verify if current time is greater than start window
			if time.Now().After(forecast.TimeWindowStart) {
				selectedPolicy, err = setNewPolicy(newForecast,poiList,values,times, sysConfiguration)
				oldPolicy, indexConflict = policy_management.ConflictTimeOldPolicy(forecast,timeConflict)
				timeInvalidation = oldPolicy.ScalingActions[indexConflict].TimeEnd
				selectedPolicy.ScalingActions[0].TimeStart = timeInvalidation
				//update policy
				oldPolicy.ScalingActions = append(oldPolicy.ScalingActions[:indexConflict], selectedPolicy.ScalingActions...)

			}else{
				//Invalidate completely old policy and create new one
				selectedPolicy, err = setNewPolicy(forecast,poiList,values,times, sysConfiguration)
				po, _ := policyDAO.FindOneByTimeWindow(forecast.TimeWindowStart, forecast.TimeWindowEnd)
				selectedPolicy.ID = po.ID
				oldPolicy = selectedPolicy
				timeInvalidation = forecast.TimeWindowStart
			}

			err = policyDAO.UpdateById(oldPolicy.ID,oldPolicy)
			if err != nil {
				log.Error("The policy could not be updated. Error %s\n", err)
			}

			//Invalidate States
			err = InvalidateScalingStates(sysConfiguration, timeInvalidation)
			if err == nil {
				//Schedule scaling steps
				ScheduleScaling(sysConfiguration, selectedPolicy)
			}
			//Subscribe to the notifications
			SubscribeForecastingUpdates(sysConfiguration, selectedPolicy, forecast.IDPrediction)
		}
	}
}
