package config

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

//Parameters of the scaling policies
//Scaling Method have three valid values: horizonzal, vertical, hybrid
//The parameter HeterogeneosAllowed is only used if Scaling Method is Horizonzal or Hybrid
//Max Underprovision is only used if Underprovisioning is allowed
type PolicySettings struct{
	ScalingMethod            string  `yaml:"scaling-method"`
	HetereogeneousAllowed    bool    `yaml:"heterogeneous-vms-allowed"`
	UnderprovisioningAllowed bool    `yaml:"underprovisioning-allowed"`
	MaxUnderprovision        float64 `yaml:"max-percentage-underprovision"`
	PodsResizeAllowed        bool    `yaml:"pods-resize-allowed"`
}

//Struct that models the system configuration to derive the scaling policies
type SystemConfiguration struct {
	CSP 							string				`yaml:"CSP"`
	ServiceName						string				`yaml:"service-name"`
	ServiceType						string				`yaml:"service-type"`
	PricingModel					PricingModel  	    `yaml:"pricing-model"`
	ForecastingComponent 			Component			`yaml:"forecasting-component"`
	PerformanceProfilesComponent 	Component			`yaml:"performance-profiles-component"`
	SchedulerComponent 				Component			`yaml:"scheduler-component"`
	ScalingHorizon					ScalingHorizon 		`yaml:"scaling-horizon"`
	PreferredAlgorithm 				string				`yaml:"preferred-algorithm"`
	PolicySettings 					PolicySettings		`yaml:"policy-settings"`
						}

//Method that parses the configuration file into a struct type
func ParseConfigFile(configFile string) (SystemConfiguration, error) {
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