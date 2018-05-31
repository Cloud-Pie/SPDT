package cost_efficiency_calculation

import (
	"gopkg.in/yaml.v2"
	"log"
	"io/ioutil"
)

type VMPrice struct{
	VmType string	`yaml:"type"`
	Price float64	`yaml:"price"`
	Unit string	`yaml:"unit"`
}

type PriceModel struct{
	VMPrices []VMPrice	`yaml:"vm-prices"`
}

func ParsePricesFile(configFile string) (PriceModel,error) {
	prices := PriceModel{}
	source, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal([]byte(source), &prices)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return prices,err
}

func (priceModel PriceModel) MapPrices() (map[string] float64, string) {
	mapPrices := make(map[string]float64)
	for _,vmPrice := range priceModel.VMPrices {
		mapPrices [vmPrice.VmType ] = vmPrice.Price
	}
	return mapPrices, priceModel.VMPrices[0].Unit
}