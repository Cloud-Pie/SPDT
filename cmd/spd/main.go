package main

import (
	Pservice "github.com/Cloud-Pie/SPDT/rest_clients/performance_profiles"
	"github.com/Cloud-Pie/SPDT/pkg/policies_derivation"
	"github.com/Cloud-Pie/SPDT/pkg/policy_evaluation"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/config"
	"github.com/Cloud-Pie/SPDT/storage"
	"gopkg.in/mgo.v2/bson"
	"fmt"
	"github.com/Cloud-Pie/SPDT/types"
	"time"
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

// Main function to start the scaling policy derivation
func main () {
	styleEntry()
	setLogger()

	if FlagsVar.ConfigFile == "" {
		log.Info("Configuration file not specified. Default configuration will be used.")
		FlagsVar.ConfigFile = util.CONFIG_FILE
		log.Info("Logs can be accessed in %s", util.DEFAULT_LOGFILE)
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
		log.Info("Error in the request to get VMs Profiles")
		return
	} else {
		log.Info("Finish request VMs Profiles")
	}
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
	selectedPolicy,_ = policy_evaluation.SelectPolicy(policies, sysConfiguration, vmProfiles, serviceProfiles)
	log.Info("Finish policies evaluation")

	selectedPolicy.TimeWindowStart = forecast.TimeWindowStart
	selectedPolicy.TimeWindowEnd = forecast.TimeWindowEnd
}