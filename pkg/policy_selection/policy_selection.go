package policy_selection
import (
	"github.com/Cloud-Pie/SPDT/internal/types"
	"github.com/Cloud-Pie/SPDT/pkg/cost_efficiency"
	"sort"
)

//TODO: Include as criteria for the selection time

func SelectPolicy(policies [] types.Policy) types.Policy {

	policies = cost_efficiency_calculation.ComputeTotalCost(policies)

	sort.Slice(policies, func(i, j int) bool {
		return policies[i].TotalCost < policies[j].TotalCost
	})

	return policies[0]
}


