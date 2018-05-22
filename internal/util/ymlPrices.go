package util

import (
	"gopkg.in/yaml.v2"
	"log"
	"io/ioutil"
)

type VMPrice struct{
	VmType string	`yaml:"type"`
	Price float32	`yaml:"price"`
	Unit string	`yaml:"unit"`
}

type PriceModel struct{
	VMPrices []VMPrice	`yaml:"vmPrices"`
}

func ParsePricesFile(configFile string) PriceModel {
	prices := PriceModel{}
	source, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal([]byte(source), &prices)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	return prices
}