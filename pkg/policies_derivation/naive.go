package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"gopkg.in/mgo.v2/bson"
	"math"
	"github.com/Cloud-Pie/SPDT/util"
)

type NaivePolicy struct {
	algorithm 				string
}

func (naive NaivePolicy) CreatePolicies(poiList []types.PoI, values []int, times [] time.Time, performanceProfile types.PerformanceProfile) [] types.Policy {

	listVm := performanceProfile.PerformanceModels[0].VmProfiles; //TODO: Change according to CSP

	policies := []types.Policy {}

	timeWindows := MediumStepOverprovision{}
	timeWindows.PoIList = poiList
	processedForecast := timeWindows.WindowDerivation(values,times)

	for i := range listVm {
		new_policy := types.Policy{}
		new_policy.StartTimeDerivation = time.Now()
		configurations := []types.Configuration {}
		for _, it := range processedForecast.CriticalIntervals {
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
			state.Name = performanceProfile.DockerImageApp + time.Now().Format(util.TIME_LAYOUT)

			nConfigurations := len(configurations)
			if (nConfigurations >= 1){
				keepState := compare(configurations[nConfigurations-1].State, state)
				if (keepState) {
					configurations[nConfigurations-1].TimeEnd = it.TimeEnd
				} else {
					transitionTime := listVm[i].VmInfo.BootTimeSec		//TODO: Validate booting time
					startTime := it.TimeStart.Add(-1*time.Duration(transitionTime) * time.Second)		//Booting time VM
					startTime = startTime.Add(-1*time.Duration(totalServicesBootingTime) * time.Second)	//Start time containers
					state.ISODate = startTime
					configurations = append(configurations, types.Configuration{-1, state, it.TimeStart, it.TimeEnd})
				}
			} else {
				transitionTime := listVm[i].VmInfo.BootTimeSec
				startTime := it.TimeStart.Add(-1*time.Duration(transitionTime) * time.Second)		//Booting time VM
				startTime = startTime.Add(-1*time.Duration(totalServicesBootingTime) * time.Second)	//Start time containers
				state.ISODate = startTime
				configurations = append(configurations, types.Configuration{-1, state, it.TimeStart, it.TimeEnd})
			}
		}

		new_policy.Configurations = configurations
		new_policy.FinishTimeDerivation = time.Now()
		new_policy.Algorithm = naive.algorithm
		new_policy.ID = bson.NewObjectId()
		//store policy
		Store(new_policy)
		policies = append(policies, new_policy)
	}
	return policies
}