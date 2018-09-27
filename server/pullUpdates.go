package server

import (
	Fservice "github.com/Cloud-Pie/SPDT/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/types"
	"fmt"
)

var requestsCapacityPerState types.RequestCapacitySupply

func StartPolicyDerivation(timeStart time.Time, timeEnd time.Time, ConfigFile string) error {
	sysConfiguration := ReadSysConfigurationFile(ConfigFile)
	timeStart = sysConfiguration.ScalingHorizon.StartTime
	timeEnd = sysConfiguration.ScalingHorizon.EndTime
	timeWindowSize = timeEnd.Sub(timeStart)

	//Request Performance Profiles
	getServiceProfile(sysConfiguration)

	//Request Forecasting
	log.Info("Start request Forecasting")
	forecastURL := sysConfiguration.ForecastingComponent.Endpoint + util.ENDPOINT_FORECAST
	forecast,err := Fservice.GetForecast(forecastURL, timeStart, timeEnd)
	if err != nil {
		log.Error(err.Error())
		log.Info("Error in the request to get the forecasting")
		return err
	} else {
		log.Info("Finish request Forecasting")
	}

	//Retrieve data access to the database for forecasting
	forecastDAO := storage.GetForecastDAO(sysConfiguration.MainServiceName)
	//Check if already exist, then update
	resultQuery,err := forecastDAO.FindOneByTimeWindow(timeStart, timeEnd)

	if err != nil && resultQuery.IDdb == ""{
		//error should be not found
		//Store received information about forecast
		forecast.IDdb = bson.NewObjectId()
		err = forecastDAO.Insert(forecast)
		if err != nil {
			log.Error(err.Error())
		}
	} else if resultQuery.IDdb != ""{
		id := resultQuery.IDdb
		forecast.IDdb = id
		forecastDAO.Update(id, forecast)
	}



	selectedPolicy,err := setNewPolicy(forecast, sysConfiguration)

	if err == nil {
		//Schedule scaling states
		ScheduleScaling(sysConfiguration, selectedPolicy)

		//Subscribe to the notifications
		SubscribeForecastingUpdates(sysConfiguration, selectedPolicy, forecast.IDPrediction)
	} else {
		//Send message to user
		fmt.Println(err.Error())
		log.Warning("Process stopped. %s", err.Error())
	}

	return err
}