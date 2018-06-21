package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"github.com/Cloud-Pie/SPDT/util"
	"gopkg.in/mgo.v2/bson"
	"math"
)

type NaivePolicy struct {
	algorithm 				string
}

func (naive NaivePolicy) CreatePolicies(poiList []types.PoI, values []int, times [] time.Time, performanceProfile types.PerformanceProfile) [] types.Policy {

	listVm := performanceProfile.PerformanceModels[0].VmProfiles; //TODO: Change according to CSP
	stateName := performanceProfile.DockerImageApp + time.Now().Format(util.TIME_LAYOUT)
	policies := []types.Policy {}

	processedForecast := naive.ProcessData(poiList,values,times)		//TODO: Fix for maintenance

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
			state.Name = stateName

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

/*Given the points of interest split the time serie into the intervals where a scaling needs to be perfomed*/
func (naive NaivePolicy)ProcessData(poiList [] types.PoI, values []int, times []time.Time) (types.ProcessedForecast) {

	intervals := []types.CriticalInterval{}

	for _,item := range poiList {
		interval := types.CriticalInterval{}
		interval.Requests = values[item.Index]
		interval.TimePeak = times[item.Index]

		interSize := len(intervals)
		if (interSize > 1){
			//start time is equal to the end time from previous interval
			interval.TimeStart = intervals[interSize-1].TimeEnd
		}else {
			interval.TimeStart = times[int(item.Start.Index)]
		}

		//Calculate End Time using the ips_right
		timeValleyIpsRight := adjustTime(times[int(item.End.Index)], item.End.Left_ips - math.Floor(item.End.Left_ips))
		if timeValleyIpsRight.After(interval.TimePeak) {
			interval.TimeEnd = timeValleyIpsRight
		} else {
			interval.TimeEnd = times[int(item.End.Index)]
		}
		interval.AboveThreshold = item.Peak
		intervals = append(intervals, interval)
	}
	processedForecast := types.ProcessedForecast{}
	processedForecast.CriticalIntervals = intervals

	return processedForecast
}
