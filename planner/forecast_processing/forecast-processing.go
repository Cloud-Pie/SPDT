package forecast_processing

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
)

func ScalingIntervals(forecast types.Forecast, granularity string) (types.ProcessedForecast) {
	var factor float64

	switch granularity {
		case util.HOUR:
			factor = 3600
		case util.MINUTE:
			factor = 60
		case util.SECOND:
			factor = 1
		default:
			factor = 3600
	}

	intervals := []types.CriticalInterval{}
	i:= 0
	lenValues := len(forecast.ForecastedValues)
	value := forecast.ForecastedValues[0]
	interval := types.CriticalInterval{
		Requests: value.Requests / factor,
		TimeStart:value.TimeStamp,
		TimeEnd:value.TimeStamp,
	}
	intervals = append(intervals, interval)

	for i <= lenValues - 2  {
		value := forecast.ForecastedValues[i]
		startTimestamp := value.TimeStamp
		highestPeak := value.Requests
		var nextValue types.ForecastedValue
		var endTimestamp time.Time

		for {

			nextValue = forecast.ForecastedValues[i+1]
			withinCoolDownTime := nextValue.TimeStamp.Sub(startTimestamp).Seconds() < 300

			endTimestamp = forecast.ForecastedValues[i+1].TimeStamp
			if withinCoolDownTime {
				highestPeak = (highestPeak + forecast.ForecastedValues[i+1].Requests)/2
			}
			i+=1
			if nextValue.TimeStamp.Sub(startTimestamp).Seconds() >= 300 {
				break
			}
		}

		interval := types.CriticalInterval{
			Requests:  highestPeak/factor,
			TimeStart: startTimestamp,
			TimeEnd:   endTimestamp,
		}
		intervals = append(intervals, interval)
	}
	processedForecast := types.ProcessedForecast{}
	processedForecast.CriticalIntervals = intervals
	return  processedForecast
}

