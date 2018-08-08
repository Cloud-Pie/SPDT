package main

import (
	Pservice "github.com/Cloud-Pie/SPDT/rest_clients/performance_profiles"
	Fservice "github.com/Cloud-Pie/SPDT/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/pkg/policies_derivation"
	"github.com/Cloud-Pie/SPDT/pkg/policy_evaluation"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/config"
	"github.com/Cloud-Pie/SPDT/storage"
	"gopkg.in/mgo.v2/bson"
	"fmt"
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/pkg/forecast_processing"
	"time"
	"sort"
	"github.com/Cloud-Pie/SPDT/pkg/reconfiguration"
	"github.com/op/go-logging"
	"os"
	"path/filepath"
	"io"
)

var (
	FlagsVar = util.ParseFlags()

 	log = logging.MustGetLogger("spdt")
 	format = logging.MustStringFormatter(
			`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`, )
	sysConfiguration 		config.SystemConfiguration
	serviceProfiles 		types.ServiceProfile
	priceModel 				types.PriceModel
	selectedPolicy			types.Policy
	vmProfiles				[]types.VmProfile
	policies				[]types.Policy

)

func main () {
	styleEntry()
	setLogger()
	fmt.Println("Logs can be accessed in %s", util.DEFAULT_LOGFILE)

	if FlagsVar.ConfigFile == "" {
		log.Info("Configuration file not specified. Default configuration will be used.")
		FlagsVar.ConfigFile = util.CONFIG_FILE
	}

	out := make(chan types.Forecast)
	server := SetUpServer(out)
	go updatePolicyDerivation(out)
	go periodicPolicyDerivation()
	server.Run(":" + FlagsVar.Port)

}

func periodicPolicyDerivation() {
	for {
		startPolicyDerivation()
		time.Sleep(10 * time.Hour)
	}
}

func startPolicyDerivation() {
	//Read Configuration File
	readSysConfiguration()
	//Request VM Profiles
	getVMProfiles()
	//Request Performance Profiles
	getServiceProfile()

	//Request Forecasting
	log.Info("Start request Forecasting")
	timeStart := time.Now().Add(time.Hour)			//TODO: Adjust real times
	timeEnd := timeStart.Add(time.Hour * 24)
	forecast,err := Fservice.GetForecast(sysConfiguration.ForecastingComponent.Endpoint + util.ENDPOINT_FORECAST, timeStart, timeEnd)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Finish request Forecasting")


	//Store received information about forecast
	forecast.ID = bson.NewObjectId()
	forecastDAO := storage.ForecastDAO{
		Server:util.DEFAULT_DB_SERVER_FORECAST,
		Database:util.DEFAULT_DB_FORECAST,
	}
	forecastDAO.Connect()

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

		log.Info("Start request Scheduler")
		//reconfiguration.TriggerScheduler(selectedPolicy, sysConfiguration.SchedulerComponent.Endpoint + util.ENDPOINT_STATES)
		fmt.Sprintf(string(selectedPolicy.ID))
		log.Info("Finish request Scheduler")
}



func updatePolicyDerivation(forecastChannel chan types.Forecast){
	for forecast := range forecastChannel {
		forecastDAO := storage.ForecastDAO{
			Server:util.DEFAULT_DB_SERVER_FORECAST,
			Database:util.DEFAULT_DB_FORECAST,
		}
		forecastDAO.Connect()

		//Set start - end time window
		forecast.TimeWindowStart = forecast.ForecastedValues[0].TimeStamp
		l := len(forecast.ForecastedValues)
		forecast.TimeWindowEnd = forecast.ForecastedValues[l-1].TimeStamp

		var storedPolicy types.Policy
		var indexUpdate int

		//Compare with the previous forecast if sth changed
		resultQuery, err := forecastDAO.FindAll() //TODO: Write better query
		if len(resultQuery) == 1 {
			oldForecast := resultQuery[0]
			if shouldUpdate, timeConflict := conflict(forecast, oldForecast); shouldUpdate {

				id := resultQuery[0].ID
				forecastDAO.Update(id, forecast)

				//Update policy/new policy
				policyDAO := storage.PolicyDAO{
					Server:util.DEFAULT_DB_SERVER_POLICIES,
					Database:util.DEFAULT_DB_POLICIES,
				}

				//Search policy created for the time window
				storedPolicy, _ = policyDAO.FindByStartTime(forecast.TimeWindowStart)

				//search configuration state that contains the time where conflict was found
				indexUpdate = stateToUpdate(storedPolicy.Configurations, timeConflict)

				//set timeConflict as start of that configuration
				timeInvalidation := storedPolicy.Configurations [indexUpdate].TimeStart
				reconfiguration.InvalidateStates(timeInvalidation, sysConfiguration.SchedulerComponent.Endpoint+util.ENDPOINT_INVALIDATE_STATES)
			}
		} else {
			err = forecastDAO.Insert(forecast)
			if err != nil {
				log.Error(err.Error())
			}
		}

		//Read Configuration File
		readSysConfiguration()
		//Request VM Profiles
		getVMProfiles()
		//Request Performance Profiles
		getServiceProfile()

		log.Info("Start points of interest search in time serie")
		poiList, values, times := forecast_processing.PointsOfInterest(forecast)
		log.Info("Finish points of interest search in time serie")

		sort.Slice(vmProfiles, func(i, j int) bool {
			return vmProfiles[i].Pricing.Price <= vmProfiles[j].Pricing.Price
		})

		//Derive Strategies
		setNewPolicy(forecast,poiList,values,times)

		log.Info("Start request Scheduler")
		reconfiguration.TriggerScheduler(selectedPolicy, sysConfiguration.SchedulerComponent.Endpoint+util.ENDPOINT_STATES)
		fmt.Sprintf(string(selectedPolicy.ID))
		log.Info("Finish request Scheduler")

	}
}

