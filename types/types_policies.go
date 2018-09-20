package types

import (
	"time"
	"gopkg.in/mgo.v2/bson"
)

/*VMScale is the factor for which a type of VM is scales*/
type VMTimeRecord map[string][]time.Time

/*Service keeps the name and scale of the scaled service*/
type Service map[string]ServiceInfo

/*VMScale is the factor for which a type of VM is scales*/
type VMScale map[string]int

/*_________________________________________
		VMScale Methods
___________________________________________
*/

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

/**Function that calculates the cost of a VM Set*/
func (vmSet VMScale) Cost(mapVMProfiles map[string] VmProfile) float64{
	cost := float64(0.0)
	for k,v := range vmSet {
		cost += mapVMProfiles[k].Pricing.Price * float64(v)
	}
	return cost
}

/*Function that calculates the capacity to host service replicas for a VM Set*/
func (vmSet VMScale) ReplicasCapacity(mapVMProfiles map[string] VmProfile) int{
	totalCapacity :=0
	for k,v := range vmSet {
		totalCapacity += mapVMProfiles[k].ReplicasCapacity * v
	}
	return totalCapacity
}

/*Function that returns the total number of VMs of a VM Set*/
func (vmSet VMScale) TotalVMs() int{
	totalNVMs := 0
	for _,v := range vmSet {
		totalNVMs += v
	}
	return totalNVMs
}


/*Function that compares if two vmSets are equal*/
func (vmSet VMScale) Equal(vmSet2 VMScale) bool {
	if len(vmSet) != len(vmSet2) {
		return false
	}
	for i, v := range vmSet {
		if v != vmSet2[i] {
			return false
		}
	}
	return true
}

type ServiceInfo struct {
	Scale 	int			`json:"Replicas"`
	CPU 	float64		`json:"Cpu_cores"`
	Memory	float64		`json:"Mem_gb"`
}

/*_________________________________________
		ServiceInfo Methods
___________________________________________
*/

/*Compare if two container configurations for a given service are equal*/
func (conf1 ServiceInfo) Equal(conf2 ServiceInfo) bool {
	if conf1.Scale != conf2.Scale {
		return false
	}else if conf1.CPU != conf2.CPU {
		return false
	}else if conf1.Memory != conf2.Memory {
		return false
	}
	return true
}

/*State is the metadata of the state expected to scale to*/
type State struct {
	LaunchTime    time.Time `json:"ISODate"`
	Services      Service   `json:"Services"`
	Hash          string    `json:"Hash"`
	VMs           VMScale   `json:"VMs"`
}

type RequestCapacitySupply struct {
	IDPrediction      string                `json:"predictions_id"`
	StatesCapacity   []StateLoadCapacity	`json:"values"`
	URL				  string				`json:"url"`
}

/*Represent the number of requests for a time T*/
type StateLoadCapacity struct {
	 TimeStamp   time.Time	`json:"timestamp"`
	 Requests	float64     `json:"requests"`
}

/*_________________________________________
		State Methods
___________________________________________
*/

/*Compare if two states are equal*/
func (currentState State) Equal(candidateState State) bool {
	if len(currentState.VMs) != len(candidateState.VMs) {
		return false
	}
	for i, v := range currentState.VMs {
		if v != candidateState.VMs[i] {
			return false
		}
	}
	if len(currentState.Services) != len(candidateState.Services) {
		return false
	}
	for i, v := range currentState.Services {
		if v != candidateState.Services[i] {
			return false
		}
	}
	return true
}



type ConfigMetrics struct {
	Cost             float64 `json:"cost" bson:"cost"`
	OverProvision    float64 `json:"over_provision" bson:"over_provision"`
	UnderProvision   float64 `json:"under_provision" bson:"under_provision"`
	RequestsCapacity float64 `json:"requests_capacity" bson:"requests_capacity"`
	CPUUtilization   float64 `json:"cpu_utilization" bson:"cpu_utilization"`
	MemoryUtilization float64 `json:"mem_utilization" bson:"mem_utilization"`
	ShadowTimeSec			float64				`json:"shadow_time" bson:"shadow_time"`
}

type PolicyMetrics struct {
	Cost                          float64		`json:"cost" bson:"cost"`
	OverProvision                 float64		`json:"over_provision" bson:"over_provision"`
	UnderProvision                float64		`json:"under_provision" bson:"under_provision"`
	NumberScalingActions          int			`json:"n_scaling_actions" bson:"n_scaling_actions"`
	StartTimeDerivation           time.Time		`json:"start_derivation_time" bson:"start_derivation_time"`
	FinishTimeDerivation          time.Time		`json:"finish_derivation_time" bson:"finish_derivation_time"`
	DerivationDuration            float64       `json:"derivation_duration" bson:"derivation_duration"`
	NumberVMScalingActions        int		  	`json:"num_scale_vms" bson:"num_scale_vms"`
	NumberContainerScalingActions int		  	`json:"num_scale_containers" bson:"num_scale_containers"`
}

/*Resource configuration*/
type ScalingStep struct {
	State					State			    `json:"State" bson:"State"`
	TimeStart 				time.Time			`json:"time_start" bson:"time_start"`
	TimeEnd					time.Time			`json:"time_end" bson:"time_end"`
	Metrics					ConfigMetrics		`json:"metrics" bson:"metrics"`
	TimeStartBilling		time.Time			`json:"time_start_billing" bson:"time_start_billing"`
	TimeEndBilling			time.Time			`json:"time_end_billing" bson:"time_end_billing"`
}


//Policy Parameters
const (
	ISUNDERPROVISION = "underprovisioning-allowed"
	MAXUNDERPROVISION= "max-percentage-underprovision"
	METHOD = "scaling-method"
	ISHETEREOGENEOUS= "heterogeneous-vms-allowed"
	ISRESIZEPODS= "pods-resize-allowed"
	VMTYPES= "vm-types"

)

//Policy States
const (
	DISCARTED = "discarted"
	SCHEDULED = "scheduled"
	SELECTED = "selected"
	)

//Policy states the scaling transitions
type Policy struct {
	ID              bson.ObjectId     ` bson:"_id" json:"id"`
	Algorithm       string            `json:"algorithm" bson:"algorithm"`
	Metrics         PolicyMetrics     `json:"metrics" bson:"metrics"`
	Status          string            `json:"status" bson:"status"`
	Parameters      map[string]string `json:"parameters" bson:"parameters"`
	ScalingActions  []ScalingStep     `json:"scaling_actions" bson:"scaling_actions"`
	TimeWindowStart time.Time         `json:"window_time_start"  bson:"window_time_start"`
	TimeWindowEnd   time.Time         `json:"window_time_end"  bson:"window_time_end"`

}

//Utility struct to represent a key value object
type StructMap struct {
	Key   string		`json:"key" bson:"key"`
	Value int			`json:"value" bson:"value"`
}

/*
Structure to keep the configuration associated to a set of containers
It includes the resource limits per replica, number of replicas, bootTime of the set,
a VMSet suitable to deploy the containers set and the cost of the solution
*/
type ContainersConfig struct {
	Limits     Limit            `json:"limits" bson:"limits"`
	MSCSetting MSCSimpleSetting `json:"mscs" bson:"mscs"`
	VMSet      VMScale          `json:"vms" bson:"vms"`
	Cost       float64          `json:"cost" bson:"cost"`
}