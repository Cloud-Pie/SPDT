package types

import (
	"time"
)

/*Service keeps the name and scale of the scaled service*/
type Service struct {
	Name  string	`json:Name`
	Scale int	`json:Scale`
}

/*State is the metadata of the state expected to scale to*/
type State struct {
	Time     time.Time     `json:Time`
	Services [] Service `json:Services`
	Name     string     `json:Name`
	Vms      [] VmScale `json:Vms`

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
	ID string
	TotalCost float64
	Configurations    [] Configuration
}

/*Critical Interval is the interval of time for which the requests
are above/below the capacity*/
type CriticalInterval struct {
	TimeStart	time.Time	`json:"TimeStart"`
	Requests	int	`json:"Requests"`	//max/min point in the interval
	Trend	int `json:"Trend"`	//1:= aboveThreshold; -1:= below
	TimeEnd	time.Time	`json:"TimeEnd"`
}

/*Forecast metadata after processing */
type Forecast struct {
	NeedToScale       bool
	CriticalIntervals [] CriticalInterval
}