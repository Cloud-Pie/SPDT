package forecast

import (
	"testing"
	"time"
	"net/http"
	"github.com/Cloud-Pie/SPDT/util"
)

func TestGetForecast(t *testing.T) {
	if !isServerAvailable() {
				t.Skip("Server is not available")
	}

	forecast, err := GetForecast()
	if err != nil {
		t.Error(
			"For", "Forecast Service",
			"expected", nil,
			"got", err,
		)
	}

	if len (forecast.ForecastedValues) == 0 {
		t.Error(
			"For", "Forecasted values lenght",
			"expected", ">0",
			"got", 0,
		)
	}
}

func isServerAvailable() bool {
	timeout := time.Duration(time.Second)
	client := http.Client{Timeout: timeout}
	_, err := client.Get(util.URL_FORECAST)

	if err == nil {
		return true
	}
	return false
}