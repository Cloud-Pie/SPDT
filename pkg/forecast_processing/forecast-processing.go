package forecast_processing

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	SProcessing "github.com/Cloud-Pie/SPDT/rest_clients/time_serie_processing"
	"time"
	"math"
)

//DEPRECATED
func ProcessData(forecast types.Forecast) (types.ProcessedForecast) {
	values := [] int {}
	times := [] time.Time {}
	for _,x := range forecast.ForecastedValues {
		values = append(values,x.Requests)
		times = append(times,x.TimeStamp)
	}
	threshold := 1200		//TODO: Get current TRN
	poiList,_ := SProcessing.ProcessData(values, threshold, util.URL_SERIE_PROCESSING)
	intervals := []types.CriticalInterval{}

	for _,item := range poiList {
		interval := types.CriticalInterval{}
		interval.Requests = values[item.Index]
		interval.TimePeak = times[item.Index]
		//interval.TimeStart = adjustTime(times[int(item.Start.Index)], math.Floor(item.Start.) - item.Start)
		//interval.TimeEnd = adjustTime(times[int(item.End.Index)], math.Floor(item.End) - item.End)

		interSize := len(intervals)
		if (interSize > 1){
			//start time is equal to the end time from previous interval
			interval.TimeStart = intervals[interSize-1].TimeEnd
		}else {
			interval.TimeStart = times[int(item.Start.Index)]
		}

		//Calculate End Time using the ips_left
		 //timeValley := times[int(item.End.Index)]
		 timeValleyIpsRight := adjustTime(times[int(item.End.Index)], math.Floor(item.End.Right_ips) - item.End.Right_ips)
		 interval.TimeEnd = timeValleyIpsRight
		interval.AboveThreshold = item.Peak
		intervals = append(intervals, interval)
	}
	processedForecast := types.ProcessedForecast{}
	processedForecast.CriticalIntervals = intervals

	return processedForecast
}

func adjustTime(t time.Time, factor float64) time.Time{
	f := factor*3600
	return t.Add(time.Duration(f) * time.Second)
}

func PointsOfInterest(forecast types.Forecast) ([]types.PoI, []int, [] time.Time){
	values := [] int {}
	times := [] time.Time {}
	for _,x := range forecast.ForecastedValues {
		values = append(values,x.Requests)
		times = append(times,x.TimeStamp)
	}
	threshold := 1200		//TODO: Get current TRN
	poiList,_ := SProcessing.ProcessData(values, threshold, util.URL_SERIE_PROCESSING)
	return  poiList,values,times
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
