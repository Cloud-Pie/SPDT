package server

import (
	Pservice "github.com/Cloud-Pie/SPDT/rest_clients/performance_profiles"
	Sservice "github.com/Cloud-Pie/SPDT/rest_clients/scheduler"
	"github.com/Cloud-Pie/SPDT/util"
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
	"github.com/Cloud-Pie/SPDT/pkg/schedule"
	"errors"
)

var (
	FlagsVar         = util.ParseFlags()
 	log              = logging.MustGetLogger("spdt")
	policies        []types.Policy
	pullingInterval time.Duration
	timeWindowSize  time.Duration
	timeStart       time.Time
	timeEnd         time.Time
	ConfigFile      string
	testJSON        []Sservice.StateToSchedule
)

// Main function to start the scaling policy derivation
func Start(port string, configFile string) {

	//Print Tool Name
	styleEntry()

	//Set up the logs
	setLogger()

	//Read Configuration File
	ConfigFile = configFile
	sysConfiguration,err := util.ReadConfigFile(configFile)
	if err != nil {
		log.Error("%s", err)
	}
	timeStart = sysConfiguration.ScalingHorizon.StartTime
	timeEnd = sysConfiguration.ScalingHorizon.EndTime
	timeWindowSize = timeEnd.Sub(timeStart)
	pullingInterval = time.Duration(sysConfiguration.PullingInterval)
	out := make(chan types.Forecast)
	server := SetUpServer(out)
	go updatePolicyDerivation(out)
	go periodicPolicyDerivation(sysConfiguration)
	server.Run(":" + port)

}

