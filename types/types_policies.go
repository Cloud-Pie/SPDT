package types

import (
	"time"
	"gopkg.in/mgo.v2/bson"
)

/*Service keeps the name and scale of the scaled service*/
type Service struct {
	Name  string	`json:Name`
	Scale int	`json:Scale`
}

/*State is the metadata of the state expected to scale to*/
type State struct {
	ISOTime  time.Time
	Services [] Service `json:Services`
	Name     string     `json:Name`
	Vms      [] VmScale `json:Vms`
	Time string 	`json:Time`

}

func  (state State) MapTypesScale() map[string] int{
	mapTypesScale := make(map[string]int)

	for _,vm := range state.Vms {
		mapTypesScale [vm.Type] = vm.Scale
	}
	return  mapTypesScale
}

/*VmScale is the factor for which a type of VM is scales*/
type VmScale struct {
	Type string `json:"Type"`
	Scale int `json:"Scale"`
}

/*Resource configuration*/
type Configuration struct {
	TransitionCost float64
	State	State
	TimeStart time.Time
	TimeEnd	time.Time
}

//Policy states the scaling transitions
type Policy struct {
	ID     bson.ObjectId 	  `bson:"_id" json:"id"`
	Algorithm 	string		  `json:"algorithm" bson:"algorithm"`
	TotalCost float64		`json:"total_cost" bson:"total_cost"`
	Configurations    [] Configuration	`json:"configuration" bson:"configuration"`
	StartTimeDerivation	time.Time	`json:"start_derivation_time" bson:"start_derivation_time"`
	FinishTimeDerivation time.Time	`json:"finish_derivation_time" bson:"finish_derivation_time"`
}

/*Critical Interval is the interval of time for which the requests
are above/below the capacity*/
type CriticalInterval struct {
	TimeStart	time.Time	`json:"TimeStart"`
	Requests	int	`json:"Requests"`	//max/min point in the interval
	AboveThreshold	bool `json:"AboveThreshold"`	//1:= aboveThreshold; -1:= below
	TimeEnd	time.Time	`json:"TimeEnd"`
	TimePeak time.Time
	DeltaT	int			//Distance between peaks
}

/*ProcessedForecast metadata after processing */
type ProcessedForecast struct {
	NeedToScale       bool
	CriticalIntervals [] CriticalInterval
	RawForecast		Forecast
}

type ForecastedValue struct {
	TimeStamp   time.Time	`json:"time-stamp"`
	Requests	int         `json:"requests"`
}

type Forecast struct {
	ID	string							`json:"id"`
	ForecastedValues []ForecastedValue	`json:"values"`
}


type QoSParams struct {
	Request_time_out_sec int `json:"request_time_out_sec" bson:"request_time_out_sec"`
	Request_availability_percent int `json:"request_availability_percent" bson:"request_availability_percent"`
}

type VmInfo struct {
	Type   			string `json:"type" bson:"type"`
	NumCores 		int    `json:"num_cores" bson:"num_cores"`
	MemoryGb 		int    `json:"memory_gb" bson:"memory_gb"`
	BootTimeSec     int    `json:"boot_time_sec" bson:"boot_time_sec"`
	OS     			string `json:"os" bson:"os"`
}

type Limit struct {
	NumCores	int		`json:"num_cores" bson:"num_cores"`
	Memory		int		`json:"memory" bson:"memory"`
}

type ServiceInfo struct {
	Name					string	`json:"name" bson:"name"`
	NumReplicas				int		`json:"num_replicas" bson:"num_replicas"`
	DockerImage				string		`json:"docker_image" bson:"docker_image"`
	ContainerStartTime		int		`json:"container_start_time_sec" bson:"container_start_time_sec"`
	Limits					Limit	`json:"limits" bson:"limits"`
}

type VmProfile struct {
	VmInfo			VmInfo			`json:"vm_info" bson:"vm_info"`
	ServiceInfo		[]ServiceInfo		`json:"services" bson:"services"`
	TRN				int 			`json:"maximum_service_capacity_per_sec" bson:"maximum_service_capacity_per_sec"`
}

type PerformanceModel struct {
	CSP        string             `json:"CSP" bson:"CSP"`
	VmProfiles [] VmProfile `json:"profiles" bson:"profiles"`
}

func  (model PerformanceModel) MapTypeCapacity() map[string] int{
	mapTypesCapacity := make(map[string]int)

	for _,vm := range model.VmProfiles {
		mapTypesCapacity [vm.VmInfo.Type] = vm.TRN
	}
	return  mapTypesCapacity
}


type PerformanceProfile struct {
	ID          	  bson.ObjectId 	  `bson:"_id" json:"id"`
	AppType           string              `json:"app_type" bson:"app_type"`
	DockerImageApp    string              `json:"docker_image_app" bson:"docker_image_app"`
	GitUrlApp         string              `json:"git_url_app" bson:"git_url_app"`
	QoSParams         QoSParams           `json:"qos_params" bson:"qos_params"`
	PerformanceModels [] PerformanceModel `json:"perf_model" bson:"perf_model"`
}

type PoI struct {
	Peak	bool	 `json:"peak"`
	Index 	int  	 `json:"index"`
	Start 	float64  `json:"start"`
	End 	float64  `json:"end"`
}

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