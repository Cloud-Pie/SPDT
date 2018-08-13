package forecast

import (
	"github.com/Cloud-Pie/SPDT/types"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"net/url"
	"time"
)

func GetForecast(endpoint string, startTime time.Time, endTime time.Time) (types.Forecast, error){

	forecast := types.Forecast{}
	q := url.Values{}
	q.Add("timestamp-start", startTime.String())
	q.Add("timestamp-end", endTime.String())

	req, err := http.NewRequest("GET",endpoint,nil)
	if err != nil {
		return forecast,err
	}
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return forecast,err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return forecast,err
	}
	err = json.Unmarshal(data, &forecast)
	if err != nil {
		return forecast,err
	}
	return forecast,nil
}