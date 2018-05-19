package policies_derivation

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"github.com/yemramirezca/SPDT/pkg/performance_profiles"
	)

//TODO: Profile for Current config

type Policy_Set_interface interface {
	CreatePolicies()
}

func CreatePolicies (forecasting types.Forecast, performance_profile performance_profiles.PerformanceProfile) []types.Policy {
	naive := NaivePolicy {forecasting, performance_profile}
	return naive.CreatePolicies()
}
