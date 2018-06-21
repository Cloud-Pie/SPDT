package main

import (
	Pservice "github.com/Cloud-Pie/SPDT/rest_clients/performance_profiles"
	Fservice "github.com/Cloud-Pie/SPDT/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/pkg/policies_derivation"
	"github.com/Cloud-Pie/SPDT/pkg/policy_evaluation"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/config"
	"github.com/Cloud-Pie/SPDT/pkg/performance_profiles"
	"gopkg.in/mgo.v2/bson"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/Cloud-Pie/SPDT/types"
	"net/http"
	"github.com/Cloud-Pie/SPDT/pkg/forecast_processing"
	"time"
)

var Log = util.NewLogger()
var FlagsVar = util.ParseFlags()
var priceModel types.PriceModel

func main () {

	if FlagsVar.LogFile {
		Log.Info.Printf("Logs can be accessed in %s", util.DEFAULT_LOGFILE)
		Log.SetLogFile(util.DEFAULT_LOGFILE)
	}

	if FlagsVar.ConfigFile == "" {
		Log.Info.Printf("Configuration file not specified. Default configuration will be used.")
		FlagsVar.ConfigFile = util.CONFIG_FILE
	}

	if FlagsVar.PricesFile == "" {
		Log.Info.Printf("Prices file not specified. Default pricing file will be used.")
		FlagsVar.PricesFile = util.PRICES_FILE
	} else {
		var err error
		priceModel,err = policy_evaluation.ParsePricesFile(util.PRICES_FILE)
		if err != nil {
			Log.Error.Fatalf("Prices file could not be processed %s", err)
		}
	}

	server := SetUpServer()
	server.Run(":" + FlagsVar.Port)

	/*for {
		go startPolicyDerivation(configuration)
		time.Sleep(24 * time.Hour)
	}*/

	//startPolicyDerivation(configuration)

}

func startPolicyDerivation() [] types.Policy {

	configuration,err := config.ParseConfigFile(FlagsVar.ConfigFile)
	if err != nil {
		Log.Error.Fatalf("Configuration file could not be processed %s", err)
	}

	//Request Performance Profiles
	Log.Trace.Printf("Start request Performance Profiles")
	vmProfiles,err := Pservice.GetPerformanceProfiles(configuration.PerformanceProfilesComponent.Endpoint)
	if err != nil {
		Log.Error.Fatalf(err.Error())
	}
	Log.Trace.Printf("Finish request Performance Profiles")

	//Store received information about Performance Profiles
	vmProfiles.ID = bson.NewObjectId()
	vmProfileDAO := performance_profiles.PerformanceProfileDAO{
		util.DEFAULT_DB_SERVER_PROFILES,
		util.DEFAULT_DB_PROFILES,
	}
	vmProfileDAO.Connect()
	err = vmProfileDAO.Insert(vmProfiles)
	if err != nil {
		Log.Error.Fatalf(err.Error())
	}

	//Request Forecasting
	Log.Trace.Printf("Start request Forecasting")
	timeStart := time.Now().Add(time.Hour)			//TODO: Adjust real times
	timeEnd := timeStart.Add(time.Hour * 24)
	data,err := Fservice.GetForecast(configuration.ForecastingComponent.Endpoint, timeStart, timeEnd)
	if err != nil {
		Log.Error.Fatalf(err.Error())
	}
	Log.Trace.Printf("Finish request Forecasting")


	Log.Trace.Printf("Start points of interest search in time serie")
	poiList, values, times := forecast_processing.PointsOfInterest(data)
	Log.Trace.Printf("Finish points of interest search in time serie")


	var policies []types.Policy

	//Derive Strategies
	Log.Trace.Printf("Start policies derivation")
	policies = policies_derivation.Policies(poiList, values, times, vmProfiles, configuration, priceModel)
	Log.Trace.Printf("Finish policies derivation")

	Log.Trace.Printf("Start policies evaluation")
	policy,err := policy_evaluation.SelectPolicy(policies)
	Log.Trace.Printf("Finish policies evaluation")

	if err != nil {
		Log.Trace.Printf("No policy found")
	} else {
		Log.Trace.Printf("Start request Scheduler")
		//reconfiguration.TriggerScheduler(policy, configuration.SchedulerComponent.Endpoint)
		fmt.Sprintf(string(policy.ID))
		Log.Trace.Printf("Finish request Scheduler")
	}

	return  policies
}

func serverCall(c *gin.Context) {
	policiy := startPolicyDerivation()
	c.JSON(http.StatusOK, policiy[0])
}
