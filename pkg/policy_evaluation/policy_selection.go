package policy_evaluation
import (
	"github.com/Cloud-Pie/SPDT/types"
	"sort"
	"errors"
	"github.com/Cloud-Pie/SPDT/config"
)

//TODO: Include as criteria for the selection time

func SelectPolicy(policies [] types.Policy, sysConfig config.SystemConfiguration) (types.Policy, error) {

	policies = ComputeTotalCost(policies, sysConfig)

	sort.Slice(policies, func(i, j int) bool {
		return policies[i].Metrics.Cost < policies[j].Metrics.Cost
	})

	if len(policies)>0{
		return policies[0], nil
	}else {
		return types.Policy{}, errors.New("No suitable policy found")
	}

}


