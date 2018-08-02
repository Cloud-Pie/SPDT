package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"log"
	"github.com/Cloud-Pie/SPDT/config"
)

type NaivePolicy struct {
	algorithm  		string               //Algorithm's name
	timeWindow 		TimeWindowDerivation //Algorithm used to process the forecasted time serie
	currentState	types.State			 //Current State
	mapVMProfiles   map[string]types.VmProfile
	sysConfiguration	config.SystemConfiguration
}

func (p NaivePolicy) CreatePolicies(processedForecast types.ProcessedForecast, serviceProfile types.ServiceProfile) []types.Policy {
	policies := []types.Policy {}
	newPolicy := types.Policy{}
	newPolicy.StartTimeDerivation = time.Now()
	configurations := []types.Configuration {}

	for _, it := range processedForecast.CriticalIntervals {
		requests := it.Requests
		services :=  make(map[string]types.ServiceInfo)
		//Select the performance profile that fits better
		performanceProfile := selectProfile(serviceProfile.PerformanceProfiles)
		//Compute number of replicas needed depending on requests
		newNumServiceReplicas := int(math.Ceil(requests / performanceProfile.TRN)) * performanceProfile.NumReplicas
		services[serviceProfile.Name] = types.ServiceInfo{
											Scale: newNumServiceReplicas,
											CPU: performanceProfile.Limit.NumCores,
											Memory: performanceProfile.Limit.Memory, }

		vmType := p.currentVMType()
		newVMSet := p.FindSuitableVMs(newNumServiceReplicas, performanceProfile.Limit,vmType)
		state := types.State{}
		state.Services = services
		state.VMs = newVMSet

		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		totalServicesBootingTime := performanceProfile.BootTimeSec
		setConfiguration(&configurations,state,timeStart,timeEnd,serviceProfile.Name, totalServicesBootingTime, p.sysConfiguration)
	}

	totalConfigurations := len(configurations)
	//Add new policy
	newPolicy.Configurations = configurations
	newPolicy.FinishTimeDerivation = time.Now()
	newPolicy.Algorithm = p.algorithm
	newPolicy.ID = bson.NewObjectId()
	newPolicy.Metrics = types.Metrics {
		NumberConfigurations:totalConfigurations,
	}
	policies = append(policies, newPolicy)
	return policies
}

//Find suitable Vm(s) depending on resources limit, VM Type and Number of replicas that should be deployed
func (p NaivePolicy) FindSuitableVMs(targetNumReplicas int, resourceslimit types.Limit, vmType string) types.VMScale {
	vmScale := make(map[string]int)
	profile := p.mapVMProfiles[vmType]
	maxReplicas := maxReplicasCapacityInVM(profile, resourceslimit)
	if maxReplicas > targetNumReplicas {
		vmScale[vmType] = 1
		return vmScale
	} else if maxReplicas > 0 {
		numVMs := targetNumReplicas / maxReplicas
		vmScale[vmType] = int(numVMs)
		return vmScale
	}
	return vmScale
}

//Return the VM type used by the current Homogeneous VM cluster
func (p NaivePolicy) currentVMType() string{
	//Assumption for p approach: There is only 1 vm Type in current state
	var vmType string
	for k,_ := range p.currentState.VMs {
		vmType = k
	}
	if len(p.currentState.VMs) > 1 {
		log.Printf("Warning, current config has more than one VM type, type %s was selected to continue", vmType)
	}
	return vmType
}

