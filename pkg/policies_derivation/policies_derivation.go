package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/config"
	"time"
	"github.com/Cloud-Pie/SPDT/rest_clients/scheduler"
	"log"
	"math"
	"sort"
	"github.com/Cloud-Pie/SPDT/rest_clients/performance_profiles"
	"strconv"
)
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
		log.Printf("Error to get current state")
	}

	timeWindows := SmallStepOverProvision{PoIList:poiList}
	processedForecast := timeWindows.WindowDerivation(values,times)

	mapVMProfiles := make(map[string]types.VmProfile)
	for _,p := range sortedVMProfiles {
		mapVMProfiles[p.Type] = p
	}

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
							sysConfiguration: sysConfiguration}
		policies = tree.CreatePolicies(processedForecast, serviceProfiles)

	case util.DELTA_REPACKED:
		algorithm := DeltaRepackedPolicy {algorithm:util.DELTA_REPACKED, timeWindow:timeWindows, currentState:currentState,
		sortedVMProfiles:sortedVMProfiles, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies = algorithm.CreatePolicies(processedForecast, serviceProfiles)
	default:
		/*timeWindows := SmallStepOverProvision{}
		timeWindows.PoIList = poiList
		processedForecast := timeWindows.WindowDerivation(values,times)
		naive := NaiveVerticalPolicy {util.NAIVE_ALGORITHM, 100, timeWindows}
		policies = naive.CreatePolicies(processedForecast, mapVMProfiles, serviceProfiles)
		sstep := DeltaRepackedPolicy{ util.SMALL_STEP_ALGORITHM}
		policies = append(naive.CreatePolicies(processedForecast, mapVMProfiles, serviceProfiles),sstep.CreatePolicies(poiList,values,times, mapVMProfiles, serviceProfiles)...)
	*/
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
			log.Printf("Error in bootingTime query", error.Error())
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
			log.Printf("Error in terminationTime query %s", error.Error())
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

func copyMap(m map[string]int) map[string]int {
	newM := make(map[string] int)
	for k,v := range m {
		newM[k]=v
	}
	return newM
}

//Compare 2 VM Sets and returns a set of the new machines and one set with the machines that were removed
func deltaVMSet(current types.VMScale, candidate types.VMScale) (types.VMScale, types.VMScale){
	delta := types.VMScale{}
	startSet := types.VMScale{}
	shutdownSet := types.VMScale{}

	for k,_ :=  range current{
		if _,ok := candidate[k]; ok {
			delta[k] = -1 * (current[k] - candidate[k])
			if (delta[k]> 0) {
				startSet[k] = delta[k]
			} else if (delta[k] < 0) {
				shutdownSet[k] = -1 * delta[k]
			}
		}else {
			delta[k] = -1 * current[k]
			shutdownSet[k] =  current[k]
		}
	}

	for k,_ :=  range candidate {
		if _,ok := current[k]; !ok {
			delta[k] = candidate[k]
			startSet[k] = candidate[k]
		}
	}
	return startSet, shutdownSet
}

func setConfiguration(configurations *[]types.Configuration, state types.State, timeStart time.Time, timeEnd time.Time, name string, totalServicesBootingTime int, sysConfiguration config.SystemConfiguration) {
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
			})
	}
}