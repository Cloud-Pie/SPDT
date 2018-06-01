package performance_profiles

import (
	"github.com/Cloud-Pie/SPDT/internal/util"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"github.com/Cloud-Pie/SPDT/internal/types"
)


func GetPerformanceProfiles() (types.PerformanceProfile, error){
	performanceProfile := types.PerformanceProfile {}
	response, err := http.Get(util.URL_PROFILER)
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