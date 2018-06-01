package main

import (
	Pservice "github.com/Cloud-Pie/SPDT/internal/rest_clients/performance_profiles"
	Fservice "github.com/Cloud-Pie/SPDT/internal/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/pkg/policies_derivation"
	"github.com/Cloud-Pie/SPDT/pkg/policy_selection"
	"github.com/Cloud-Pie/SPDT/internal/util"
	"github.com/Cloud-Pie/SPDT/config"
	costs "github.com/Cloud-Pie/SPDT/pkg/cost_efficiency"
	"github.com/Cloud-Pie/SPDT/pkg/forecast_processing"
	"github.com/Cloud-Pie/SPDT/pkg/performance_profiles"
	"gopkg.in/mgo.v2/bson"
	"fmt"
	"github.com/Cloud-Pie/SPDT/pkg/reconfiguration"
)


var Log = util.NewLogger()

func main () {
	var flagsVar = util.ParseFlags()
	if flagsVar.LogFile {
		Log.Info.Printf("Logs can be accessed in %s", util.DEFAULT_LOGFILE)
		Log.SetLogFile(util.DEFAULT_LOGFILE)
	}
	if flagsVar.ConfigFile == "" {
		Log.Info.Printf("Configuration file not specified. Default configuration will be used.")
		flagsVar.ConfigFile = util.CONFIG_FILE
	} else {
		_,err := config.ParseConfigFile(flagsVar.ConfigFile)
		if err != nil {
			Log.Error.Fatalf("Configuration file could not be processed %s", err)
		}
	}
	if flagsVar.PricesFile == "" {
		Log.Info.Printf("Prices file not specified. Default pricing file will be used.")
		flagsVar.PricesFile = util.PRICES_FILE
	} else {
		_,err := costs.ParsePricesFile(util.PRICES_FILE)
		if err != nil {
			Log.Error.Fatalf("Prices file could not be processed %s", err)
		}
	}

	/*for {
		time.Sleep(2 * time.Second)
		go startPolicyDerivation()
	}*/
	startPolicyDerivation()

	server := SetUpServer()
	server.Run(":" +flagsVar.Port)
}

func startPolicyDerivation() {
	//Request Performance Profiles
	Log.Trace.Printf("Start request Performance Profiles")
	vmProfiles,err := Pservice.GetPerformanceProfiles()
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
	data,err := Fservice.GetForecast()
	if err != nil {
		Log.Error.Fatalf(err.Error())
	}
	Log.Trace.Printf("Finish request Forecasting")

	forecast := forecast_processing.ProcessData(data)


	if (forecast.NeedToScale) {
		//Derive Strategies
		Log.Trace.Printf("Start policies derivation")
		policies := policies_derivation.Policies(forecast, vmProfiles)
		Log.Trace.Printf("Finish policies derivation")

		Log.Trace.Printf("Start policies evaluation")
		policy := policy_selection.SelectPolicy(policies)
		Log.Trace.Printf("Finish policies evaluation")

		Log.Trace.Printf("Start request Scheduler")
		reconfiguration.TriggerScheduler(policy)
		fmt.Sprintf(string(policy.ID))
		Log.Trace.Printf("Finish request Scheduler")

	} else {
		Log.Trace.Printf("No need to startPolicyDerivation for the requested time window")
	}
}
