package server

import (
	Fservice "github.com/Cloud-Pie/SPDT/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/pkg/forecast_processing"
	"sort"
	"github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/pkg/reconfiguration"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
	"gopkg.in/mgo.v2/bson"
)

func StartPolicyDerivation(timeStart time.Time, timeEnd time.Time) error {
	ReadSysConfiguration()
	timeStart = sysConfiguration.ScalingHorizon.StartTime
	timeEnd = sysConfiguration.ScalingHorizon.EndTime
	timeWindowSize = timeEnd.Sub(timeStart)

	//Request VM Profiles
	getVMProfiles()
	//Request Performance Profiles
	getServiceProfile()
	//Request Forecasting
	log.Info("Start request Forecasting")
	forecast,err := Fservice.GetForecast(sysConfiguration.ForecastingComponent.Endpoint + util.ENDPOINT_FORECAST, timeStart, timeEnd)
	if err != nil {
		log.Error(err.Error())
		log.Info("Error in the request to get the forecasting")
		return err
	} else {
		log.Info("Finish request Forecasting")
	}

	//Retrieve data access to the database for forecasting
	forecastDAO := storage.GetForecastDAO()

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


	sort.Slice(vmProfiles, func(i, j int) bool {
		return vmProfiles[i].Pricing.Price <=  vmProfiles[j].Pricing.Price
	})

	setNewPolicy(forecast, poiList,values,times)


	log.Info("Start request Scheduler")
	err = reconfiguration.TriggerScheduler(selectedPolicy, sysConfiguration.SchedulerComponent.Endpoint + util.ENDPOINT_STATES)
	if err != nil {
		log.Error("The scheduler request failed with error %s\n", err)
	} else {
		log.Info("Finish request Scheduler")
	}

	//TODO:Improve for a better pub/sub system
	log.Info("Start subscribe to prediction updates")
	url := sysConfiguration.ForecastingComponent.Endpoint + util.ENDPOINT_SUBSCRIBE_NOTIFICATIONS
	requestsCapacityPerState := forecast_processing.GetMaxRequestCapacity(selectedPolicy)
	requestsCapacityPerState.IDPrediction = forecast.IDPrediction
	requestsCapacityPerState.URL = util.ENDPOINT_RECIVE_NOTIFICATIONS
	Fservice.PostMaxRequestCapacities(requestsCapacityPerState, url)

	if err != nil {
		log.Error("The subscription to prediction updates failed with error %s\n", err)
	} else {
		log.Info("Finish subscribe to prediction updates")
	}

	return nil
}