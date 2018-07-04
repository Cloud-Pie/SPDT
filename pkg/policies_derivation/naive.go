package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"sort"
	"github.com/Cloud-Pie/SPDT/util"
	"strconv"
)

type NaivePolicy struct {
	algorithm  string               //Algorithm's name
	limitNVMS  int                  //Max number of vms of the same type in a cluster
	timeWindow TimeWindowDerivation //Algorithm used to process the forecasted time serie
}

func (naive NaivePolicy) CreatePolicies(currentState types.State, processedForecast types.ProcessedForecast, mapVMProfiles map[string]types.VmProfile, serviceProfile types.ServiceProfile) [] types.Policy {

	policies := []types.Policy {}
	newPolicy := types.Policy{}
	newPolicy.StartTimeDerivation = time.Now()
	configurations := []types.Configuration {}
	totalOverProvision := float32(0.0)
	totalUnderProvision := float32(0.0)

	for _, it := range processedForecast.CriticalIntervals {
		requests := it.Requests
		services := [] types.Service{}

		//Select the performance profile that fits better
		performanceProfile := selectProfile(serviceProfile.PerformanceProfiles)
		//Compute number of replicas needed depending on requests
		nProfileCopies := int(math.Ceil(float64(requests) / float64(performanceProfile.TRN)))
		nServiceReplicas := nProfileCopies * performanceProfile.NumReplicas
		services = append(services, types.Service{Name:serviceProfile.Name,Scale:nServiceReplicas})

		//Compute under/over provision
		diff := (nProfileCopies* performanceProfile.TRN) - requests
		var over, under float32
		if diff >= 0 {
			over = float32(diff*100/requests)
			under = 0
		}else{
			over = 0
			under = -1*float32(diff*100/requests)
		}

		//Set total resource limit needed
		limit := types.Limit{}
		limit.Memory = performanceProfile.Limit.Memory * float32(nServiceReplicas)
		limit.NumCores = performanceProfile.Limit.NumCores * nServiceReplicas

		//Find suitable Vm(s) depending on resources limit and current state
		vms := naive.findSuitableVMs(currentState, mapVMProfiles, limit)
		totalServicesBootingTime := performanceProfile.BootTimeSec //TODO: It should include a booting rate

		state := types.State{}
		state.Services = services
		state.VMs = vms

		//Adjust booting times for resources configuration
		nConfigurations := len(configurations)
		if nConfigurations >= 1 && state.Equal(configurations[nConfigurations-1].State) {
			configurations[nConfigurations-1].TimeEnd = it.TimeEnd
		} else {
			transitionTime := computeVMBootingTime(mapVMProfiles, vms)                            //TODO: It should include a booting rate
			startTime := it.TimeStart.Add(-1 * time.Duration(transitionTime) * time.Second)       //Booting time VM
			startTime = startTime.Add(-1 * time.Duration(totalServicesBootingTime) * time.Second) //Start time containers
			state.ISODate = startTime
			state.Name = strconv.Itoa(nConfigurations) + "__" + serviceProfile.Name + "__" + startTime.Format(util.TIME_LAYOUT)
			configurations = append(configurations,
				types.Configuration {
					State:          state,
					TimeStart:      it.TimeStart,
					TimeEnd:        it.TimeEnd,
					OverProvision:  over,
					UnderProvision: under,
				})

			totalOverProvision += over
			totalUnderProvision += under
		}
	}

	totalConfigurations := len(configurations)
	//Add new policy
	newPolicy.Configurations = configurations
	newPolicy.FinishTimeDerivation = time.Now()
	newPolicy.Algorithm = naive.algorithm
	newPolicy.ID = bson.NewObjectId()
	newPolicy.TotalOverProvision = totalOverProvision/float32(totalConfigurations)
	newPolicy.TotalUnderProvision = totalUnderProvision/float32(totalConfigurations)
	//store policy
	Store(newPolicy)
	policies = append(policies, newPolicy)
	return policies
}

func computeVMBootingTime(mapVMProfiles map[string]types.VmProfile, vmsScale []types.VmScale) int {
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

func selectProfile(performanceProfiles []types.PerformanceProfile) types.PerformanceProfile {
	//In a naive case, select the one with rank 1
	for _,p := range performanceProfiles {
		if p.RankWithLimits == 1 {
			return p
		}
	}
	return performanceProfiles[0]
}


func (naive NaivePolicy) findSuitableVMs(currentState types.State, mapVMProfiles map[string]types.VmProfile, limit types.Limit) []types.VmScale {
	vmscale := []types.VmScale{}
	vm := mapVMProfiles [currentState.VMs[0].Type] //Naive case assumes that there is only 1 VM type
	for nScale := 1; nScale < naive.limitNVMS; nScale++ {
		if limit.NumCores <= vm.NumCores*nScale && limit.Memory <= vm.MemoryGb * float32(nScale) {
			vmscale = append(vmscale, types.VmScale{Type:vm.Type, Scale:nScale})
			return vmscale
		}
	}
	return vmscale
}

func (naive NaivePolicy) findCheapestVMs(vmProfiles []types.VmProfile, limit types.Limit) []types.VmScale {
	vmscale := []types.VmScale{}
	sort.Slice(vmProfiles, func(i, j int) bool {
		return vmProfiles[i].Pricing.Price < vmProfiles[j].Pricing.Price
	})
	vms:= vmProfiles[0]		//Naive case choose the cheapest possible
	for nScale := 1; nScale < naive.limitNVMS; nScale++ {
		if limit.NumCores <= vms.NumCores*nScale && limit.Memory <= vms.MemoryGb * float32(nScale) {
			vmscale = append(vmscale, types.VmScale{Type:vms.Type, Scale:nScale})
			return vmscale
		}
	}
	return vmscale
}