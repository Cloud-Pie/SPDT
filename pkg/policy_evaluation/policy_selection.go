package policy_evaluation
import (
	"github.com/Cloud-Pie/SPDT/types"
	"sort"
	"errors"
)

//TODO: Include as criteria for the selection time

func SelectPolicy(policies [] types.Policy) (types.Policy, error) {

	policies = ComputeTotalCost(policies)

	sort.Slice(policies, func(i, j int) bool {
		return policies[i].TotalCost < policies[j].TotalCost
	})

	if len(policies)>0{
		return policies[0], nil
	}else {
		return types.Policy{}, errors.New("No suitable policy found")
	}

}


