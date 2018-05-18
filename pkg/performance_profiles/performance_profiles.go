package performance_profiles

import (
	"github.com/yemramirezca/SPDT/internal/util"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"fmt"
)

type VM struct {
	Vm_type string `json:"vm_type"`
	Trn int `json:"trn"`
	Num_cores int `json:"num_cores"`
	Memory_gb int `json:"memory_gb"`
}

type QoS_Params struct {
	Request_time_out_sec int `json:"request_time_out_sec"`
	Request_availability_percent int `json:"request_availability_percent"`
}

type Perf_model struct {
	CSP string `json:"CSP"`
	VMs []VM `json:"VMs"`
}

type PerformanceProfile struct {
	App_type string `json:"app_type"`
	Docker_image_app string `json:"docker_image_app"`
	Git_url_app string `json:"git_url_app"`
	QoS_Params QoS_Params `json:"qos_params"`
	Perf_models [] Perf_model `json:perf_model`
}

func GetPerformanceProfiles () PerformanceProfile {
	fmt.Println("start profiles")

	var performanceProfile PerformanceProfile
	response, err := http.Get(util.URL_PROFILER)
	if err != nil {
		fmt.Printf("The profiler request failed with error %s\n", err)
		panic(err)
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	json.Unmarshal(data, &performanceProfile)
	fmt.Println("Successfully reading JSON file for App type: " + performanceProfile.App_type)
	return performanceProfile
}