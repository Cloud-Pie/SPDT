package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/config"
	"time"
	"github.com/Cloud-Pie/SPDT/rest_clients/scheduler"
	"log"
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
		naive := NaiveTypesPolicy {algorithm:util.NAIVE_ALGORITHM, timeWindow:timeWindows}
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

func ComputeVMBootingTime(mapVMProfiles map[string]types.VmProfile, vmsScale []types.VmScale) int {
	bootingTime := 0
	//take the longestTime
	for _,s := range vmsScale {
		vmBootTime := mapVMProfiles[s.Type].BootTimeSec
		if bootingTime <  vmBootTime{
			bootingTime = vmBootTime
		}
	}
	return bootingTime
}

func ComputeVMTerminationTime(mapVMProfiles map[string]types.VmProfile, vmsScale []types.VmScale) int {
	terminationTime := 0
	//take the longestTime
	for _,s := range vmsScale {
		vmTermTime := mapVMProfiles[s.Type].TerminationTimeSec
		if terminationTime <  vmTermTime{
			terminationTime = vmTermTime
		}
	}
	return terminationTime
}