package config

import (
	"time"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"log"
)

type Component struct {
	Endpoint string	`yaml:"endpoint"`
	Username string	`yaml:"username"`
	Password string	`yaml:"password"`
	ApiKey string	`yaml:"api-key"`
}

type ScalingHorizon struct {
	StartTime time.Time	`yaml:"start-time"`
	EndTime	time.Time	`yaml:"end-time"`
}

type PolicySettings struct{
	HorizontalEnabled	bool	`yaml:"horizontal-enabled"`
	VerticalEnabled		bool	`yaml:"vertical-enabled"`
}

type SystemConfiguration struct {
	ForecastingComponent 			Component	`yaml:"forecasting-component"`
	PerformanceProfilesComponent 	Component	`yaml:"performance-profiles-component"`
	SchedulerComponent 				Component	`yaml:"scheduler-component"`
	ScalingHorizon					ScalingHorizon `yaml:"scaling-horizon"`
	PreferredAlgorithm 				string	`yaml:"preferred-algorithm"`
	PolicySettings 					PolicySettings	`yaml:"policy-settings"`
	MonthlyBudget 					float64	`yaml:"monthly-budget"`
	PricingModel struct {
		Budget float64			`yaml:"monthly-budget"`
		PriceUnit string		`yaml:"price-unit"`
	}						`yaml:"pricing-model"`

}

func ParseConfigFile(configFile string) (SystemConfiguration, error) {
	systemConfig := SystemConfiguration{}
	source, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("error: %v", err)
		return systemConfig,err
	}
	err = yaml.Unmarshal([]byte(source), &systemConfig)
	if err != nil {
		log.Fatalf("error: %v", err)
		return systemConfig,err
	}
	return systemConfig,err
}