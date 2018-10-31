package server

import (
	Fservice "github.com/Cloud-Pie/SPDT/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/types"
	"fmt"
	"github.com/Cloud-Pie/SPDT/planner/updatesHandler"
)

var requestsCapacityPerState types.RequestCapacitySupply

func StartPolicyDerivation(timeStart time.Time, timeEnd time.Time, sysConfiguration util.SystemConfiguration) (types.Policy, error) {
	var selectedPolicy types.Policy
	mainService := sysConfiguration.MainServiceName

	//Request Performance Profiles
	error := FetchApplicationProfile(sysConfiguration)
	if error != nil {
		return types.Policy{},error
	}
	//Request Forecasting
	forecast,err := fetchForecast(sysConfiguration, timeStart, timeEnd)
	if err != nil {
		return types.Policy{},err
	}

	//Get VM Profiles
	vmProfiles,err := ReadVMProfiles()
	if err != nil {
		return types.Policy{},err
	}
	//Get VM booting Profiles
	err = FetchVMBootingProfiles(sysConfiguration, vmProfiles)
	if err != nil {
		return types.Policy{},err
	}

	updateForecastInDB(forecast, sysConfiguration)

	policyDAO := storage.GetPolicyDAO(mainService)
	storedPolicy, err := policyDAO.FindSelectedByTimeWindow(timeStart, timeEnd)
	if err != nil {
		selectedPolicy,err = setNewPolicy(forecast, sysConfiguration, vmProfiles)
		ScheduleScaling(sysConfiguration, selectedPolicy)
	}else {
		shouldUpdate := updatesHandler.ValidateMSCThresholds(forecast,storedPolicy, sysConfiguration)
		if shouldUpdate {
			updatesHandler.InvalidateOldPolicies(sysConfiguration, timeStart, timeEnd )
			selectedPolicy,err = setNewPolicy(forecast, sysConfiguration, vmProfiles)
			ScheduleScaling(sysConfiguration, selectedPolicy)
			if err != nil {
				return types.Policy{},err
			}
		}
	}

	return selectedPolicy, err
}

func fetchForecast(sysConfiguration util.SystemConfiguration, timeStart time.Time, timeEnd time.Time) (types.Forecast,  error) {

	forecastURL := sysConfiguration.ForecastComponent.Endpoint + util.ENDPOINT_FORECAST
	mainService := sysConfiguration.MainServiceName

	//Request Forecasting
	log.Info("Start request Forecasting")
	forecast,err := Fservice.GetForecast(forecastURL, timeStart, timeEnd)
	if err != nil {
		return types.Forecast{},err
	} else {
		log.Info("Finish request Forecasting")
	}

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
	return forecast,  nil
}



func subscribeForecastingUpdates(sysConfiguration util.SystemConfiguration, idPrediction string){
	log.Info("Start subscribe to prediction updates")
	forecastUpdatesURL := sysConfiguration.ForecastComponent.Endpoint + util.ENDPOINT_SUBSCRIBE_NOTIFICATIONS
	urlNotifications := sysConfiguration.Host+util.ENDPOINT_RECIVE_NOTIFICATIONS
	fmt.Println(urlNotifications)
	err := Fservice.SubscribeNotifications(urlNotifications, idPrediction, forecastUpdatesURL)
	if err != nil {
		log.Error("The subscription to prediction updates failed with error %s\n", err)
	} else {
		log.Info("Finish subscribe to prediction updates")
	}
}