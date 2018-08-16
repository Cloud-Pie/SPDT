package types

type PriceModel struct{
	VMPrices []VMPrice	`yaml:"vm-prices"`
}

type VMPrice struct{
	VmType string	`yaml:"type"`
	Price float64	`yaml:"price"`
	Unit string	`yaml:"unit"`
}
