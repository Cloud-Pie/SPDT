package policy_evaluation
import (
	"github.com/Cloud-Pie/SPDT/types"
	"sort"
)

//TODO: Include as criteria for the selection time

func SelectPolicy(policies [] types.Policy) types.Policy {

	policies = ComputeTotalCost(policies)

	sort.Slice(policies, func(i, j int) bool {
		return policies[i].TotalCost < policies[j].TotalCost
	})

	return policies[0]
}


