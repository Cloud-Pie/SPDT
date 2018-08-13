package policy_evaluation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"github.com/Cloud-Pie/SPDT/config"
)

const (
	HOUR = "Hour"
	SECOND = "Second"
)

func ComputeTotalCost(policies [] types.Policy, sysConfig config.SystemConfiguration, vmProfiles []types.VmProfile) [] types.Policy {
	mapVMProfiles := make(map[string]types.VmProfile)
	for _,p := range vmProfiles {
		mapVMProfiles[p.Type] = p
	}

	for pi,policy := range policies {
		totalCost := float64(0.0)
		configurations := policy.Configurations
		for cfi,cf := range configurations {
				configurationCost := ComputeConfigurationCost (cf, sysConfig.PricingModel.BillingUnit, mapVMProfiles)
				policies[pi].Configurations[cfi].Metrics.Cost = configurationCost
				totalCost += configurationCost
		}
		policies[pi].Metrics.Cost = totalCost
	}
	return policies
}

func ComputeConfigurationCost (cf types.Configuration, unit string, mapVMProfiles map[string] types.VmProfile) float64 {
	configurationCost := float64(0.0)
	deltaTime := setDeltaTime(cf.TimeStart,cf.TimeEnd,unit)
	for k,v := range cf.State.VMs {
		//transitionTime := setDeltaTime(cf.State.LaunchTime, cf.TimeStart, unit)
		transitionTime := 0.0
		configurationCost += mapVMProfiles [k].Pricing.Price * float64(v) * (deltaTime + transitionTime)
	}
	return  configurationCost
}

func setDeltaTime (timeStart time.Time, timeEnd time.Time, unit string) float64 {
	var delta float64
	delta = timeEnd.Sub(timeStart).Hours()
	switch unit {
		case SECOND :
			if delta < (0.01666666666) {return 0.01666666666} else {return delta}		//It charges at least 60 seconds
		case HOUR:
			if delta < 1 {return 1} else {return delta}									//It charges at least 1 hour
	}
	return delta
}
