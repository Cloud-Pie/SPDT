package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"gopkg.in/mgo.v2/bson"
	"math"
	"github.com/Cloud-Pie/SPDT/util"
)

type SStepPolicy struct {
	priceModel types.PriceModel
	algorithm string
}

func (policy SStepPolicy) CreatePolicies(poiList []types.PoI, values []int, times [] time.Time, performanceProfile types.PerformanceProfile) [] types.Policy {
	listVm := performanceProfile.PerformanceModels[0].VmProfiles; //TODO: Change according to CSP
	policies := []types.Policy {}

	new_policy := types.Policy{}
	new_policy.StartTimeDerivation = time.Now()

	timeWindows := SmallStepOverProvision{}
	timeWindows.PoIList = poiList
	processedForecast := timeWindows.WindowDerivation(values,times)

	for i := range listVm {
		new_policy := types.Policy{}
		new_policy.StartTimeDerivation = time.Now()
		configurations := []types.Configuration {}
		stateName := performanceProfile.DockerImageApp + time.Now().Format(util.TIME_LAYOUT)

		for ind, it := range  processedForecast.CriticalIntervals {
			requests := it.Requests
			nVms := int(math.Ceil(float64(requests) / float64(listVm[i].TRN)))
			servicesList := listVm[i].ServiceInfo
			services := [] types.Service{}
			totalServicesBootingTime := 0
			for _,s := range servicesList {
				services = append(services, types.Service{ s.Name, s.NumReplicas})
				totalServicesBootingTime += s.ContainerStartTime
			}
			vms := [] types.VmScale {{listVm[i].VmInfo.Type, nVms}}

			state :=  types.State{}
			state.Services = services
			state.VMs = vms
			state.Name = stateName
			nConfigurations := len(configurations)
			if (nConfigurations >= 1){
				keepState := compare(configurations[nConfigurations-1].State, state)
				if (keepState) {
					configurations[nConfigurations-1].TimeEnd = times[ind]
				} else {
					transitionTime := listVm[i].VmInfo.BootTimeSec		//TODO: Validate booting time
					startTime :=  times[ind].Add(-1*time.Duration(transitionTime) * time.Second)		//Booting time VM
					startTime = startTime.Add(-1*time.Duration(totalServicesBootingTime) * time.Second)	//Start time containers
					state.ISODate = startTime
					configurations = append(configurations, types.Configuration{-1, state, times[ind-1],  times[ind]})
				}
			} else {
				transitionTime := listVm[i].VmInfo.BootTimeSec
				startTime := times[ind].Add(-1*time.Duration(transitionTime) * time.Second)		//Booting time VM
				startTime = startTime.Add(-1*time.Duration(totalServicesBootingTime) * time.Second)	//Start time containers
				state.ISODate = startTime
				configurations = append(configurations, types.Configuration{-1, state, startTime,  times[ind]})
			}
		}
		new_policy.Configurations = configurations
		new_policy.FinishTimeDerivation = time.Now()
		new_policy.Algorithm = policy.algorithm
		new_policy.ID = bson.NewObjectId()
		//store policy
		Store(new_policy)
		policies = append(policies, new_policy)
	}
	return policies
}




