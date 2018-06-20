package types

type PriceModel struct{
	VMPrices []VMPrice	`yaml:"vm-prices"`
}

type VMPrice struct{
	VmType string	`yaml:"type"`
	Price float64	`yaml:"price"`
	Unit string	`yaml:"unit"`
}


func (priceModel PriceModel) MapPrices() (map[string] float64, string) {
	mapPrices := make(map[string]float64)
	for _,vmPrice := range priceModel.VMPrices {
		mapPrices [vmPrice.VmType ] = vmPrice.Price
	}
	return mapPrices, priceModel.VMPrices[0].Unit
}
