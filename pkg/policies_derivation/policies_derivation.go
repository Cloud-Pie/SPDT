package policies_derivation

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"github.com/yemramirezca/SPDT/pkg/performance_profiles"
	"github.com/yemramirezca/SPDT/pkg/forecast_processing"
	)

//TODO: Profile for Current config

type PolicyList interface {
	CreatePolicies()
}

func CreatePolicies (forecasting forecast_processing.Forecast, performance_profile performance_profiles.PerformanceProfile) []types.Policy {
	naive := NaivePolicy {forecasting, performance_profile}
	return naive.CreatePolicies()
}
