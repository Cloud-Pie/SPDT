package server

import (
	Fservice "github.com/Cloud-Pie/SPDT/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/types"
)

var requestsCapacityPerState types.RequestCapacitySupply

func StartPolicyDerivation(timeStart time.Time, timeEnd time.Time, configFile string) (types.Policy, string, error) {
	sysConfiguration,_ := util.ReadConfigFile(configFile)
	//pullingInterval = timeEnd.Sub(timeStart)

	//Request Performance Profiles
	error := getServiceProfile(sysConfiguration)
	if error != nil {
		return types.Policy{},"",error
	}
	//Request Forecasting
	log.Info("Start request Forecasting")
	forecastURL := sysConfiguration.ForecastingComponent.Endpoint + util.ENDPOINT_FORECAST
	forecast,err := fetchForecast(forecastURL, sysConfiguration.MainServiceName, timeStart, timeEnd)
	if err != nil {
		return types.Policy{},"",err
	}

	selectedPolicy,err := setNewPolicy(forecast, sysConfiguration)

	return selectedPolicy,forecast.IDPrediction, err
}

func fetchForecast(forecastURL string, mainService string, timeStart time.Time, timeEnd time.Time) (types.Forecast, error) {
	//Request Forecasting
	log.Info("Start request Forecasting")
	forecast,err := Fservice.GetForecast(forecastURL, timeStart, timeEnd)
	if err != nil {
		log.Error(err.Error())
		log.Info("Error in the request to get the forecasting")
		return types.Forecast{},err
	} else {
		log.Info("Finish request Forecasting")
	}

	//Retrieve data access to the database for forecasting
	forecastDAO := storage.GetForecastDAO(mainService)
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
	return forecast, nil
}

func updateDerivedPolicies(systemConfiguration util.SystemConfiguration){

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