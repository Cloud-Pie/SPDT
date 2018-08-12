package main

import (
	Fservice "github.com/Cloud-Pie/SPDT/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/pkg/forecast_processing"
	"sort"
	"github.com/Cloud-Pie/SPDT/storage"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/pkg/reconfiguration"
	"github.com/Cloud-Pie/SPDT/util"
	"fmt"
	"time"
)

func startPolicyDerivation(timeStart time.Time, timeEnd time.Time) {
	//Request VM Profiles
	getVMProfiles()
	//Request Performance Profiles
	getServiceProfile()

	//Request Forecasting
	log.Info("Start request Forecasting")
	forecast,err := Fservice.GetForecast(sysConfiguration.ForecastingComponent.Endpoint + util.ENDPOINT_FORECAST, timeStart, timeEnd)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Finish request Forecasting")

	//Store received information about forecast
	forecast.ID = bson.NewObjectId()

	forecastDAO := storage.GetForecastDAO()

	//Check if already exist, then update
	resultQuery,err := forecastDAO.FindAll()
	if len(resultQuery)==1 {
		id := resultQuery[0].ID
		forecast.TimeWindowStart = resultQuery[0].TimeWindowStart
		forecast.TimeWindowEnd = resultQuery[0].TimeWindowEnd
		forecastDAO.Update(id, forecast)
	} else {
		forecast.TimeWindowStart = forecast.ForecastedValues[0].TimeStamp
		l := len(forecast.ForecastedValues)
		forecast.TimeWindowEnd = forecast.ForecastedValues[l-1].TimeStamp
		err = forecastDAO.Insert(forecast)
		if err != nil {
			log.Error(err.Error())
		}
	}

	log.Info("Start points of interest search in time serie")
	poiList, values, times := forecast_processing.PointsOfInterest(forecast)
	log.Info("Finish points of interest search in time serie")

	sort.Slice(vmProfiles, func(i, j int) bool {
		return vmProfiles[i].Pricing.Price <=  vmProfiles[j].Pricing.Price
	})

	setNewPolicy(forecast, poiList,values,times)

	//Store policyByID
	policyDAO := storage.GetPolicyDAO()

	selectedPolicy.ID = bson.NewObjectId()
	err = policyDAO.Insert(selectedPolicy)
	if err != nil {
		log.Fatalf(err.Error())
	}

	log.Info("Start request Scheduler")
	reconfiguration.TriggerScheduler(selectedPolicy, sysConfiguration.SchedulerComponent.Endpoint + util.ENDPOINT_STATES)
	fmt.Sprintf(string(selectedPolicy.ID))
	log.Info("Finish request Scheduler")
}