package performance_profiles

import (
	"net/http"
	"encoding/json"
	"io/ioutil"
	"github.com/Cloud-Pie/SPDT/types"
	"net/url"
	"github.com/Cloud-Pie/SPDT/util"
	"strconv"
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

func GetServicePerformanceProfiles(endpoint string, appName string, appType string, mainServiceName string) (types.ServicePerformanceProfile, error){

	servicePerformanceProfile := types.ServicePerformanceProfile{}
	parameters := make(map[string]string)
	parameters["apptype"] = appType
	parameters["appname"] = appName
	parameters["mainservicename"] = mainServiceName
	endpoint = util.ParseURL(endpoint, parameters)

	response, err := http.Get(endpoint)
	if err != nil {
		return servicePerformanceProfile,err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return servicePerformanceProfile,err
	}
	err = json.Unmarshal(data, &servicePerformanceProfile)
	if err != nil {
		return servicePerformanceProfile,err
	}
	return servicePerformanceProfile,err
}

func GetPredictedReplicas(endpoint string, appName string, appType string, mainServiceName string,  msc float64, cpuCores float64, memGb float64) (types.MSCCompleteSetting, error){
	mscSetting := types.MSCCompleteSetting{}
	parameters := make(map[string]string)
	parameters["apptype"] = appType
	parameters["appname"] = appName
	parameters["mainservicename"] = mainServiceName
	parameters["msc"] = strconv.FormatFloat(msc, 'f', 1, 64)
	parameters["numcoresutil"] = strconv.FormatFloat(cpuCores, 'f', 1, 64)
	parameters["numcoreslimit"] = strconv.FormatFloat(cpuCores, 'f', 1, 64)
	parameters["nummemlimit"] = strconv.FormatFloat(memGb, 'f', 1, 64)

	endpoint = util.ParseURL(endpoint, parameters)

	response, err := http.Get(endpoint)
	if err != nil {
		return mscSetting,err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return mscSetting,err
	}
	err = json.Unmarshal(data, &mscSetting)
	if err != nil {
		return mscSetting,err
	}
	return mscSetting,err
}

func GetPredictedMSCByReplicas(endpoint string, appName string, appType string, mainServiceName string,  replicas int, cpuCores float64, memGb float64) (types.MSCCompleteSetting, error){
	mscSetting := types.MSCCompleteSetting{}
	parameters := make(map[string]string)
	parameters["apptype"] = appType
	parameters["appname"] = appName
	parameters["mainservicename"] = mainServiceName
	parameters["replicas"] = strconv.Itoa(replicas)
	parameters["numcoresutil"] = strconv.FormatFloat(cpuCores, 'f', 1, 64)
	parameters["numcoreslimit"] = strconv.FormatFloat(cpuCores, 'f', 1, 64)
	parameters["nummemlimit"] = strconv.FormatFloat(memGb, 'f', 1, 64)

	endpoint = util.ParseURL(endpoint, parameters)

	response, err := http.Get(endpoint)
	if err != nil {
		return mscSetting,err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return mscSetting,err
	}
	err = json.Unmarshal(data, &mscSetting)
	if err != nil {
		return mscSetting,err
	}
	return mscSetting,err
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

func GetAllBootShutDownProfilesByType(endpoint string, vmType string, region string, csp string) (types.InstancesBootShutdownTime, error){
	instanceValues := types.InstancesBootShutdownTime{}
	q := url.Values{}
	q.Add("instanceType", vmType)
	q.Add("region", region)
	q.Add("approach", "regression_vm_boot")
	q.Add("csp", csp)

	req, err := http.NewRequest("GET",endpoint,nil)
	if err != nil {
		return instanceValues,err
	}

	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	response, err := client.Do(req)
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

func GetBootShutDownProfileByType(endpoint string, vmType string, numberInstance int, csp string, region string) (types.BootShutDownTime, error){
	instanceValues := types.BootShutDownTime{}

	q := url.Values{}
	q.Add("instanceType", vmType)
	q.Add("region", region)
	q.Add("approach", "avg")
	q.Add("csp", csp)
	q.Add("numInstances", strconv.Itoa(numberInstance))

	req, err := http.NewRequest("GET",endpoint,nil)
	if err != nil {
		return instanceValues,err
	}

	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	response, err := client.Do(req)
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