package policy_evaluation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
)

const (
	HOUR = "hour"
	SECOND = "second"
)

//Compute the total cost for a given policy
//It takes into account the billing unit according to the pricing model
func computePolicyCost(policy types.Policy, billingUnit string, mapVMProfiles map[string] types.VmProfile) float64 {
	totalCost := 0.0
	for cfi,cf := range policy.Configurations {
		configurationCost := computeConfigurationCost(cf, billingUnit, mapVMProfiles)
		policy.Configurations[cfi].Metrics.Cost = math.Ceil(configurationCost*100)/100
		totalCost += configurationCost
	}
	return totalCost
}

//Compute cost for a configuration of resources
func computeConfigurationCost(cf types.Configuration, unit string, mapVMProfiles map[string] types.VmProfile) float64 {
	configurationCost := 0.0
	deltaTime := setDeltaTime(cf.TimeStart,cf.TimeEnd,unit)
	for k,v := range cf.State.VMs {
		configurationCost += mapVMProfiles [k].Pricing.Price * float64(v) * deltaTime
	}
	return  configurationCost
}

//Calculate detlta time for a time window
func setDeltaTime (timeStart time.Time, timeEnd time.Time, unit string) float64 {
	/*s1 := timeStart.String()
	s2 := timeEnd.String()
	fmt.Print(s1 ,"--", s2)*/
	var delta float64
	delta = timeEnd.Sub(timeStart).Hours()
	switch unit {
		case SECOND :
			if delta < (0.01666666666) {return 0.01666666666} else {return delta}		//It charges at least 60 seconds
		case HOUR:
			return math.Ceil(delta)									//It charges at least 1 hour
	}
	return delta
}

//Calculate overprovisioning and underprovisioning of a state
func computeMetricsCapacity(configurations *[]types.Configuration, forecast []types.ForecastedValue) (float64, float64){
	var avgOver float64
	var avgUnder float64
	fi := 0
	totalOver := 0.0
	totalUnder := 0.0
	numConfigurations := float64(len(*configurations))
	for i,_ := range *configurations {
		confOver := 0.0
		confUnder := 0.0
		numSamplesOver := 0.0
		numSamplesUnder := 0.0
		for  (*configurations)[i].TimeEnd.After(forecast[fi].TimeStamp){
			deltaLoad := (*configurations)[i].Metrics.CapacityTRN - forecast[fi].Requests
			if deltaLoad > 0 {
				confOver += deltaLoad*100.0/ forecast[fi].Requests
				numSamplesOver++
			} else if deltaLoad < 0 {
				confUnder += -1*deltaLoad*100.0/ forecast[fi].Requests
				numSamplesUnder++
			}
			fi++
		}
		if numSamplesUnder > 0 {
			(*configurations)[i].Metrics.UnderProvision = confUnder /numSamplesUnder
			totalUnder += confUnder /numSamplesUnder
		}
		if numSamplesOver > 0 {
			(*configurations)[i].Metrics.OverProvision = confOver /numSamplesOver
			totalOver += confOver /numSamplesOver
		}
	}
	avgOver = totalOver/numConfigurations
	avgUnder = totalUnder /numConfigurations
	return avgOver,avgUnder
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
		for _,c := range policy.Configurations {
			spentBudget+= c.Metrics.Cost
			if (spentBudget >= monthlyBudget) {
				timeBudgetLimit = c.TimeStart
			}
		}
	}

	return false, timeBudgetLimit
}
