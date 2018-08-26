package types

import (
	"time"
	"gopkg.in/mgo.v2/bson"
)

/*Service keeps the name and scale of the scaled service*/
type Service map[string]ServiceInfo

/*VMScale is the factor for which a type of VM is scales*/
type VMScale map[string]int

/*Function that merges two VM sets*/
func (vmSetTarget VMScale) Merge(vmSource VMScale){
	for k,v :=  range  vmSource {
		if _,ok := vmSetTarget[k]; ok {
			vmSetTarget[k] += v
		}else {
			vmSetTarget[k] = v
		}
	}
}

/**/
func (vmSet VMScale) Cost(mapVMProfiles map[string] VmProfile) float64{
	cost := float64(0.0)
	for k,v := range vmSet {
		cost += mapVMProfiles[k].Pricing.Price * float64(v)
	}
	return cost
}

/**/
func (vmSet VMScale) ReplicasCapacity(mapVMProfiles map[string] VmProfile) int{
	totalCapacity :=0
	for k,v := range vmSet {
		totalCapacity += mapVMProfiles[k].ReplicasCapacity * v
	}
	return totalCapacity
}

/**/
func (vmSet VMScale) TotalVMs() int{
	totalNVMs := 0
	for _,v := range vmSet {
		totalNVMs += v
	}
	return totalNVMs
}

type ServiceInfo struct {
	Scale 	int			`json:Scale`
	CPU 	float64		`json:CPU`
	Memory	float64		`json:MemoryGB`
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
type ConfigMetrics struct {
	Cost 					float64
	OverProvision			float64
	UnderProvision			float64
	CapacityTRN				float64
}

type PolicyMetrics struct {
	Cost 					float64
	OverProvision			float64
	UnderProvision			float64
	NumberConfigurations	int
	StartTimeDerivation		time.Time		  `json:"start_derivation_time" bson:"start_derivation_time"`
	FinishTimeDerivation 	time.Time		  `json:"finish_derivation_time" bson:"finish_derivation_time"`
}

/*Resource configuration*/
type Configuration struct {
	State					State
	TimeStart 				time.Time
	TimeEnd					time.Time
	Metrics					ConfigMetrics
}


//Policy Parameters
const (
	ISUNDERPROVISION = "underprovisioning-allowed"
	MAXUNDERPROVISION= "max-percentage-underprovision"
	METHOD = "scaling-method"
	ISHETEREOGENEOUS= "heterogeneous-vms-allowed"

)

//Policy States
const (
	DISCARTED = "discarted"
	SCHEDULED = "scheduled"
	SELECTED = "selected"
	)

//Policy states the scaling transitions
type Policy struct {
	ID     					bson.ObjectId 	  `bson:"_id" json:"id"`
	Algorithm 				string		  	  `json:"algorithm" bson:"algorithm"`
	Metrics					PolicyMetrics	  `json:"metrics" bson:"metrics"`
	Status					string
	Parameters				map[string]string
	Configurations    		[]Configuration	  `json:"configuration" bson:"configuration"`
	TimeWindowStart   		time.Time		  `json:"window_time_start"  bson:"window_time_start"`
	TimeWindowEnd   		time.Time		  `json:"window_time_end"  bson:"window_time_end"`

}

type StructMap struct {
	Key   string
	Value int
}