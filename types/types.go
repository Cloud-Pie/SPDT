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
	Trend	int `json:"Trend"`	//1:= aboveThreshold; -1:= below
	TimeEnd	time.Time	`json:"TimeEnd"`
	TimePeak time.Time
	DeltaT	int			//Distance between peaks
}

/*ProcessedForecast metadata after processing */
type ProcessedForecast struct {
	NeedToScale       bool
	CriticalIntervals [] CriticalInterval
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

type VmProfile struct {
	VmType   string `json:"vm_type" bson:"vm_type"`
	Trn      int    `json:"trn" bson:"trn"`
	NumCores int    `json:"num_cores" bson:"num_cores"`
	MemoryGb int    `json:"memory_gb" bson:"memory_gb"`
}

type PerformanceModel struct {
	CSP        string             `json:"CSP" bson:"CSP"`
	VmProfiles [] VmProfile `json:"VMs" bson:"VMs"`
}

type PerformanceProfile struct {
	ID          	  bson.ObjectId 	  `bson:"_id" json:"id"`
	AppType           string              `json:"app_type" bson:"app_type"`
	DockerImageApp    string              `json:"docker_image_app" bson:"docker_image_app"`
	GitUrlApp         string              `json:"git_url_app" bson:"git_url_app"`
	QoSParams         QoSParams           `json:"qos_params" bson:"qos_params"`
	PerformanceModels [] PerformanceModel `json:"perf_model" bson:"perf_model"`
}