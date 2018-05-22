package policy_selection
import (
	"github.com/yemramirezca/SPDT/internal/types"
	"github.com/yemramirezca/SPDT/pkg/cost_efficiency"
	"sort"
)



func SelectPolicy(policies [] types.Policy) types.Policy {
	policies2 := cost_efficiency_calculation.ComputeCost(policies)

	sort.Slice(policies2, func(i, j int) bool {
		return policies2[i].TotalCost < policies2[j].TotalCost
	})

	return policies2[0]
}


