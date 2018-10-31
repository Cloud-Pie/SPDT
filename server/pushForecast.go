package server

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/planner/updatesHandler"
	"github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/util"
	"fmt"
	"gopkg.in/mgo.v2/bson"
)

func updatePolicyDerivation(forecastChannel chan types.Forecast) {
	for forecast := range forecastChannel {
		timeStart := forecast.TimeWindowStart
		timeEnd := forecast.TimeWindowEnd
		mainService := sysConfiguration.MainServiceName

		//Read Configuration File
		sysConfiguration,_ := util.ReadConfigFile(util.CONFIG_FILE)
		if err := recover(); err != nil {
			fmt.Println(err)
		}
		//Request Performance Profiles
		FetchApplicationProfile(sysConfiguration)
		//Get VM Profiles
		vmProfiles,err := ReadVMProfiles()
		if err != nil {
			fmt.Println(err)
		}
		//Get VM booting Profiles
		err = FetchVMBootingProfiles(sysConfiguration, vmProfiles)
		if err != nil {
			fmt.Println(err)
		}
		updateForecastInDB(forecast, sysConfiguration)
		policyDAO := storage.GetPolicyDAO(mainService)
		storedPolicy, err := policyDAO.FindSelectedByTimeWindow(timeStart, timeEnd)
		shouldUpdate := updatesHandler.ValidateMSCThresholds(forecast,storedPolicy, sysConfiguration)
		if shouldUpdate {
			updatesHandler.InvalidateOldPolicies(sysConfiguration, timeStart, timeEnd )
			selectedPolicy,_ := setNewPolicy(forecast, sysConfiguration, vmProfiles)
			ScheduleScaling(sysConfiguration, selectedPolicy)
		} else {
			log.Info("Forecast updated. Scaling policy is still valid")
		}
	}
}

func updateForecastInDB(forecast types.Forecast, sysConfiguration util.SystemConfiguration) error {
	timeStart := forecast.TimeWindowStart
	timeEnd := forecast.TimeWindowEnd
	mainService := sysConfiguration.MainServiceName

	//Retrieve data access to the database for forecasting
	forecastDAO := storage.GetForecastDAO(mainService)
	//Check if already exist, then update
	resultQuery,err := forecastDAO.FindOneByTimeWindow(timeStart, timeEnd)

	if err != nil && resultQuery.IDdb == "" {
		//In case it is not already stored
		forecast.IDdb = bson.NewObjectId()
		err = forecastDAO.Insert(forecast)
		if err != nil {
			log.Error(err.Error())
		}
	} else if resultQuery.IDdb != "" {
		id := resultQuery.IDdb
		forecast.IDdb = id
		if resultQuery.IDPrediction != forecast.IDPrediction {
			subscribeForecastingUpdates(sysConfiguration, forecast.IDPrediction)
		}
		forecastDAO.Update(id, forecast)
	}
	return err
}