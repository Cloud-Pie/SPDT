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
	LaunchTime time.Time  `json:ISODate`
	Services   [] Service `json:Services`
	Name       string     `json:Name`
	VMs        [] VmScale `json:VMs`
}

func (state State) MapTypesScale() map[string] int {
	mapTypesScale := make(map[string]int)
	for _,vm := range state.VMs {
		mapTypesScale [vm.Type] = vm.Scale
	}
	return  mapTypesScale
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

/*VmScale is the factor for which a type of VM is scales*/
type VmScale struct {
	Type string `json:"Type"`
	Scale int `json:"Scale"`
}

/*Resource configuration*/
type Configuration struct {
	TransitionCost 			float64
	State					State
	TimeStart 				time.Time
	TimeEnd					time.Time
	OverProvision			float32
	UnderProvision 			float32
}

//Policy states the scaling transitions
type Policy struct {
	ID     					bson.ObjectId 	  `bson:"_id" json:"id"`
	Algorithm 				string		  	  `json:"algorithm" bson:"algorithm"`
	TotalCost 				float64			  `json:"total_cost" bson:"total_cost"`
	Configurations    		[]Configuration	  `json:"configuration" bson:"configuration"`
	StartTimeDerivation		time.Time		  `json:"start_derivation_time" bson:"start_derivation_time"`
	FinishTimeDerivation 	time.Time		  `json:"finish_derivation_time" bson:"finish_derivation_time"`
	TotalOverProvision 		float32				  `json:"total_overprovision" bson:"total_overprovision"`
	TotalUnderProvision 	float32				  `json:"total_underprovision" bson:"total_underprovision"`
}