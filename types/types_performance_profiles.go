package types

import "gopkg.in/mgo.v2/bson"

type Pricing struct {
	Price float64	`json:"price" bson:"price"`
	Unit string		`json:"unit" bson:"unit"`
}

type VmProfile struct {
	Type               string  `json:"type" bson:"type"`
	NumCores           float64 `json:"num_cores" bson:"num_cores"`
	Memory             float64 `json:"mem_gb" bson:"mem_gb"`
	OS                 string  `json:"os" bson:"os"`
	Pricing            Pricing `json:"pricing" bson:"pricing"`
	ReplicasCapacity   int
}

//Times in seconds
type BootShutDownTime struct {
	BootTime			int		`json:"BootTime" bson:"boot_time"`
	ShutDownTime		int		`json:"ShutDownTime" bson:"shutdown_time"`
	NumInstances		int		`json:"NumInstances" bson:"num_instances"`
}

type Limit struct {
	NumberCores      float64 `json:"cpu_cores" bson:"cpu_cores"`
	MemoryGB         float64 `json:"mem_gb" bson:"mem_gb"`
	RequestPerSecond int     `json:"request_per_second" bson:"request_per_second"`
}

type PerformanceProfile struct {
	TRNConfiguration    []TRNConfiguration 		`json:"trns" bson:"trns"`
	Limit          		Limit   				`json:"limits" bson:"limits"`
}

type ServiceProfile struct {
	ID                		bson.ObjectId       `bson:"_id" json:"id"`
	Name					string				`json:"service_name" bson:"service_name"`
	AppType          		string              `json:"app_type" bson:"app_type"`
	PerformanceProfiles 	[]PerformanceProfile 	`json:"profiles" bson:"profiles"`
}

type TRNConfiguration struct{
	NumberReplicas int     `json:"replicas" bson:"replicas"`
	TRN            float64 `json:"maximum_service_capacity_per_sec" bson:"maximum_service_capacity_per_sec"`
	BootTimeSec    int     `json:"pod_boot_time_sec" bson:"pod_boot_time_sec"`
}

