package policies_derivation

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"github.com/yemramirezca/SPDT/pkg/performance_profiles"
	)

func CreatePolicies(forecasting types.Forecast, performance_profile performance_profiles.PerformanceProfile) [] types.Policy{
	requests := 1000
	list_vm := performance_profile.Perf_models[0].VMs;

	for i := range list_vm {
		a := requests / list_vm[i].Trn
	}
	return nil
}
