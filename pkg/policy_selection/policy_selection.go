package policy_selection
import (
	"github.com/yemramirezca/SPDT/internal/types"
	"github.com/yemramirezca/SPDT/pkg/cost_efficiency"
	"sort"
)



func SelectPolicy(policies [] types.Policy) types.Policy {
	policies = cost_efficiency_calculation.ComputeCost(policies)

	sort.Slice(policies, func(i, j int) bool {
		return policies[i].TotalCost < policies[j].TotalCost
	})

	return policies[0]
}


