package main

import (
	Pservice "github.com/Cloud-Pie/SPDT/rest_clients/performance_profiles"
	Fservice "github.com/Cloud-Pie/SPDT/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/pkg/policies_derivation"
	"github.com/Cloud-Pie/SPDT/pkg/policy_evaluation"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/config"
	storageProfile "github.com/Cloud-Pie/SPDT/storage/profile"
	storageForecast "github.com/Cloud-Pie/SPDT/storage/forecast"
	"gopkg.in/mgo.v2/bson"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/Cloud-Pie/SPDT/types"
	"net/http"
	"github.com/Cloud-Pie/SPDT/pkg/forecast_processing"
	"time"
	"sort"
)

var Log = util.NewLogger()
var FlagsVar = util.ParseFlags()
var priceModel types.PriceModel

func main () {

	styleEntry()

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

	sysConfiguration,err := config.ParseConfigFile(FlagsVar.ConfigFile)
	if err != nil {
		Log.Error.Fatalf("Configuration file could not be processed %s", err)
	}

	//Request Performance Profiles
	Log.Trace.Printf("Start request VMs Profiles")
	vmProfiles,err := Pservice.GetVMsProfiles(sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_VMS_PROFILES)
	if err != nil {
		Log.Error.Fatalf(err.Error())
	}
	Log.Trace.Printf("Finish request VMs Profiles")

	Log.Trace.Printf("Start request Performance Profiles")
	servicesProfiles,err := Pservice.GetPerformanceProfiles(sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_SERVICE_PROFILES)
	if err != nil {
		Log.Error.Fatalf(err.Error())
	}
	Log.Trace.Printf("Finish request Performance Profiles")

	//Store received information about Performance Profiles
	servicesProfiles.ID = bson.NewObjectId()
	serviceProfileDAO := storageProfile.PerformanceProfileDAO{
		util.DEFAULT_DB_SERVER_PROFILES,
		util.DEFAULT_DB_PROFILES,
	}
	serviceProfileDAO.Connect()
	err = serviceProfileDAO.Insert(servicesProfiles)
	if err != nil {
		Log.Error.Fatalf(err.Error())
	}

	//Request Forecasting
	Log.Trace.Printf("Start request Forecasting")
	timeStart := time.Now().Add(time.Hour)			//TODO: Adjust real times
	timeEnd := timeStart.Add(time.Hour * 24)
	forecast,err := Fservice.GetForecast(sysConfiguration.ForecastingComponent.Endpoint + util.ENDPOINT_FORECAST, timeStart, timeEnd)
	if err != nil {
		Log.Error.Fatalf(err.Error())
	}
	Log.Trace.Printf("Finish request Forecasting")

	//Store received information about forecast
	forecast.ID = bson.NewObjectId()
	forecastDAO := storageForecast.ForecastDAO{
		util.DEFAULT_DB_SERVER_FORECAST,
		util.DEFAULT_DB_FORECAST,
	}
	forecastDAO.Connect()
	err = forecastDAO.Insert(forecast)
	if err != nil {
		Log.Error.Fatalf(err.Error())
	}

	Log.Trace.Printf("Start points of interest search in time serie")
	poiList, values, times := forecast_processing.PointsOfInterest(forecast)
	Log.Trace.Printf("Finish points of interest search in time serie")


	//match prices to available VMs
	priceModel,err = policy_evaluation.ParsePricesFile(util.PRICES_FILE)
	mapVmPrice, unit := priceModel.MapPrices()
	//mapVMProfiles := make(map[string]types.VmProfile)

	for i,p := range vmProfiles {
		vmProfiles[i].Pricing.Price = mapVmPrice[p.Type]
		vmProfiles[i].Pricing.Unit = unit
		if (vmProfiles[i].Pricing.Price == 0.0) {
			Log.Warning.Printf("No price found for %s", p.Type)
		}
	}

	sort.Slice(vmProfiles, func(i, j int) bool {
		return vmProfiles[i].Pricing.Price <=  vmProfiles[j].Pricing.Price
	})

	var policies []types.Policy

	//Derive Strategies
	Log.Trace.Printf("Start policies derivation")
	policies = policies_derivation.Policies(poiList, values, times, vmProfiles, servicesProfiles, sysConfiguration)
	Log.Trace.Printf("Finish policies derivation")

	Log.Trace.Printf("Start policies evaluation")
	policy,err := policy_evaluation.SelectPolicy(policies, sysConfiguration)
	Log.Trace.Printf("Finish policies evaluation")

	if err != nil {
		Log.Trace.Printf("No policy found")
	} else {
		Log.Trace.Printf("Start request Scheduler")
		//reconfiguration.TriggerScheduler(policy, sysConfiguration.SchedulerComponent.Endpoint + util.ENDPOINT_STATES)
		fmt.Sprintf(string(policy.ID))
		Log.Trace.Printf("Finish request Scheduler")
	}

	return  policies
}

func serverCall(c *gin.Context) {
	policies := startPolicyDerivation()
	c.JSON(http.StatusOK, policies)
}



func styleEntry() {
	fmt.Println(`
   _____ ____  ____  ______
  / ___// __ \/ __ \/_  __/
  \__ \/ /_/ / / / / / /   
 ___/ / ____/ /_/ / / /    
/____/_/   /_____/ /_/     

	`)
}