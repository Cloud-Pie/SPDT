package policy_evaluation
import (
	"github.com/Cloud-Pie/SPDT/types"
	"sort"
	"errors"
	"github.com/Cloud-Pie/SPDT/config"
)

//TODO: Include as criteria for the selection time

func SelectPolicy(policies [] types.Policy, sysConfig config.SystemConfiguration, vmProfiles []types.VmProfile, serviceProfiles types.ServiceProfile)(types.Policy, error) {

	policies = ComputeTotalCost(policies, sysConfig, vmProfiles)

	sort.Slice(policies, func(i, j int) bool {
		return policies[i].Metrics.Cost < policies[j].Metrics.Cost
	})
	replicasPerProfile := selectProfile(serviceProfiles.PerformanceProfiles)
	for i,_ := range policies {
		SetMetrics(&policies[i].Configurations,replicasPerProfile.NumReplicas, replicasPerProfile.TRN)
	}

	if len(policies)>0{
		return policies[0], nil
	}else {
		return types.Policy{}, errors.New("No suitable policy found")
	}
}

func SetMetrics(configurations *[]types.Configuration, replicasPerProfile int,serviceProfileTRN float64){
	for _,c := range *configurations {
		computeConfigurationMetrics(&c,replicasPerProfile,serviceProfileTRN)
	}
}

func computeConfigurationMetrics(resources *types.Configuration, replicasPerProfile int,serviceProfileTRN float64){
	var numberReplicas int
	for _,v := range resources.State.Services {
		numberReplicas = v.Scale
	}
	requestsCapacity := float64(numberReplicas/replicasPerProfile) * serviceProfileTRN
	resources.Metrics.CapacityTRN = requestsCapacity
}


func selectProfile(performanceProfiles []types.PerformanceProfile) types.PerformanceProfile{
	//select the one with rank 1
	for _,p := range performanceProfiles {
		if p.RankWithLimits == 1 {
			return p
		}
	}
	return performanceProfiles[0]
}