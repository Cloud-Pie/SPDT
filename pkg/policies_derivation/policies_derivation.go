package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/config"
	"time"
	"github.com/Cloud-Pie/SPDT/rest_clients/scheduler"
	"math"
	"sort"
	"github.com/Cloud-Pie/SPDT/rest_clients/performance_profiles"
	"strconv"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("spdt")

//Interface for strategies of how to scale
type PolicyDerivation interface {
	CreatePolicies (processedForecast types.ProcessedForecast,serviceProfile types.ServiceProfile) []types.Policy
}

//Interface for strategies of when to scale
type TimeWindowDerivation interface {
	NumberIntervals()	int
	WindowDerivation(values []float64, times [] time.Time)	types.ProcessedForecast
}

func Policies(poiList []types.PoI, values []float64, times [] time.Time, sortedVMProfiles []types.VmProfile, serviceProfiles types.ServiceProfile, sysConfiguration config.SystemConfiguration) []types.Policy {
	var policies []types.Policy

	currentState,err := scheduler.CurrentState(sysConfiguration.SchedulerComponent.Endpoint + util.ENDPOINT_CURRENT_STATE)
	if err != nil {
		log.Error("Error to get current state")
	}

	timeWindows := SmallStepOverProvision{PoIList:poiList}
	processedForecast := timeWindows.WindowDerivation(values,times)

	mapVMProfiles := VMListToMap(sortedVMProfiles)

	switch sysConfiguration.PreferredAlgorithm {
	case util.NAIVE_ALGORITHM:
		naive := NaivePolicy {algorithm:util.NAIVE_ALGORITHM, timeWindow:timeWindows,
							 currentState:currentState, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies = naive.CreatePolicies(processedForecast, serviceProfiles)

	case util.NAIVE_TYPES_ALGORITHM:
		naive := NaiveTypesPolicy {algorithm:util.NAIVE_TYPES_ALGORITHM, timeWindow:timeWindows,
									mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies = naive.CreatePolicies(processedForecast, serviceProfiles)

	case util.SMALL_STEP_ALGORITHM:
		sstep := SStepRepackPolicy{algorithm:util.SMALL_STEP_ALGORITHM, timeWindow:timeWindows,
									mapVMProfiles:mapVMProfiles ,sysConfiguration: sysConfiguration}
		policies = sstep.CreatePolicies(processedForecast, serviceProfiles)

	case util.SEARCH_TREE_ALGORITHM:
		tree := TreePolicy {algorithm:util.SEARCH_TREE_ALGORITHM, timeWindow:timeWindows, currentState:currentState,
			sortedVMProfiles:sortedVMProfiles,mapVMProfiles:mapVMProfiles,sysConfiguration: sysConfiguration}
		policies = tree.CreatePolicies(processedForecast, serviceProfiles)

	case util.DELTA_REPACKED:
		algorithm := DeltaRepackedPolicy {algorithm:util.DELTA_REPACKED, timeWindow:timeWindows, currentState:currentState,
		sortedVMProfiles:sortedVMProfiles, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies = algorithm.CreatePolicies(processedForecast, serviceProfiles)
	default:
		//naive
		naive := NaivePolicy {algorithm:util.NAIVE_ALGORITHM, timeWindow:timeWindows,
			currentState:currentState, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies1 := naive.CreatePolicies(processedForecast, serviceProfiles)
		policies = append(policies, policies1...)
		//types
		naiveT := NaiveTypesPolicy {algorithm:util.NAIVE_TYPES_ALGORITHM, timeWindow:timeWindows,
			mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies2 := naiveT.CreatePolicies(processedForecast, serviceProfiles)
		policies = append(policies, policies2...)
		//sstep
		sstep := SStepRepackPolicy{algorithm:util.SMALL_STEP_ALGORITHM, timeWindow:timeWindows,
			mapVMProfiles:mapVMProfiles ,sysConfiguration: sysConfiguration}
		policies3 := sstep.CreatePolicies(processedForecast, serviceProfiles)
		policies = append(policies, policies3...)
		//delta repack
		algorithm := DeltaRepackedPolicy {algorithm:util.DELTA_REPACKED, timeWindow:timeWindows, currentState:currentState,
			sortedVMProfiles:sortedVMProfiles, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies4 := algorithm.CreatePolicies(processedForecast, serviceProfiles)
		policies = append(policies, policies4...)

	}
	return policies
}

func computeVMBootingTime(vmsScale types.VMScale, sysConfiguration config.SystemConfiguration) int {
	bootTime := 0
	// If Heterogeneous cluster, take the bigger cluster
	list := mapToList(vmsScale)
	sort.Slice(list, func(i, j int) bool {
		return list[i].Value > list[j].Value
	})

	//Check in db if already data is stored
	//Call API
	if len(list) > 0 {
		url := sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_VM_TIMES
		times, error := performance_profiles.GetBootShutDownProfile(url,list[0].Key, list[0].Value)
		if error != nil {
			log.Error("Error in bootingTime query", error.Error())
		}
		bootTime = times.BootTime
	}
	return bootTime
}

//Compute the termination time of a set of VMs
//Result must be in seconds
func computeVMTerminationTime(vmsScale types.VMScale, sysConfiguration config.SystemConfiguration) int {
	terminationTime := 0
	list := mapToList(vmsScale)
	sort.Slice(list, func(i, j int) bool {
		return list[i].Value > list[j].Value
	})

	//Check in db if already data is stored
	//Call API
	if len(list) > 0 {
		url := sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_VM_TIMES
		times, error := performance_profiles.GetBootShutDownProfile(url,list[0].Key, list[0].Value)
		if error != nil {
			log.Error("Error in terminationTime query %s", error.Error())
		}
		terminationTime = times.ShutDownTime
	}
	return terminationTime
}

func maxReplicasCapacityInVM(vmProfile types.VmProfile, resourceLimit types.Limit) int {
		m := float64(vmProfile.NumCores) / float64(resourceLimit.NumCores)
		n := float64(vmProfile.Memory) / float64(resourceLimit.Memory)
		numReplicas := math.Min(n,m)
		return int(numReplicas)
}

func selectProfile(performanceProfiles []types.PerformanceProfile) types.PerformanceProfile{
	//select the one with rank 1
	for _,p := range performanceProfiles {
		if p.RankWithLimits == 1 {
			return p
		}
	}
	return performanceProfiles[0]
}

func mapToList(vmSet map[string]int)[]types.StructMap {
	var ss [] types.StructMap
	for k, v := range vmSet {
		ss = append(ss, types.StructMap{k, v})
	}
	return ss
}

func setConfiguration(configurations *[]types.Configuration, state types.State, timeStart time.Time, timeEnd time.Time, name string, totalServicesBootingTime int, sysConfiguration config.SystemConfiguration, stateLoadCapacity float64) {
	nConfigurations := len(*configurations)
	if nConfigurations >= 1 && state.Equal((*configurations)[nConfigurations-1].State) {
		(*configurations)[nConfigurations-1].TimeEnd = timeEnd
	} else {
		var deltaTime int //time in seconds
		var finishTimeVMRemoved int
		var bootTimeVMAdded int

		//Adjust booting times for resources configuration
		if nConfigurations >= 1 {
			vmAdded, vmRemoved := deltaVMSet((*configurations)[nConfigurations-1].State.VMs ,state.VMs)
			//Adjust previous configuration
			if len(vmRemoved) > 0 {
				finishTimeVMRemoved = computeVMTerminationTime(vmRemoved, sysConfiguration)
				previousTimeEnd := (*configurations)[nConfigurations-1].TimeEnd
				(*configurations)[nConfigurations-1].TimeEnd = previousTimeEnd.Add(time.Duration(finishTimeVMRemoved) * time.Second)
			}
			if len(vmAdded) > 0 {
				bootTimeVMAdded = computeVMBootingTime(vmAdded, sysConfiguration)
			}
			//Select the biggest time
			if finishTimeVMRemoved > bootTimeVMAdded {
				deltaTime = finishTimeVMRemoved
			} else {
				deltaTime = bootTimeVMAdded
			}
		}

		startTime := timeStart.Add(-1 * time.Duration(deltaTime) * time.Second)       //Booting/Termination time VM
		startTime = startTime.Add(-1 * time.Duration(totalServicesBootingTime) * time.Second) //Start time containers
		state.LaunchTime = startTime
		state.Name = strconv.Itoa(nConfigurations) + "__" + name + "__" + startTime.Format(util.TIME_LAYOUT)
		*configurations = append(*configurations,
			types.Configuration {
				State:          state,
				TimeStart:      timeStart,
				TimeEnd:        timeEnd,
				Metrics:types.ConfigMetrics{CapacityTRN:stateLoadCapacity,},
			})
	}
}