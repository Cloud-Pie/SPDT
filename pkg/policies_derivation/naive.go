package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"fmt"
	"github.com/Cloud-Pie/SPDT/util"
)

type NaivePolicy struct {
	forecasting        types.ProcessedForecast
	performanceProfile types.PerformanceProfile
}

func (naive NaivePolicy) CreatePolicies() [] types.Policy {
	fmt.Println("start derivation of policies")
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
			startTime := it.TimeStart.Add(transitionTime)
			state :=  types.State{startTime,services,"unknown", vms, startTime.Format(util.TIME_LAYOUT)}
			configurations = append(configurations, types.Configuration{-1, state, it.TimeStart, it.TimeEnd})

		}
		new_policy := types.Policy{"naive", -1, configurations}
		policies = append(policies, new_policy)
	}
	return policies
}
