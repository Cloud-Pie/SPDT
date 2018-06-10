package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/config"
)

//TODO: Profile for Current config

type PolicyDerivation interface {
	CreatePolicies()
}

func Policies(forecasting types.ProcessedForecast, performance_profile types.PerformanceProfile, configuration config.SystemConfiguration) []types.Policy {
	var policies []types.Policy

	switch configuration.PreferredAlgorithm {
	case util.NAIVE_ALGORITHM:
		naive := NaivePolicy {forecasting, performance_profile}
		policies = naive.CreatePolicies()
	case util.INTEGER_PROGRAMMING_ALGORITHM:
		integer := IntegerPolicy{forecasting, performance_profile}
		policies = integer.CreatePolicies()
	default:
		naive := NaivePolicy {forecasting, performance_profile}
		integer := IntegerPolicy{forecasting, performance_profile}
		policies = append(naive.CreatePolicies(),integer.CreatePolicies()...)
	}

	return policies
}
