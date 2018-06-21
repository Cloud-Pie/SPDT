package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"gopkg.in/mgo.v2/bson"
	"fmt"
)

type SStepPolicy struct {
	priceModel types.PriceModel
	algorithm string
}

func (policy SStepPolicy) CreatePolicies(poiList []types.PoI, values []int, times [] time.Time, performanceProfile types.PerformanceProfile) [] types.Policy {
	listVm := performanceProfile.PerformanceModels[0].VmProfiles; //TODO: Change according to CSP
	service := performanceProfile.DockerImageApp
	policies := []types.Policy {}

	new_policy := types.Policy{}
	new_policy.StartTimeDerivation = time.Now()

	mapTypeCapacity := performanceProfile.PerformanceModels[0].MapTypeCapacity()
	mapTypePrice,_ := policy.priceModel.MapPrices()
	listPricesTrn(mapTypeCapacity,mapTypePrice )

	for i := range listVm {
		new_policy := types.Policy{}
		new_policy.StartTimeDerivation = time.Now()
		configurations := []types.Configuration {}

		processedForecast := types.ProcessedForecast{}

		for _, it := range processedForecast.CriticalIntervals {
			requests := it.Requests
			n_vms := requests / listVm[i].TRN
			services := [] types.Service{{ service, n_vms}} //TODO: Change according to # Services
			vms := [] types.VmScale {{listVm[i].VmInfo.Type, n_vms}}
			transitionTime := -10*time.Minute		//TODO: Calculate booting time
			startTime := it.TimeStart.Add(transitionTime)
			state :=  types.State{startTime,services,"unknown", vms}
			configurations = append(configurations, types.Configuration{-1, state, it.TimeStart, it.TimeEnd})

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

func listPricesTrn(mapTypeCapacity map[string] int, mapTypePrice map[string] float64) ([]float64, []float64) {
	prices := [] float64{}
	trnList := [] float64{}

	for k, v := range mapTypeCapacity {
		fmt.Println("k:", k, "v:", v)
		prices = append(prices, mapTypePrice[k])
		trnList = append(trnList, float64(mapTypeCapacity[k]))
	}

	return  prices, trnList
}


