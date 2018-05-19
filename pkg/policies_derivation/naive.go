package policies_derivation

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"github.com/yemramirezca/SPDT/pkg/performance_profiles"
)

type NaivePolicy struct {
	forecasting types.Forecast
	performance_profile performance_profiles.PerformanceProfile
}

func (naive NaivePolicy) CreatePolicies() [] types.Policy{

	list_vm := naive.performance_profile.Perf_models[0].VMs;
	policies := []types.Policy {}

	for i := range list_vm {
		states := []types.State {}
		for _, it := range naive.forecasting.CriticalIntervals {
			requests := it.Requests
			n_vms := requests / list_vm[i].Trn
			services := [] types.Service{{list_vm[i].Vm_type + "_" + string(i), n_vms}}
			states = append(states, types.State{it.TimeStart,services,"unknown"})
			new_policy := types.Policy{-1, states}
			policies = append(policies, new_policy)
		}
	}
	return policies
}