package policies_derivation

import (
	"fmt"
	"time"

	"github.com/Cloud-Pie/Passa/ymlparser"

	"github.com/Cloud-Pie/SPDT/internal/types"
	"github.com/Cloud-Pie/SPDT/pkg/performance_profiles"
)

type NaivePolicy struct {
	forecasting        types.Forecast
	performanceProfile performance_profiles.PerformanceProfile
}

func (naive NaivePolicy) CreatePolicies() []types.Policy {
	fmt.Println("start derivation of policies")
	listVm := naive.performanceProfile.PerformanceModels[0].VmProfiles //TODO: Change according to CSP
	service := naive.performanceProfile.DockerImageApp
	policies := []types.Policy{}

	for i := range listVm {
		configurations := []types.Configuration{}
		for _, it := range naive.forecasting.CriticalIntervals {
			requests := it.Requests
			n_vms := requests / listVm[i].Trn
			services := []ymlparser.Service{{Name: service, Scale: n_vms}} //TODO: Change according to # Services

			vms := []ymlparser.VM{{Type: listVm[i].VmType, Scale: n_vms}}
			transitionTime := -10 * time.Minute //TODO: Calculate booting time
			startTime := it.TimeStart.Add(transitionTime)
			state := ymlparser.State{
				ISODate:  startTime,
				Services: services,
				Name:     "unknown",
				VMs:      vms,
			}
			configurations = append(configurations, types.Configuration{TransitionCost: -1, State: state, TimeStart: it.TimeStart, TimeEnd: it.TimeEnd})

		}
		new_policy := types.Policy{"naive", -1, configurations}
		policies = append(policies, new_policy)
	}
	return policies
}
