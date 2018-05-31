package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/internal/types"
	"github.com/Cloud-Pie/SPDT/internal/rest_clients/performance_profiles"
	)

//TODO: Profile for Current config

type PolicyDerivation interface {
	CreatePolicies()
}

func Policies(forecasting types.ProcessedForecast, performance_profile performance_profiles.PerformanceProfile) []types.Policy {
	naive := NaivePolicy {forecasting, performance_profile}
	return naive.CreatePolicies()
}
