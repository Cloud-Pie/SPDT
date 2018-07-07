package policy_evaluation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
)

const (
	HOUR = "Hour"
	SECOND = "Second"
)

func ComputeTotalCost(policies [] types.Policy) [] types.Policy {
	priceModel,_ := ParsePricesFile(util.PRICES_FILE)
	mapPrices,unit :=  priceModel.MapPrices()
	for pi,policy := range policies {
		totalCost := float64(0.0)
		configurations := policy.Configurations
		for cfi,cf := range configurations {
				configurationCost := ComputeConfigurationCost (cf, unit, mapPrices)
				policies[pi].Configurations[cfi].TransitionCost = configurationCost
				totalCost += configurationCost
		}
		policies[pi].TotalCost = totalCost
	}
	return policies
}

func ComputeConfigurationCost (cf types.Configuration, unit string, mapPrices map[string] float64) float64 {
	configurationCost := float64(0.0)
	deltaTime := setDeltaTime(cf.TimeStart,cf.TimeEnd,unit)
	for _,vm := range cf.State.VMs {
		transitionTime := setDeltaTime(cf.State.LaunchTime, cf.TimeStart, unit)
		configurationCost += mapPrices [vm.Type] * float64(vm.Scale) * (deltaTime + transitionTime)
	}
	return  configurationCost
}

func setDeltaTime (timeStart time.Time, timeEnd time.Time, unit string) float64{
	if unit == SECOND {
		return timeEnd.Sub(timeStart).Seconds()
	}
	return timeEnd.Sub(timeStart).Hours()
}
