package forecast_processing

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func getMockData() types.Forecast{

	forecast := types.Forecast{}

	data, err := ioutil.ReadFile("pkg/forecast_processing/mock_forecast.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	err = json.Unmarshal(data, &forecast)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	return forecast
}
