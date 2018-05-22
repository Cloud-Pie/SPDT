package performance_profiles

import (
	"github.com/yemramirezca/SPDT/internal/util"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"fmt"
)


type QoSParams struct {
	Request_time_out_sec int `json:"request_time_out_sec"`
	Request_availability_percent int `json:"request_availability_percent"`
}

type VmProfile struct {
	VmType   string `json:"vm_type"`
	Trn      int    `json:"trn"`
	NumCores int    `json:"num_cores"`
	MemoryGb int    `json:"memory_gb"`
}

type PerformanceModel struct {
	CSP        string             `json:"CSP"`
	VmProfiles [] VmProfile `json:"VMs"`
}

type PerformanceProfile struct {
	AppType           string              `json:"app_type"`
	DockerImageApp    string              `json:"docker_image_app"`
	GitUrlApp         string              `json:"git_url_app"`
	QoSParams         QoSParams           `json:"qos_params"`
	PerformanceModels [] PerformanceModel `json:perf_model`
}

func GetPerformanceProfiles () PerformanceProfile {
	fmt.Println("start profiles")

	performanceProfile := PerformanceProfile {}
	response, err := http.Get(util.URL_PROFILER)
	if err != nil {
		fmt.Printf("The profiler request failed with error %s\n", err)
		panic(err)
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)

	if err != nil {
		fmt.Printf("The profiler request failed with error %s\n", err)
		panic(err)
	}

	err = json.Unmarshal(data, &performanceProfile)
	if err != nil {
		fmt.Printf("The profiler request failed with error %s\n", err)
		panic(err)
	}
	fmt.Println("Successfully reading JSON file for App type: " + performanceProfile.AppType)
	return performanceProfile
}