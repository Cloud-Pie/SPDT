package forecast_processing

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	SProcessing "github.com/Cloud-Pie/SPDT/rest_clients/time_serie_processing"
	"time"
	"math"
)

func ProcessData(forecast types.Forecast) types.ProcessedForecast {
	values := [] int {}
	times := [] time.Time {}
	for _,x := range forecast.ForecastedValues {
		values = append(values,x.Requests)
		times = append(times,x.TimeStamp)
	}
	threshold := 12		//TODO: Get current TRN
	poiList,_ := SProcessing.ProcessData(values, threshold, util.URL_SERIE_PROCESSING)

	intervals := []types.CriticalInterval{}

	for _,item := range poiList {
		interval := types.CriticalInterval{}
		interval.Requests = values[item.Index]
		interval.TimePeak = times[item.Index]
		interval.TimeStart = adjustTime(times[int(item.Start)], math.Floor(item.Start) - item.Start)
		interval.TimeEnd = adjustTime(times[int(item.End)], math.Floor(item.End) - item.End)
		interval.AboveThreshold = item.Peak
		intervals = append(intervals, interval)
	}
	processedForecast := types.ProcessedForecast{true, intervals}

	return processedForecast
}

func adjustTime(t time.Time, factor float64) time.Time{
	f := factor*3600
	return t.Add(time.Duration(f) * time.Second)
}
func detectBurst(forecast types.Forecast, threshold int) [] types.CriticalInterval{

	flag := false
	criticalIntervals := []types.CriticalInterval{}

	for _, fv := range forecast.ForecastedValues {
		if (fv.Requests > threshold) {
			if flag == true {
				leng := len(criticalIntervals)
				criticalIntervals[leng-1].TimeEnd = fv.TimeStamp
				if (criticalIntervals[leng-1].Requests < fv.Requests) {
					criticalIntervals[leng-1].Requests = fv.Requests
					criticalIntervals[leng-1].TimePeak = fv.TimeStamp
				}
			}else {
				interval := types.CriticalInterval{}
				interval.TimeStart = fv.TimeStamp
				interval.TimeEnd = fv.TimeStamp
				interval.Requests = fv.Requests
				interval.TimePeak = fv.TimeStamp
				criticalIntervals = append(criticalIntervals, interval)
				flag = true
			}
		} else {
			flag = false
		}
	}
	return criticalIntervals
}
