package server

import (
	Fservice "github.com/Cloud-Pie/SPDT/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/pkg/forecast_processing"
	"github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/types"
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
	forecastDAO := storage.GetForecastDAO(sysConfiguration.ServiceName)
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

	log.Info("Start points of interest search in time serie")
	poiList, values, times, err:= forecast_processing.PointsOfInterest(forecast)
	if err != nil {
		log.Error("The request failed with error %s\n", err)
		return err
	} else {
		log.Info("Finish points of interest search in time serie")
	}

	selectedPolicy,err := setNewPolicy(forecast, poiList,values,times, sysConfiguration)

	if err == nil {
		//Schedule scaling states
		ScheduleScaling(sysConfiguration, selectedPolicy)

		//Subscribe to the notifications
		SubscribeForecastingUpdates(sysConfiguration, selectedPolicy, forecast.IDPrediction)
	}

	return err
}