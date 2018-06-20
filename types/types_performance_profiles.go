package types

import (
	"gopkg.in/mgo.v2/bson"
)

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
