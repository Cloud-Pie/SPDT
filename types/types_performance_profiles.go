package types

import "gopkg.in/mgo.v2/bson"

type QoSParams struct {
	Request_time_out_sec int `json:"request_time_out_sec" bson:"request_time_out_sec"`
	Request_availability_percent int `json:"request_availability_percent" bson:"request_availability_percent"`
}

type Pricing struct {
	Price float64
	Unit string
}

type VmProfile struct {
	Type               string  `json:"type" bson:"type"`
	NumCores           float64 `json:"num_cores" bson:"num_cores"`
	Memory             float64 `json:"mem_gb" bson:"mem_gb"`
	BootTimeSec        int     `json:"boot_time_sec" bson:"boot_time_sec"`
	OS                 string  `json:"os" bson:"os"`
	Pricing            Pricing
	TerminationTimeSec int     `json:"term_time_sec" bson:"term_time_sec"`
}


type Limit struct {
	NumCores	float64			`json:"num_cores" bson:"num_cores"`
	Memory		float64		`json:"mem_gb" bson:"mem_gb"`
	RequestPerSecond	int `json:"request_per_sercond" bson:"request_per_sercond"`
}

type PerformanceProfile struct {
	NumReplicas 		int   `json:"replicas" bson:"replicas"`
	BootTimeSec 		int   `json:"pod_boot_time_sec" bson:"pod_boot_time_sec"`
	Limit       		Limit `json:"resources_limit" bson:"resources_limit"`
	TRN         		int   `json:"maximum_service_capacity_per_sec" bson:"maximum_service_capacity_per_sec"`
	RankWithLimits		int   `json:"profiles_with_limits_rank" bson:"profiles_with_limits_rank"`
}

//DEPRECATED
/*type VmProfile struct {
	VmProfile			VmProfile                `json:"vm_info" bson:"vm_info"`
	ServiceInfo		[]PerformanceProfile `json:"services" bson:"services"`
	TRN				int                  `json:"maximum_service_capacity_per_sec" bson:"maximum_service_capacity_per_sec"`
}

type PerformanceModel struct {
	CSP        string             `json:"CSP" bson:"CSP"`
	//VmProfiles [] VmProfile `json:"profiles" bson:"profiles"`
}

func  (model PerformanceModel) MapTypeCapacity() map[string] int{
	mapTypesCapacity := make(map[string]int)

	for _,vm := range model.VmProfiles {
		mapTypesCapacity [vm.VmInfo.Type] = vm.TRN
	}
	return  mapTypesCapacity
}
*/

type ServiceProfile struct {
	ID                		bson.ObjectId       `bson:"_id" json:"id"`
	Name					string				`json:"name" bson:"name"`
	AppType          		string              `json:"app_type" bson:"app_type"`
	DockerImage      		string              `json:"docker_image" bson:"docker_image"`
	//GitUrlApp         string             		`json:"git_url_app" bson:"git_url_app"`
	//QoSParams         QoSParams           	`json:"qos_params" bson:"qos_params"`
	PerformanceProfiles []PerformanceProfile 	`json:"perf_profiles_with_limits" bson:"perf_profiles_with_limits"`
}
