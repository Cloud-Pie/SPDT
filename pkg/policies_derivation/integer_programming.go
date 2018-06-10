package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"github.com/Cloud-Pie/SPDT/util"
	"gopkg.in/mgo.v2/bson"
)

type IntegerPolicy struct {
	forecasting        types.ProcessedForecast
	performanceProfile types.PerformanceProfile
}

func (integer IntegerPolicy) CreatePolicies() [] types.Policy {
	listVm := integer.performanceProfile.PerformanceModels[0].VmProfiles; //TODO: Change according to CSP
	service := integer.performanceProfile.DockerImageApp
	policies := []types.Policy {}

	for i := range listVm {
		new_policy := types.Policy{}
		new_policy.StartTimeDerivation = time.Now()
		configurations := []types.Configuration {}
		for _, it := range integer.forecasting.CriticalIntervals {
			requests := it.Requests
			n_vms := requests / listVm[i].Trn
			services := [] types.Service{{ service, n_vms}} //TODO: Change according to # Services
			vms := [] types.VmScale {{listVm[i].VmType, n_vms}}
			transitionTime := -10*time.Minute		//TODO: Calculate booting time
			startTime := it.TimeStart.Add(transitionTime)
			state :=  types.State{startTime,services,"unknown", vms, startTime.Format(util.TIME_LAYOUT)}
			configurations = append(configurations, types.Configuration{-1, state, it.TimeStart, it.TimeEnd})

		}
		new_policy.Configurations = configurations
		new_policy.FinishTimeDerivation = time.Now()
		new_policy.Algorithm = util.INTEGER_PROGRAMMING_ALGORITHM
		new_policy.ID = bson.NewObjectId()
		//store policy
		Store(new_policy)
		policies = append(policies, new_policy)
	}
	return policies
}

