package forecast

import (
	"github.com/Cloud-Pie/SPDT/types"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

func GetForecast(endpoint string) (types.Forecast, error){
	forecast := types.Forecast{}
	response, err := http.Get(endpoint)
	if err != nil {
		return forecast,err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return forecast,err
	}
	err = json.Unmarshal(data, &forecast)
	if err != nil {
		return forecast,err
	}
	return forecast,nil
}