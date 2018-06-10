package performance_profiles

import (
	"net/http"
	"encoding/json"
	"io/ioutil"
	"github.com/Cloud-Pie/SPDT/types"
)


func GetPerformanceProfiles(endpoint string) (types.PerformanceProfile, error){
	performanceProfile := types.PerformanceProfile{}
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