package types

import (
	"time"
	"gopkg.in/mgo.v2/bson"
)

/*Service keeps the name and scale of the scaled service*/
type Service map[string]ServiceInfo

/*VMScale is the factor for which a type of VM is scales*/
type VMScale map[string]int

type ServiceInfo struct {
	Scale 	int			`json:Scale`
	CPU 	float64		`json:CPU`
	Memory	float64		`json:Memory`
}

/*State is the metadata of the state expected to scale to*/
type State struct {
	LaunchTime time.Time `json:ISODate`
	Services   Service   `json:Services`
	Name       string    `json:Name`
	VMs        VMScale   `json:VMs`
}


//Compare if two states are equal
func (state State) Equal(s2 State) bool {
	if len(state.VMs) != len(s2.VMs) {
		return false
	}
	for i, v := range state.VMs {
		if v != s2.VMs[i] {
			return false
		}
	}
	if len(state.Services) != len(s2.Services) {
		return false
	}
	for i, v := range state.Services {
		if v != s2.Services[i] {
			return false
		}
	}
	return true
}
type Metrics struct {
	Cost 					float64
	OverProvision			float32
	UnderProvision			float32
	CapacityTRN				float64
	NumberConfigurations	int
}

/*Resource configuration*/
type Configuration struct {
	State					State
	TimeStart 				time.Time
	TimeEnd					time.Time
	Metrics					Metrics
}

//Policy states the scaling transitions
type Policy struct {
	ID     					bson.ObjectId 	  `bson:"_id" json:"id"`
	Algorithm 				string		  	  `json:"algorithm" bson:"algorithm"`
	Metrics					Metrics			  `json:"metrics" bson:"metrics"`
	StartTimeDerivation		time.Time		  `json:"start_derivation_time" bson:"start_derivation_time"`
	FinishTimeDerivation 	time.Time		  `json:"finish_derivation_time" bson:"finish_derivation_time"`
	Configurations    		[]Configuration	  `json:"configuration" bson:"configuration"`
	TimeWindowStart   		time.Time		  `json:"window_time_start"  bson:"window_time_start"`
	TimeWindowEnd   		time.Time		  `json:"window_time_end"  bson:"window_time_end"`

}

type StructMap struct {
	Key   string
	Value int
}