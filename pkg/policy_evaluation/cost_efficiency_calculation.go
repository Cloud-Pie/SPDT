package policy_evaluation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
	"github.com/Cloud-Pie/SPDT/config"
)

const (
	HOUR = "Hour"
	SECOND = "Second"
)

func ComputeTotalCost(policies [] types.Policy, sysConfig config.SystemConfiguration) [] types.Policy {
	priceModel,_ := ParsePricesFile(util.PRICES_FILE)
	mapPrices,_ :=  priceModel.MapPrices()
	for pi,policy := range policies {
		totalCost := float64(0.0)
		configurations := policy.Configurations
		for cfi,cf := range configurations {
				configurationCost := ComputeConfigurationCost (cf, sysConfig.PriceUnit, mapPrices)
				policies[pi].Configurations[cfi].Metrics.Cost = configurationCost
				totalCost += configurationCost
		}
		policies[pi].Metrics.Cost = totalCost
	}
	return policies
}

func ComputeConfigurationCost (cf types.Configuration, unit string, mapPrices map[string] float64) float64 {
	configurationCost := float64(0.0)
	deltaTime := setDeltaTime(cf.TimeStart,cf.TimeEnd,unit)
	for k,v := range cf.State.VMs {
		//transitionTime := setDeltaTime(cf.State.LaunchTime, cf.TimeStart, unit)
		transitionTime := 0.0
		configurationCost += mapPrices [k] * float64(v) * (deltaTime + transitionTime)
	}
	return  configurationCost
}

func setDeltaTime (timeStart time.Time, timeEnd time.Time, unit string) float64 {
	var delta float64
	switch unit {
		case SECOND :
			delta = timeEnd.Sub(timeStart).Seconds()
			if delta < 60 {return 60} else {return delta}		//It charges at least 60 seconds
		case HOUR:
			delta = timeEnd.Sub(timeStart).Hours()
			if delta < 1 {return 1} else {return delta}			//It charges at least 1 hour
	}
	return delta
}
