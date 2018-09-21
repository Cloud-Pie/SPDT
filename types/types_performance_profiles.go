package types

import "gopkg.in/mgo.v2/bson"

type Pricing struct {
	Price float64	`json:"price" bson:"price"`
	Unit string		`json:"unit" bson:"unit"`
}

type VmProfile struct {
	Type             string  `json:"type" bson:"type"`
	CPUCores         float64 `json:"cpu_cores" bson:"cpu_cores"`
	Memory           float64 `json:"mem_gb" bson:"mem_gb"`
	OS               string  `json:"os" bson:"os"`
	Pricing          Pricing `json:"pricing" bson:"pricing"`
	ReplicasCapacity int	 `json:"replicas_capacity" bson:"replicas_capacity"`
}

//Times in seconds
type BootShutDownTime struct {
	BootTime			float64		`json:"BootTime" bson:"boot_time"`
	ShutDownTime		float64		`json:"ShutDownTime" bson:"shutdown_time"`
}

//Times in seconds
type InstancesBootShutdownTime struct {
	BootTime			float64		`json:"BootTime" bson:"boot_time"`
	ShutDownTime		float64		`json:"ShutDownTime" bson:"shutdown_time"`
	NumInstances		int			`json:"NumInstances" bson:"num_instances"`
}

type Limit struct {
	CPUCores         float64 `json:"Cpu_cores" bson:"cpu_cores"`
	MemoryGB         float64 `json:"Mem_gb" bson:"mem_gb"`
	RequestPerSecond int     `json:"Request_per_second" bson:"request_per_second"`
}

type PerformanceProfile struct {
	ID          bson.ObjectId      `bson:"_id" json:"id"`
	MSCSettings []MSCSimpleSetting `json:"mscs" bson:"mscs"`
	Limit       Limit              `json:"limits" bson:"limits"`
}

type ServiceProfile struct {
	ID                  bson.ObjectId        `bson:"_id" json:"id"`
	Name                string               `json:"service_name" bson:"service_name"`
	ServiceType         string               `json:"service_type" bson:"service_type"`
	PerformanceProfiles []PerformanceProfile `json:"performance_profiles" bson:"performance_profiles"`
}

type MSCSimpleSetting struct {
	Replicas            int     `json:"replicas" bson:"replicas"`
	MSCPerSecond        float64 `json:"maximum_service_capacity_per_sec" bson:"maximum_service_capacity_per_sec"`
	BootTimeSec         float64     `json:"pod_boot_time_sec" bson:"pod_boot_time_sec"`
	StandDevBootTimeSec float64 `json:"sd_pod_boot_time_ms" bson:"sd_pod_boot_time_ms"`
}

type MaxServiceCapacity struct {
	Experimental            float64	`json:"Experimental" bson:"experimental"`
	RegBruteForce           float64	`json:"RegBruteForce" bson:"reg_brute_force"`
	RegSmart            	float64	`json:"RegSmart" bson:"reg_smart"`
}

/*Struct used to parse the information received from the performance profiles API*/
type ServicePerformanceProfile struct {
	HostInstanceType string `json:"HostInstanceType"`
	ServiceName      string	`json:"ServiceName"`
	MainServiceName  string `json:"MainServiceName"`
	ServiceType      string `json:"ServiceType"`
	TestAPI          string `json:"TestAPI"`
	Profiles			[] struct {
								Limits Limit                 `json:"Limits"`
								MSCs	[]MSCCompleteSetting `json:"MSCs"`
	}`json:"Profiles"`
}

type MSCCompleteSetting struct {
	Replicas      		int     `json:"Replicas"`
	BootTimeMs         	float64 `json:"Pod_boot_time_ms"`
	StandDevBootTimeMS 	float64 `json:"Sd_Pod_boot_time_ms"`
	MSCPerSecond        MaxServiceCapacity `json:"Maximum_service_capacity_per_sec"`
	MSCPerMinute        MaxServiceCapacity `json:"Maximum_service_capacity_per_min"`
}