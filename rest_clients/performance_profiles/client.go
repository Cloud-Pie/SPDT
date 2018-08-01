package performance_profiles

import (
	"net/http"
	"encoding/json"
	"io/ioutil"
	"github.com/Cloud-Pie/SPDT/types"
)


func GetPerformanceProfiles(endpoint string) (types.ServiceProfile, error){
	performanceProfile := types.ServiceProfile{}
	response, err := http.Get(endpoint)
	if err != nil {
		return performanceProfile,err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return performanceProfile,err
	}
	err = json.Unmarshal(data, &performanceProfile)
	if err != nil {
		return performanceProfile,err
	}
	return performanceProfile,err
}


func GetVMsProfiles(endpoint string) ([]types.VmProfile, error){
	vmList := []types.VmProfile{}
	response, err := http.Get(endpoint)
	if err != nil {
		return vmList,err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return vmList,err
	}
	err = json.Unmarshal(data, &vmList)
	if err != nil {
		return vmList,err
	}
	return vmList,err
}

func GetAllBootShutDownProfiles(endpoint string, vmType string) ([]types.BootShutDownTime, error){
	instanceValues := []types.BootShutDownTime{}
	response, err := http.Get(endpoint)
	if err != nil {
		return instanceValues,err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return instanceValues,err
	}
	err = json.Unmarshal(data, &instanceValues)
	if err != nil {
		return instanceValues,err
	}
	return instanceValues,err
}

func GetBootShutDownProfile(endpoint string, vmType string, numberInstance int) (types.BootShutDownTime, error){
	instanceValues := types.BootShutDownTime{}
	response, err := http.Get(endpoint)
	if err != nil {
		return instanceValues,err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return instanceValues,err
	}
	err = json.Unmarshal(data, &instanceValues)
	if err != nil {
		return instanceValues,err
	}
	return instanceValues,err
}