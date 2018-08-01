package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	db "github.com/Cloud-Pie/SPDT/storage/policies"
	"github.com/Cloud-Pie/SPDT/config"
)

type SStepRepackPolicy struct {
	algorithm 		string				 //Algorithm's name
	timeWindow 		TimeWindowDerivation //Algorithm used to process the forecasted time serie
	mapVMProfiles map[string]types.VmProfile
	sysConfiguration	config.SystemConfiguration
}

func (p SStepRepackPolicy) CreatePolicies(processedForecast types.ProcessedForecast, serviceProfile types.ServiceProfile) [] types.Policy {

	policies := []types.Policy{}
	//Compute results for cluster of each type

		newPolicy := types.Policy{}
		newPolicy.StartTimeDerivation = time.Now()
		configurations := []types.Configuration{}

		for _, it := range processedForecast.CriticalIntervals {
			requests := it.Requests
			services := make(map[string]int)

			//Select the performance profile that fits better
			performanceProfile := selectProfile(serviceProfile.PerformanceProfiles)
			//Compute number of replicas needed depending on requests
			nProfileCopies := int(math.Ceil(float64(requests) / float64(performanceProfile.TRN)))
			nServiceReplicas := nProfileCopies * performanceProfile.NumReplicas
			services[serviceProfile.Name] = nServiceReplicas

			//Find suitable Vm(s) depending on resources limit and current state
			vms := p.FindSuitableVMs(p.mapVMProfiles, nServiceReplicas, performanceProfile.Limit)

			state := types.State{}
			state.Services = services
			state.VMs = vms

			timeStart := it.TimeStart
			timeEnd := it.TimeEnd
			totalServicesBootingTime := performanceProfile.BootTimeSec
			setConfiguration(&configurations,state,timeStart,timeEnd,serviceProfile.Name, totalServicesBootingTime, p.sysConfiguration)
		}

		//Add new policy
		newPolicy.Configurations = configurations
		newPolicy.FinishTimeDerivation = time.Now()
		newPolicy.Algorithm = p.algorithm
		newPolicy.ID = bson.NewObjectId()
		newPolicy.Metrics = types.Metrics{
			NumberConfigurations: len(configurations),
		}
		//store p
		db.Store(newPolicy)
		policies = append(policies, newPolicy)
		return policies
}

func (p SStepRepackPolicy) FindSuitableVMs(mapVMProfiles map[string]types.VmProfile, nReplicas int, limit types.Limit) types.VMScale {
	vmScale :=  make(map[string]int)
	bestVmScale :=  make(map[string]int)

		for _,v := range mapVMProfiles {
			maxReplicas := maxReplicasCapacityInVM(v, limit)
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

	return bestVmScale
}

