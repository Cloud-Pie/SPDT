package policies_derivation

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"github.com/yemramirezca/SPDT/pkg/performance_profiles"
)

type NaivePolicy struct {
	forecasting        types.Forecast
	performanceProfile performance_profiles.PerformanceProfile
}

func (naive NaivePolicy) CreatePolicies() [] types.Policy {
	listVm := naive.performanceProfile.PerformanceModels[0].VmProfiles; //TODO: Change according to CSP
	service := naive.performanceProfile.DockerImageApp
	policies := []types.Policy {}

	for i := range listVm {
		states := []types.State {}
		for _, it := range naive.forecasting.CriticalIntervals {
			requests := it.Requests
			n_vms := requests / listVm[i].Trn
			services := [] types.Service{{ service, n_vms}} //TODO: Change according to # Services
			vms := [] types.VmScale {{listVm[i].VmType, n_vms}}
			states = append(states, types.State{it.TimeStart,services,"unknown", vms})
			new_policy := types.Policy{"naive", -1, states}
			policies = append(policies, new_policy)
		}
	}
	return policies
}
