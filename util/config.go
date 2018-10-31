package util

import (
	"time"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"log"
)

//Struct that models the external components to which SPDT should be connected
type Component struct {
	Endpoint string	`yaml:"endpoint"`
	Username string	`yaml:"username"`
	Password string	`yaml:"password"`
	ApiKey string	`yaml:"api-key"`
}

//Struct that models the external components to which SPDT should be connected
type ForecastComponent struct {
	Endpoint string	`yaml:"endpoint"`
	Granularity string	`yaml:"granularity"`
}

//The future timespan for which the autoscaling policy is derived
type ScalingHorizon struct {
	StartTime time.Time	`yaml:"start-time"`
	EndTime	time.Time	`yaml:"end-time"`
}

//
type PricingModel struct {
	Budget      float64 `yaml:"monthly-budget"`
	BillingUnit string  `yaml:"billing-unit"`
}

type PolicySettings struct{
	ScalingMethod            string  `yaml:"vm-scaling-method"`
	PreferredMetric        string    `yaml:"preferred-metric"`
}

//Struct that models the system configuration to derive the scaling policies
type SystemConfiguration struct {
	Host 						 string			   `yaml:"host"`
	CSP                          string            `yaml:"CSP"`
	Region                       string            `yaml:"region"`
	AppName                      string            `yaml:"app-name"`
	MainServiceName              string            `yaml:"main-service-name"`
	AppType                      string            `yaml:"app-type"`
	PricingModel                 PricingModel      `yaml:"pricing-model"`
	ForecastComponent            ForecastComponent `yaml:"forecasting-component"`
	PerformanceProfilesComponent Component         `yaml:"performance-profiles-component"`
	SchedulerComponent           Component         `yaml:"scheduler-component"`
	ScalingHorizon               ScalingHorizon    `yaml:"scaling-horizon"`
	PreferredAlgorithm           string            `yaml:"preferred-algorithm"`
	PolicySettings               PolicySettings    `yaml:"policy-settings"`
	PullingInterval              int               `yaml:"pulling-interval"`
	StorageInterval              string            `yaml:"storage-interval"`
}

//Method that parses the configuration file into a struct type
func ReadConfigFile(configFile string) (SystemConfiguration, error) {
	systemConfig := SystemConfiguration{}
	source, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("There was a problem reading the configuration file: %v", err)
		return systemConfig,err
	}
	err = yaml.Unmarshal([]byte(source), &systemConfig)
	if err != nil {
		log.Fatalf("There was a problem parsing the configuration file. Please review the parameters: %v", err)
		return systemConfig,err
	}
	return systemConfig,err
}