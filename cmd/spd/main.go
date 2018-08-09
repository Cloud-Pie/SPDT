package main

import (
	Pservice "github.com/Cloud-Pie/SPDT/rest_clients/performance_profiles"
	Fservice "github.com/Cloud-Pie/SPDT/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/pkg/policies_derivation"
	"github.com/Cloud-Pie/SPDT/pkg/policy_evaluation"
	"github.com/Cloud-Pie/SPDT/pkg/policy_management"
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
	selectedPolicy			types.Policy
	vmProfiles				[]types.VmProfile
	policies				[]types.Policy
	timeWindowSize			time.Duration
	timeStart				time.Time
	timeEnd					time.Time
)

func main () {
	styleEntry()
	setLogger()
	fmt.Println("Logs can be accessed in %s", util.DEFAULT_LOGFILE)

	if FlagsVar.ConfigFile == "" {
		log.Info("Configuration file not specified. Default configuration will be used.")
		FlagsVar.ConfigFile = util.CONFIG_FILE
	}

	//Read Configuration File
	readSysConfiguration()
	timeStart = sysConfiguration.ScalingHorizon.StartTime
	timeEnd = sysConfiguration.ScalingHorizon.EndTime
	timeWindowSize = timeEnd.Sub(timeStart)

	out := make(chan types.Forecast)
	server := SetUpServer(out)
	go updatePolicyDerivation(out)
	go periodicPolicyDerivation()
	server.Run(":" + FlagsVar.Port)

}

func periodicPolicyDerivation() {
	for {
		startPolicyDerivation(timeStart,timeEnd)
		time.Sleep(timeWindowSize)
		timeStart.Add(timeWindowSize)
		timeEnd.Add(timeWindowSize)
	}
}

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

func updatePolicyDerivation(forecastChannel chan types.Forecast) {
	for forecast := range forecastChannel {
		shouldUpdate, newForecast, timeConflict := forecast_processing.UpdateForecast(forecast)
		if(shouldUpdate) {
			//Read Configuration File
			readSysConfiguration()
			//Request VM Profiles
			getVMProfiles()
			//Request Performance Profiles
			getServiceProfile()

			log.Info("Start points of interest search in time serie")
			poiList, values, times := forecast_processing.PointsOfInterest(newForecast)
			log.Info("Finish points of interest search in time serie")

			sort.Slice(vmProfiles, func(i, j int) bool {
				return vmProfiles[i].Pricing.Price <= vmProfiles[j].Pricing.Price
			})

			var timeInvalidation time.Time
			var oldPolicy types.Policy
			var indexConflict int

			policyDAO := storage.GetPolicyDAO()


			//verify if current time is greater than start window
			if time.Now().After(forecast.TimeWindowStart) {
				setNewPolicy(newForecast,poiList,values,times)
				oldPolicy, indexConflict = policy_management.ConflictTimeOldPolicy(forecast,timeConflict)
				timeInvalidation = oldPolicy.Configurations[indexConflict].TimeEnd
				selectedPolicy.Configurations[0].TimeStart = timeInvalidation
				//update policy
				oldPolicy.Configurations = append(oldPolicy.Configurations[:indexConflict], selectedPolicy.Configurations...)

			}else{
				//Discart completely old policy and create new one
				setNewPolicy(forecast,poiList,values,times)
				po, _ := policyDAO.FindOneByTimeWindow(forecast.TimeWindowStart, forecast.TimeWindowEnd)
				selectedPolicy.ID = po.ID
				oldPolicy = selectedPolicy
				timeInvalidation = forecast.TimeWindowStart
			}


			err := policyDAO.UpdateById(oldPolicy.ID,oldPolicy)
			if err != nil {
				log.Fatalf(err.Error())
			}

			reconfiguration.InvalidateStates(timeInvalidation, sysConfiguration.SchedulerComponent.Endpoint+util.ENDPOINT_INVALIDATE_STATES)
			log.Info("Start request Scheduler")
			reconfiguration.TriggerScheduler(selectedPolicy, sysConfiguration.SchedulerComponent.Endpoint+util.ENDPOINT_STATES)
			log.Info("Finish request Scheduler")
		}
	}
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

	serviceProfileDAO := storage.GetPerformanceProfileDAO()

	storedServiceProfiles,_ := serviceProfileDAO.FindAll()
	if(len(storedServiceProfiles)>0) {
		serviceProfiles = storedServiceProfiles[0]
	}else {
		log.Info("Start request Performance Profiles")
		serviceProfiles, err = Pservice.GetPerformanceProfiles(sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_SERVICE_PROFILES)
		if err != nil {
			log.Error(err.Error())
		}
		log.Info("Finish request Performance Profiles")

		//Store received information about Performance Profiles
		serviceProfiles.ID = bson.NewObjectId()
		err = serviceProfileDAO.Insert(serviceProfiles)
		if err != nil {
			log.Error(err.Error())
		}
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
}