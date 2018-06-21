package policy_evaluation

import (
	"gopkg.in/yaml.v2"
	"log"
	"io/ioutil"
	"github.com/Cloud-Pie/SPDT/types"
)

type VMPrice struct{
	VmType string	`yaml:"type"`
	Price float64	`yaml:"price"`
	Unit string	`yaml:"unit"`
}

type PriceModel struct{
	VMPrices []VMPrice	`yaml:"vm-prices"`
}

func ParsePricesFile(configFile string) (types.PriceModel,error) {
	prices := types.PriceModel{}
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
