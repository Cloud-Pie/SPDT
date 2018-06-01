package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/internal/types"
	)

//TODO: Profile for Current config

type PolicyDerivation interface {
	CreatePolicies()
}

func Policies(forecasting types.ProcessedForecast, performance_profile types.PerformanceProfile) []types.Policy {
	naive := NaivePolicy {forecasting, performance_profile}
	return naive.CreatePolicies()
}
