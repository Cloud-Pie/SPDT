package server

import (
	Pservice "github.com/Cloud-Pie/SPDT/rest_clients/performance_profiles"
	Fservice "github.com/Cloud-Pie/SPDT/rest_clients/forecast"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/config"
	"github.com/Cloud-Pie/SPDT/storage"
	"fmt"
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"github.com/op/go-logging"
	"os"
	"path/filepath"
	"io"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"encoding/json"
	"sort"
	"github.com/Cloud-Pie/SPDT/pkg/derivation"
	"github.com/Cloud-Pie/SPDT/pkg/evaluation"
	"github.com/Cloud-Pie/SPDT/pkg/schedule"
	"github.com/Cloud-Pie/SPDT/pkg/forecast_processing"
)

var (
	FlagsVar = util.ParseFlags()
 	log = logging.MustGetLogger("spdt")
	policies				[]types.Policy
	timeWindowSize			time.Duration
	timeStart				time.Time
	timeEnd					time.Time
	ConfigFile				string
)

// Main function to start the scaling policy derivation
func Start(port string, configFile string) {

	//Print Tool Name
	styleEntry()

	//Set up the logs
	setLogger()

	//Read Configuration File
	ConfigFile = configFile
	sysConfiguration := ReadSysConfigurationFile(configFile)
	timeStart = sysConfiguration.ScalingHorizon.StartTime
	timeEnd = sysConfiguration.ScalingHorizon.EndTime
	timeWindowSize = timeEnd.Sub(timeStart)

	out := make(chan types.Forecast)
	server := SetUpServer(out)
	go updatePolicyDerivation(out)
	//go periodicPolicyDerivation()
	server.Run(":" + port)

}

