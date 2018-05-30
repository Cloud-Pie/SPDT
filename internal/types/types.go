package types

import (
	"time"

	"github.com/Cloud-Pie/Passa/ymlparser"
)

func MapTypesScale(state ymlparser.State) map[string]int {
	mapTypesScale := make(map[string]int)

	for _, vm := range state.VMs {
		mapTypesScale[vm.Type] = vm.Scale
	}
	return mapTypesScale
}

/*Resource configuration*/
type Configuration struct {
	TransitionCost float64
	State          ymlparser.State
	TimeStart      time.Time
	TimeEnd        time.Time
}

//Policy states the scaling transitions
type Policy struct {
	ID             string
	TotalCost      float64
	Configurations []Configuration
}

/*Critical Interval is the interval of time for which the requests
are above/below the capacity*/
type CriticalInterval struct {
	TimeStart time.Time `json:"TimeStart"`
	Requests  int       `json:"Requests"` //max/min point in the interval
	Trend     int       `json:"Trend"`    //1:= aboveThreshold; -1:= below
	TimeEnd   time.Time `json:"TimeEnd"`
}

/*Forecast metadata after processing */
type Forecast struct {
	NeedToScale       bool
	CriticalIntervals []CriticalInterval
}
