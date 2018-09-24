package forecast_processing

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	SProcessing "github.com/Cloud-Pie/SPDT/rest_clients/time_serie_processing"
	"time"
)

//Calls the service to process the forecast received and return the points of interest  found
func PointsOfInterest(forecast types.Forecast) ([]types.PoI, []float64, [] time.Time, error){
	values := [] float64 {}
	times := [] time.Time {}
	for _,x := range forecast.ForecastedValues {
		values = append(values,x.Requests)
		times = append(times,x.TimeStamp)
	}
	poiList,err := SProcessing.ProcessData(values, util.URL_SERIE_PROCESSING)
	return  poiList,values,times, err
}


func WindowDerivation(forecast types.Forecast) (types.ProcessedForecast) {
	intervals := []types.CriticalInterval{}

	for i, fv := range forecast.ForecastedValues {
		if i > 0 {
			interval := types.CriticalInterval{}
			interval.Requests = fv.Requests
			interval.TimePeak = fv.TimeStamp
			interval.TimeStart = forecast.ForecastedValues[i-1].TimeStamp
			interval.TimeEnd = fv.TimeStamp
			intervals = append(intervals, interval)
		}
	}
	processedForecast := types.ProcessedForecast{}
	processedForecast.CriticalIntervals = intervals

	return  processedForecast
}
