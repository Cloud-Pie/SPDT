package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/config"
	"time"
)

//TODO: Profile for Current config

//Interface for strategies of how to scale
type PolicyDerivation interface {
	CreatePolicies(poiList []types.PoI, values []int, times [] time.Time, profile types.PerformanceProfile)
}

//Interface for strategies of when to scale
type TimeWindowDerivaton interface {
	NumberIntervals()	int
	WindowDerivation(values []int, times [] time.Time)	types.ProcessedForecast
}


func Policies(poiList []types.PoI, values []int, times [] time.Time, performance_profile types.PerformanceProfile, configuration config.SystemConfiguration, priceModel types.PriceModel) []types.Policy {
	var policies []types.Policy

	switch configuration.PreferredAlgorithm {
	case util.NAIVE_ALGORITHM:
		naive := NaivePolicy {algorithm:util.NAIVE_ALGORITHM}
		policies = naive.CreatePolicies(poiList,values,times, performance_profile)
	case util.INTEGER_PROGRAMMING_ALGORITHM:
		integer := IntegerPolicy{util.INTEGER_PROGRAMMING_ALGORITHM, priceModel}
		policies = integer.CreatePolicies(poiList,values,times, performance_profile)
	case util.SMALL_STEP_ALGORITHM:
		sstep := SStepPolicy{priceModel, util.SMALL_STEP_ALGORITHM}
		policies = sstep.CreatePolicies(poiList,values,times, performance_profile)
	default:
		naive := NaivePolicy {algorithm:util.NAIVE_ALGORITHM}
		sstep := SStepPolicy{priceModel, util.SMALL_STEP_ALGORITHM}
		policies = append(naive.CreatePolicies(poiList,values,times, performance_profile),sstep.CreatePolicies(poiList,values,times, performance_profile)...)
	}
	return policies
}

//Adjust the times that were interpolated
func adjustTime(t time.Time, factor float64) time.Time{
	f := factor*3600
	return t.Add(time.Duration(f) * time.Second)
}

//Compare two states
func compare(s1 types.State, s2 types.State) bool {
	if len(s1.VMs) != len(s2.VMs) {
		return false
	}
	for i, v := range s1.VMs {
		if v != s2.VMs[i] {
			return false
		}
	}
	if len(s1.Services) != len(s2.Services) {
		return false
	}
	for i, v := range s1.Services {
		if v != s2.Services[i] {
			return false
		}
	}
	return true
}