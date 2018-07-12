package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/config"
	"time"
	"github.com/Cloud-Pie/SPDT/rest_clients/scheduler"
	"log"
	"math"
)

//TODO: Profile for Current config

//Interface for strategies of how to scale
type PolicyDerivation interface {
	CreatePolicies (processedForecast types.ProcessedForecast, mapVMProfiles map[string]types.VmProfile, serviceProfile types.ServiceProfile) []types.Policy
}

//Interface for strategies of when to scale
type TimeWindowDerivation interface {
	Cost()	int
	NumberIntervals()	int
	WindowDerivation(values []int, times [] time.Time)	types.ProcessedForecast
}

func Policies(poiList []types.PoI, values []int, times [] time.Time, mapVMProfiles map[string]types.VmProfile, ServiceProfiles types.ServiceProfile, configuration config.SystemConfiguration) []types.Policy {
	var policies []types.Policy
	currentState,err := scheduler.CurrentState(configuration.SchedulerComponent.Endpoint + util.ENDPOINT_CURRENT_STATE)
	if err != nil {
		log.Printf("Error to get current state")
	}

	switch configuration.PreferredAlgorithm {

	case util.NAIVE_ALGORITHM:
		timeWindows := SmallStepOverProvision{PoIList:poiList}
		processedForecast := timeWindows.WindowDerivation(values,times)
		naive := NaivePolicy {algorithm:util.NAIVE_ALGORITHM, limitNVMS:100, timeWindow:timeWindows, currentState:currentState}
		policies = naive.CreatePolicies(processedForecast, mapVMProfiles, ServiceProfiles)

	case util.NAIVE_TYPES_ALGORITHM:
		timeWindows := SmallStepOverProvision{PoIList:poiList}
		processedForecast := timeWindows.WindowDerivation(values,times)
		naive := NaiveTypesPolicy {algorithm:util.NAIVE_TYPES_ALGORITHM, timeWindow:timeWindows}
		policies = naive.CreatePolicies(processedForecast, mapVMProfiles, ServiceProfiles)

	case util.LINEAR_PROGRAMMING_STEP_ALGORITHM:
		timeWindows := SmallStepOverProvision{PoIList:poiList}
		processedForecast := timeWindows.WindowDerivation(values,times)
		policy := LPStepPolicy{algorithm:util.LINEAR_PROGRAMMING_STEP_ALGORITHM}
		policies = policy.CreatePolicies(processedForecast, mapVMProfiles, ServiceProfiles)

	case util.SMALL_STEP_ALGORITHM:
		timeWindows := SmallStepOverProvision{PoIList:poiList}
		processedForecast := timeWindows.WindowDerivation(values,times)
		sstep := SStepRepackPolicy {algorithm:util.SMALL_STEP_ALGORITHM, timeWindow:timeWindows}
		policies = sstep.CreatePolicies(processedForecast, mapVMProfiles, ServiceProfiles)

	case util.SEARCH_TREE_ALGORITHM:
		timeWindows := SmallStepOverProvision{PoIList:poiList}
		processedForecast := timeWindows.WindowDerivation(values,times)
		tree := TreePolicy {algorithm:util.SEARCH_TREE_ALGORITHM, timeWindow:timeWindows, currentState:currentState}
		policies = tree.CreatePolicies(processedForecast, mapVMProfiles, ServiceProfiles)
	default:
		/*timeWindows := SmallStepOverProvision{}
		timeWindows.PoIList = poiList
		processedForecast := timeWindows.WindowDerivation(values,times)
		naive := NaivePolicy {util.NAIVE_ALGORITHM, 100, timeWindows}
		policies = naive.CreatePolicies(processedForecast, mapVMProfiles, ServiceProfiles)
		sstep := SStepRepackPolicy{ util.SMALL_STEP_ALGORITHM}
		policies = append(naive.CreatePolicies(processedForecast, mapVMProfiles, ServiceProfiles),sstep.CreatePolicies(poiList,values,times, mapVMProfiles, ServiceProfiles)...)
	*/
	}
	return policies
}

//Adjust the times that were interpolated
func adjustTime(t time.Time, factor float64) time.Time {
	f := factor*3600
	return t.Add(time.Duration(f) * time.Second)
}

func ComputeVMBootingTime(mapVMProfiles map[string]types.VmProfile, vmsScale types.VMScale) int {
	bootingTime := 0
	//take the longestTime
	for k,_ := range vmsScale {
		vmBootTime := mapVMProfiles[k].BootTimeSec
		if bootingTime <  vmBootTime{
			bootingTime = vmBootTime
		}
	}
	return bootingTime
}

func ComputeVMTerminationTime(mapVMProfiles map[string]types.VmProfile, vmsScale types.VMScale) int {
	terminationTime := 0
	//take the longestTime
	for k,_ := range vmsScale {
		vmTermTime := mapVMProfiles[k].TerminationTimeSec
		if terminationTime <  vmTermTime{
			terminationTime = vmTermTime
		}
	}
	return terminationTime
}

func MaxReplicasInVM(vmProfile types.VmProfile, limit types.Limit) int {
	m := float64(vmProfile.NumCores) / float64(limit.NumCores)
	n := float64(vmProfile.Memory) / float64(limit.Memory)
	nScale := math.Min(n,m)
	return int(nScale)
}

func FindSuitableVMs(mapVMProfiles map[string]types.VmProfile, nReplicas int, limit types.Limit, preDefinedType string) types.VMScale {
	vmScale :=  make(map[string]int)
	bestVmScale :=  make(map[string]int)

	//Case when is restricted to a unique type of VM
	if preDefinedType != "" {
		profile := mapVMProfiles[preDefinedType]
		maxReplicas := MaxReplicasInVM(profile, limit)
		if maxReplicas > nReplicas {
			vmScale[preDefinedType] = 1
			return vmScale
		} else if maxReplicas > 0 {
			nScale := nReplicas / maxReplicas
			vmScale[preDefinedType] = int(nScale)
			return vmScale
		}
		//Case when it searches through all the types
	} else {
		for _,v := range mapVMProfiles {
			maxReplicas := MaxReplicasInVM(v, limit)
			if maxReplicas > nReplicas {
				vmScale[v.Type] = 1
			} else if maxReplicas > 0 {
				nScale := nReplicas / maxReplicas
				vmScale[v.Type] = int(nScale)
			}
		}
		var cheapest string
		cost := math.Inf(1)
		//Search for the cheapest key,value pair
		for k,v := range vmScale {
			price := mapVMProfiles[k].Pricing.Price * float64(v)
			if price < cost {
				cost = price
				cheapest = k
			}
		}
		bestVmScale[cheapest] = vmScale[cheapest]
	}
	return bestVmScale
}