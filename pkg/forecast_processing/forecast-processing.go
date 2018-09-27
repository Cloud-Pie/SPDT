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
	i:= 1
	lenValues := len(forecast.ForecastedValues)
	for i <= lenValues - 1  {
		value := forecast.ForecastedValues[i]
		lastTimestamp := forecast.ForecastedValues[i-1].TimeStamp
		//TODO: Adjust
		/*lastRequests := forecast.ForecastedValues[i-1].Requests
		isCurrentScaleIn := (lastRequests - value.Requests) > 0
		isNextScaleOut := false
		if i + 1 < lenValues {
			nextRequests := forecast.ForecastedValues[i+1].Requests
			isNextScaleOut = (value.Requests - nextRequests) < 0
		}
		if value.TimeStamp.Sub(lastTimestamp).Minutes() < 20 && isCurrentScaleIn && isNextScaleOut {
			lenIntervals := len(intervals)
			if lenIntervals > 0 {
				intervals[lenIntervals-1].TimeEnd = value.TimeStamp
				lastRequest := intervals[lenIntervals-1].Requests
				if lastRequest < value.Requests {
					intervals[lenIntervals-1].Requests = value.Requests
				}
			} else {
				interval := types.CriticalInterval{Requests: value.Requests, TimeStart:lastTimestamp, TimeEnd:value.TimeStamp }
				intervals = append(intervals, interval)
			}
		} else {*/
			interval := types.CriticalInterval{Requests: value.Requests, TimeStart:lastTimestamp, TimeEnd:value.TimeStamp }
			intervals = append(intervals, interval)
		//}
		i+=1
	}
	processedForecast := types.ProcessedForecast{}
	processedForecast.CriticalIntervals = intervals

	return  processedForecast
}
