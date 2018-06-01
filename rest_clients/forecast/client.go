package forecast

import (
	"github.com/Cloud-Pie/SPDT/types"
	"net/http"
	"github.com/Cloud-Pie/SPDT/util"
	"io/ioutil"
	"encoding/json"
)

func GetForecast() (types.Forecast, error){
	forecast := types.Forecast{}
	response, err := http.Get(util.URL_FORECAST)
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