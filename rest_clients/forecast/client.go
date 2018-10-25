package forecast

import (
	"github.com/Cloud-Pie/SPDT/types"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"net/url"
	"time"
	"bytes"
	"github.com/Cloud-Pie/SPDT/util"
)

type RequestSubscription struct {
	IDPrediction      string                `json:"predictions_id"`
	URL				  string				`json:"url"`
}

func GetForecast(endpoint string, startTime time.Time, endTime time.Time) (types.Forecast, error){

	forecast := types.Forecast{}
	q := url.Values{}
	q.Add("start_time", startTime.Format(util.UTC_TIME_LAYOUT))
	q.Add("end_time", endTime.Format(util.UTC_TIME_LAYOUT))

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

func PostMaxRequestCapacities(loadCapacitiesPerState types.RequestCapacitySupply, endpoint string) error {
	jsonValue, _ := json.Marshal(loadCapacitiesPerState)
	_, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonValue))
	return err
}

func SubscribeNotifications(urlNotification string, idPrediction string, endpoint string) error {
	requestBody := RequestSubscription{IDPrediction: idPrediction, URL:urlNotification}
	jsonValue, _ := json.Marshal(requestBody)
	_, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonValue))
	return err
}