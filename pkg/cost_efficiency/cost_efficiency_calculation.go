package cost_efficiency_calculation

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"github.com/yemramirezca/SPDT/internal/util"
)


func ComputeCost (policies [] types.Policy) [] types.Policy{
	priceModel := util.ParsePricesFile(util.PRICES_FILE)
	mapPrices := structToMap(priceModel)
	for i,policy := range policies {
		totalCost := float32(0.0)
		for _,st := range policy.States {
			for _,vm := range st.VmsScale {
				totalCost += (mapPrices [vm.Type] * float32(vm.Scale))
			}
		}
		policies[i].TotalCost = totalCost
	}

	return policies
}

func structToMap(priceModel util.PriceModel) map[string] float32 {
	mapPrices := make(map[string]float32)
	for _,vmPrice := range priceModel.VMPrices {
		mapPrices [vmPrice.VmType ] = vmPrice.Price
	}
	return mapPrices
}
