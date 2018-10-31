package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"github.com/Cloud-Pie/SPDT/util"
)


//Compute the total cost for a given policy
//It takes into account the billing unit according to the pricing model
func ComputePolicyCost(policy types.Policy, billingUnit string, mapVMProfiles map[string] types.VmProfile) float64 {
	totalCost := 0.0
	for cfi,cf := range policy.ScalingActions {
		configurationCost := computeConfigurationCost(cf, billingUnit, mapVMProfiles)
		policy.ScalingActions[cfi].Metrics.Cost = math.Ceil(configurationCost*100)/100
		totalCost += configurationCost
	}
	return totalCost
}

//Compute cost for a configuration of resources
func computeConfigurationCost(cf types.ScalingAction, unit string, mapVMProfiles map[string] types.VmProfile) float64 {
	configurationCost := 0.0
	deltaTime := BilledTime(cf.TimeStart,cf.TimeEnd,unit)
	for k,v := range cf.DesiredState.VMs {
		configurationCost += mapVMProfiles [k].Pricing.Price * float64(v) * deltaTime
	}
	return  configurationCost
}

//Calculate detlta time for a time window
func BilledTime(timeStart time.Time, timeEnd time.Time, unit string) float64 {
	var delta float64
	delta = timeEnd.Sub(timeStart).Hours()
	switch unit {
	case util.SECOND :
		if delta < (0.01666666666) {return 0.01666666666} else {return delta}		//It charges at least 60 seconds
	case util.HOUR:
		return math.Ceil(delta)									//It charges at least 1 hour
	}
	return delta
}



func isEnoughBudget(monthlyBudget float64, policy types.Policy) (bool,time.Time) {
	avgHoursPerMonth := 720.0
	timeWindow := policy.TimeWindowEnd.Sub(policy.TimeWindowStart).Hours()
	numberMonths := math.Ceil(timeWindow/avgHoursPerMonth)
	var timeBudgetLimit time.Time
	if policy.Metrics.Cost <= monthlyBudget * numberMonths {
		return true, policy.TimeWindowEnd
	} else {
		spentBudget := 0.0
		for _,c := range policy.ScalingActions {
			spentBudget+= c.Metrics.Cost
			if (spentBudget >= monthlyBudget) {
				timeBudgetLimit = c.TimeStart
			}
		}
	}

	return false, timeBudgetLimit
}
