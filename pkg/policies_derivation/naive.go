package policies_derivation

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"github.com/yemramirezca/SPDT/pkg/performance_profiles"
	"time"
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
		configurations := []types.Configuration {}
		for _, it := range naive.forecasting.CriticalIntervals {
			requests := it.Requests
			n_vms := requests / listVm[i].Trn
			services := [] types.Service{{ service, n_vms}} //TODO: Change according to # Services
			vms := [] types.VmScale {{listVm[i].VmType, n_vms}}
			transitionTime := -10*time.Minute		//TODO: Calculate booting time
			state :=  types.State{it.TimeStart.Add(transitionTime),services,"unknown", vms}
			configurations = append(configurations, types.Configuration{-1, state, it.TimeStart, it.TimeEnd})

		}
		new_policy := types.Policy{"naive", -1, configurations}
		policies = append(policies, new_policy)
	}
	return policies
}