//Periodically pull a new forecast for a new time window
//and derive a the correspondent new scaling policy
func periodicPolicyDerivation(sysConfiguration util.SystemConfiguration) {
	for {
		selectedPolicy,forecastID, err := StartPolicyDerivation(timeStart,timeEnd, sysConfiguration)
		if err != nil {
			log.Error("An error has occurred and policies have been not derived. Please try again. Details: %s", err)
		}else{
			//Schedule scaling states
			ScheduleScaling(sysConfiguration, selectedPolicy)
			//Subscribe to the notifications
			SubscribeForecastingUpdates(sysConfiguration, forecastID)
			nextTimeStart := timeEnd.Add(timeWindowSize).Add(-10 * time.Minute )
			if time.Now().After(nextTimeStart) {
				timeStart = timeStart.Add(timeWindowSize)
				timeEnd = timeEnd.Add(timeWindowSize)
			}
		}
		time.Sleep(pullingInterval * time.Minute)
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
func ReadSysConfiguration() util.SystemConfiguration {
	//var err error
	sysConfiguration, err := util.ReadConfigFile(ConfigFile)
	if err != nil {
		log.Errorf("Configuration file could not be processed %s", err)
	}
	return sysConfiguration
}

//Fetch the profiles of the available Virtual Machines to generate the scaling policies
func getVMProfiles()([]types.VmProfile, error) {
	var err error
	var vmProfiles	[]types.VmProfile
	log.Info("Start request VMs Profiles")
	//vmProfiles, err = Pservice.GetVMsProfiles(sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_VMS_PROFILES)

	data, err := ioutil.ReadFile("./vm_profiles.json")
	if err != nil {
		log.Error(err.Error())
		return vmProfiles,err
	}
	err = json.Unmarshal(data, &vmProfiles)

	if err != nil {
		log.Error(err.Error())
		return vmProfiles, err
	} else {
		log.Info("Finish request VMs Profiles")
	}
	sort.Slice(vmProfiles, func(i, j int) bool {
		return vmProfiles[i].Pricing.Price <=  vmProfiles[j].Pricing.Price
	})

	return vmProfiles,err
}

//Fetch the performance profile of the microservice that should be scaled
func getVMBootingProfile(sysConfiguration util.SystemConfiguration, vmProfiles []types.VmProfile) error{
	var err error
	var vmBootingProfile types.InstancesBootShutdownTime
	vmBootingProfileDAO := storage.GetVMBootingProfileDAO()
	storedVMBootingProfiles,_ := vmBootingProfileDAO.FindAll()
	if len(storedVMBootingProfiles) == 0 {
		log.Info("Start request Performance Profiles")
		endpoint := sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_ALL_VM_TIMES
		csp := sysConfiguration.CSP
		region := sysConfiguration.Region
		for _, vm := range vmProfiles {
			vmBootingProfile, err = Pservice.GetAllBootShutDownProfilesByType(endpoint, vm.Type, region, csp)
			if err != nil {
				log.Error("Error in request VM Booting Profile for type %s. %s",vm.Type, err.Error())
			}
			vmBootingProfile.VMType = vm.Type
			vmBootingProfileDAO.Insert(vmBootingProfile)
		}
	}
	//defer serviceProfileDAO.Session.Close()
	return err
}

//Fetch the performance profile of the microservice that should be scaled
func fetchApplicationProfile(sysConfiguration util.SystemConfiguration) error {
	var err error
	var servicePerformanceProfile types.ServicePerformanceProfile
	serviceProfileDAO := storage.GetPerformanceProfileDAO(sysConfiguration.MainServiceName)
	storedPerformanceProfiles,_ := serviceProfileDAO.FindAll()
	if len(storedPerformanceProfiles) == 0 {

		log.Info("Start request Performance Profiles")
		endpoint := sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_SERVICE_PROFILES
		servicePerformanceProfile, err = Pservice.GetServicePerformanceProfiles(endpoint,sysConfiguration.AppName,
																sysConfiguration.AppType, sysConfiguration.MainServiceName)

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
					BootTimeSec: util.MillisecondsToSeconds(msc.BootTimeMs),
					MSCPerSecond: msc.MSCPerSecond.RegBruteForce,
					Replicas: msc.Replicas,
					StandDevBootTimeSec: util.MillisecondsToSeconds(msc.StandDevBootTimeMS),
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
	return err
	//defer serviceProfileDAO.Session.Close()
}

//Start Derivation of a new scaling policy for the specified scaling horizon and correspondent forecast
func setNewPolicy(forecast types.Forecast,sysConfiguration util.SystemConfiguration) (types.Policy, error){
	//Get VM Profiles
	var err error
	var selectedPolicy types.Policy
	vmProfiles,err := getVMProfiles()
	if err != nil {
		return  selectedPolicy, errors.New("VM profiles not found.")
	}
	//Get VM booting Profiles
	err = getVMBootingProfile(sysConfiguration, vmProfiles)
	if err != nil {
		return  selectedPolicy, err
	}

	//Derive Strategies
	log.Info("Start policies derivation")
	policies,err = derivation.Policies(vmProfiles, sysConfiguration, forecast)
	if err != nil {
		return selectedPolicy, err
	}
	log.Info("Finish policies derivation")

	log.Info("Start policies evaluation")
	//var err error
	selectedPolicy,err = derivation.SelectPolicy(&policies, sysConfiguration, vmProfiles, forecast)
	if err != nil {
		log.Error("Error evaluation policies: %s", err.Error())
	}else {
		log.Info("Finish policies evaluation")

		invalidateOldPolicies(sysConfiguration)
		policyDAO := storage.GetPolicyDAO(sysConfiguration.MainServiceName)
		for _,p := range policies {
			err = policyDAO.Insert(p)
			if err != nil {
				log.Error("The policy with ID = %s could not be stored. Error %s\n", p.ID, err)
			}
		}
	}
	return  selectedPolicy, err
}

func ScheduleScaling(sysConfiguration util.SystemConfiguration, selectedPolicy types.Policy) {
	log.Info("Start request Scheduler")
	schedulerURL := sysConfiguration.SchedulerComponent.Endpoint + util.ENDPOINT_STATES
	tset,err := schedule.TriggerScheduler(selectedPolicy, schedulerURL)
	testJSON = tset
	if err != nil {
		log.Error("The scheduler request failed with error %s\n", err)
	} else {
		log.Info("Finish request Scheduler")
	}
}

