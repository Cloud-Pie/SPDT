package policy_evaluation
import (
	"github.com/Cloud-Pie/SPDT/types"
	"sort"
	"errors"
	"github.com/Cloud-Pie/SPDT/config"
	misc "github.com/Cloud-Pie/SPDT/pkg/policies_derivation"
)

func SelectPolicy(policies *[]types.Policy, sysConfig config.SystemConfiguration, vmProfiles []types.VmProfile, forecast types.Forecast)(types.Policy, error) {

	mapVMProfiles := misc.VMListToMap(vmProfiles)
	//Calculate total cost of the policy
	for i := range *policies {
		(*policies)[i].Metrics.Cost = computePolicyCost((*policies)[i],sysConfig.PricingModel.BillingUnit, mapVMProfiles)
	}
	//Sort policies based on price
	sort.Slice(*policies, func(i, j int) bool {
		return (*policies)[i].Metrics.Cost < (*policies)[j].Metrics.Cost
	})

	//Calculate total cost of the policy
	for i := range *policies {
		over, under := computeMetricsCapacity(&(*policies)[i].Configurations,forecast.ForecastedValues)
		(*policies)[i].Metrics.OverProvision = over
		(*policies)[i].Metrics.UnderProvision = under
	}

	if len(*policies) >0 {
		(*policies)[0].Status = types.SELECTED
		return (*policies)[0], nil
	} else {
		return types.Policy{}, errors.New("No suitable policy found")
	}
}