//Periodically pull a new forecast for a new time window
//and derive a the correspondent new scaling policy
func periodicPolicyDerivation() {
	for {
		err := StartPolicyDerivation(timeStart,timeEnd, ConfigFile)
		if err != nil {
			log.Error("An error has occurred and policies have been not derived. Please try again. Details: %s", err)
		}else{
			timeStart.Add(timeWindowSize)
			timeEnd.Add(timeWindowSize)
			time.Sleep(timeWindowSize)
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

//Set where and how to write logs
func setLogger() {
	logFile := util.DEFAULT_LOGFILE
	os.MkdirAll(filepath.Dir(logFile), 0700)
	file, _ := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	multiOutput := io.MultiWriter(file, os.Stdout)
	backend2 := logging.NewLogBackend(multiOutput, "", 0)
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`, )
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	logging.SetBackend(backend2Formatter)
	log.Info("Logs can be accessed in %s", logFile)
}

//Read the configuration file with the setting to derive the scaling policies
func ReadSysConfiguration() config.SystemConfiguration {
	//var err error
	sysConfiguration, err := config.ParseConfigFile(ConfigFile)
	if err != nil {
		log.Errorf("Configuration file could not be processed %s", err)
	}
	return sysConfiguration
}

//Read the configuration file with the setting to derive the scaling policies
func ReadSysConfigurationFile(ConfigFile string) config.SystemConfiguration {
	//var err error
	sysConfiguration, err := config.ParseConfigFile(ConfigFile)
	if err != nil {
		log.Errorf("Configuration file could not be processed %s", err)
	}
	return sysConfiguration
}

//Fetch the profiles of the available Virtual Machines to generate the scaling policies
func getVMProfiles()[]types.VmProfile {
	var err error
	var vmProfiles	[]types.VmProfile
	log.Info("Start request VMs Profiles")
	//vmProfiles, err = Pservice.GetVMsProfiles(sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_VMS_PROFILES)

	data, err := ioutil.ReadFile("./mock_vms.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	err = json.Unmarshal(data, &vmProfiles)

	if err != nil {
		log.Error(err.Error())
		log.Info("Error in the request to get VMs Profiles")
		return vmProfiles
	} else {
		log.Info("Finish request VMs Profiles")
	}
	sort.Slice(vmProfiles, func(i, j int) bool {
		return vmProfiles[i].Pricing.Price <=  vmProfiles[j].Pricing.Price
	})

	return vmProfiles
}

//Fetch the performance profile of the microservice that should be scaled
func getServiceProfile(sysConfiguration config.SystemConfiguration){
	var err error
	var servicePerformanceProfile types.ServicePerformanceProfile
	serviceProfileDAO := storage.GetPerformanceProfileDAO(sysConfiguration.ServiceName)
	storedPerformanceProfiles,_ := serviceProfileDAO.FindAll()
	if len(storedPerformanceProfiles) == 0 {

		log.Info("Start request Performance Profiles")
		endpoint := sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_SERVICE_PROFILES
		servicePerformanceProfile, err = Pservice.GetServicePerformanceProfiles(endpoint,sysConfiguration.ServiceName, sysConfiguration.ServiceType )

		if err != nil {
			log.Error("Error in request Performance Profiles: %s",err.Error())
		} else {
			log.Info("Finish request Performance Profiles")
		}

		//Selects and stores received information about Performance Profiles
		for _,p := range servicePerformanceProfile.Profiles {
			mscSettings := []types.MSCSimpleSetting{}
			for _,msc := range p.MSCs {
				setting := types.MSCSimpleSetting{
					BootTimeSec: millisecondsToSeconds(msc.BootTimeMs),
					MSCPerSecond: msc.MSCPerSecond.RegBruteForce,
					Replicas: msc.Replicas,
					StandDevBootTimeSec: millisecondsToSeconds(msc.StandDevBootTimeMS),
				}
				mscSettings = append(mscSettings, setting)
			}
			performanceProfile := types.PerformanceProfile {
				ID: bson.NewObjectId(),Limit: p.Limits, MSCSettings: mscSettings,
			}
			err = serviceProfileDAO.Insert(performanceProfile)
			if err != nil {
				log.Error("Error Storing Performance Profiles: %s",err.Error())
			}
		}
	}
}

//Start Derivation of a new scaling policy for the specified scaling horizon and correspondent forecast
func setNewPolicy(forecast types.Forecast, poiList []types.PoI, values []float64, times []time.Time, sysConfiguration config.SystemConfiguration) (types.Policy, error){
	//Get VM Profiles
	vmProfiles := getVMProfiles()

	//Derive Strategies
	log.Info("Start policies derivation")
	policies = derivation.Policies(poiList, values, times, vmProfiles, sysConfiguration)
	log.Info("Finish policies derivation")

	log.Info("Start policies evaluation")
	//var err error
	selectedPolicy,err := evaluation.SelectPolicy(&policies, sysConfiguration, vmProfiles, forecast)
	if err != nil {
		log.Error("Error evaluation policies: %s", err.Error())
	}else {
		log.Info("Finish policies evaluation")
		policyDAO := storage.GetPolicyDAO(sysConfiguration.ServiceName)
		for _,p := range policies {
			err = policyDAO.Insert(p)
			if err != nil {
				log.Error("The policy with ID = %s could not be stored. Error %s\n", p.ID, err)
			}
		}
	}
	return  selectedPolicy, err
}

//Utility function to convert from milliseconds to seconds
func millisecondsToSeconds(m float64) float64{
	return m/1000
}

func ScheduleScaling(sysConfiguration config.SystemConfiguration, selectedPolicy types.Policy) {
	log.Info("Start request Scheduler")
	schedulerURL := sysConfiguration.SchedulerComponent.Endpoint + util.ENDPOINT_STATES
	err := schedule.TriggerScheduler(selectedPolicy, schedulerURL)
	if err != nil {
		log.Error("The scheduler request failed with error %s\n", err)
	} else {
		log.Info("Finish request Scheduler")
	}
}

func InvalidateScalingStates(sysConfiguration config.SystemConfiguration, timeInvalidation time.Time) error {
	log.Info("Start request Scheduler to invalidate states")
	statesInvalidationURL := sysConfiguration.SchedulerComponent.Endpoint+util.ENDPOINT_INVALIDATE_STATES
	err := schedule.InvalidateStates(timeInvalidation, statesInvalidationURL)
	if err != nil {
		log.Error("The scheduler request failed with error %s\n", err)
	} else {
		log.Info("Finish request Scheduler to invalidate states")
	}
	return err
}

func SubscribeForecastingUpdates(sysConfiguration config.SystemConfiguration, selectedPolicy types.Policy, idPrediction string){
	//TODO:Improve for a better pub/sub system
	log.Info("Start subscribe to prediction updates")
	forecastUpdatesURL := sysConfiguration.ForecastingComponent.Endpoint + util.ENDPOINT_SUBSCRIBE_NOTIFICATIONS
	requestsCapacityPerState := forecast_processing.GetMaxRequestCapacity(selectedPolicy)
	requestsCapacityPerState.IDPrediction = idPrediction
	requestsCapacityPerState.URL = util.ENDPOINT_RECIVE_NOTIFICATIONS
	err := Fservice.PostMaxRequestCapacities(requestsCapacityPerState, forecastUpdatesURL)
	if err != nil {
		log.Error("The subscription to prediction updates failed with error %s\n", err)
	} else {
		log.Info("Finish subscribe to prediction updates")
	}
}