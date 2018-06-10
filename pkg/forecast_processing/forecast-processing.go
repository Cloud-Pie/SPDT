package forecast_processing

import (
	"github.com/Cloud-Pie/SPDT/types"
	"fmt"
	"github.com/Cloud-Pie/SPDT/util"
)

func ProcessData(forecast types.Forecast) types.ProcessedForecast {
	//TODO: Process the Time Serie
	//arr := []int {0, 10, 20, 30, 40,2,3,4,5,11,23,1,4,5,2}
	//b := burst(arr,10)
	array := detectBurst(forecast, 1000)
	for _,in := range array {
		s1 := in.TimeStart
		s2 := in.TimeEnd
		fmt.Printf("\nstart: %s \nend: %s \nrequests: %d\n", s1.Format(util.TIME_LAYOUT), s2.Format(util.TIME_LAYOUT),in.Requests)
	}
	return getMockData()
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