func conflict(current types.Forecast, old types.Forecast) (bool, time.Time) {
	var timeWhenDiffer time.Time

	//case: Update in values but not in lenght of the window
	if len(current.ForecastedValues) == len(old.ForecastedValues) && current.TimeWindowStart.Equal(old.TimeWindowStart) {
			for i,in := range current.ForecastedValues {
				if in.Requests - old.ForecastedValues[i].Requests != 0{
					timeWhenDiffer = in.TimeStamp
					return true, timeWhenDiffer
				}
			}
	}
	//case: Update in values and lenght of the window
	if len(current.ForecastedValues) < len(old.ForecastedValues) && current.TimeWindowStart.After(old.TimeWindowStart) {

	}

	return false, timeWhenDiffer
}

func stateToUpdate(configurations []types.Configuration, conflictTime time.Time) int {
	index := 0
	for i,c := range configurations {
		if conflictTime.Equal(c.TimeStart) || conflictTime.After(c.TimeStart) {
			index = i
		}
	}
	return index
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

func setLogger() {
	logFile := util.DEFAULT_LOGFILE
	os.MkdirAll(filepath.Dir(logFile), 0700)
	file, _ := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	multiOutput := io.MultiWriter(file, os.Stdout)
	backend2 := logging.NewLogBackend(multiOutput, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	logging.SetBackend(backend2Formatter)
}

func readSysConfiguration(){
	var err error
	sysConfiguration, err = config.ParseConfigFile(FlagsVar.ConfigFile)
	if err != nil {
		log.Error("Configuration file could not be processed %s", err)
	}
}

func getVMProfiles(){
	var err error
	log.Info("Start request VMs Profiles")
	vmProfiles, err = Pservice.GetVMsProfiles(sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_VMS_PROFILES)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Finish request VMs Profiles")
}

func getServiceProfile(){
	var err error
	log.Info("Start request Performance Profiles")
	serviceProfiles, err = Pservice.GetPerformanceProfiles(sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_SERVICE_PROFILES)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Finish request Performance Profiles")

	//Store received information about Performance Profiles
	serviceProfiles.ID = bson.NewObjectId()
	serviceProfileDAO := storage.PerformanceProfileDAO{
		Server:util.DEFAULT_DB_SERVER_PROFILES,
		Database:util.DEFAULT_DB_PROFILES,
	}
	serviceProfileDAO.Connect()
	err = serviceProfileDAO.Insert(serviceProfiles)
	if err != nil {
		log.Error(err.Error())
	}
}

func setNewPolicy(forecast types.Forecast, poiList []types.PoI, values []float64, times []time.Time){

	//Derive Strategies
	log.Info("Start policies derivation")
	policies = policies_derivation.Policies(poiList, values, times, vmProfiles, serviceProfiles, sysConfiguration)
	log.Info("Finish policies derivation")

	log.Info("Start policies evaluation")
	selectedPolicy,_ = policy_evaluation.SelectPolicy(policies, sysConfiguration, vmProfiles)
	log.Info("Finish policies evaluation")

	selectedPolicy.TimeWindowStart = forecast.TimeWindowStart
	selectedPolicy.TimeWindowEnd = forecast.TimeWindowEnd

	//Store policy
	policyDAO := storage.PolicyDAO{
		Server:util.DEFAULT_DB_SERVER_POLICIES,
		Database:util.DEFAULT_DB_POLICIES,
	}
	policyDAO.Connect()
	err := policyDAO.Insert(selectedPolicy)
	if err != nil {
		log.Fatalf(err.Error())
	}
}