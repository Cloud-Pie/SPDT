package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
)

type SmallStepOverProvision struct {
	NIntervals	int
	PoIList []types.PoI
}

func (derivationStrategy SmallStepOverProvision) WindowDerivation(values []float64, times [] time.Time) (types.ProcessedForecast) {
	intervals := []types.CriticalInterval{}
	nValues := len(values)
	for _, it:= range derivationStrategy.PoIList {
		i:= it.Index
		for _,ind := range it.Index_in_interval_left {
			if (ind >= 1) {
				interval := types.CriticalInterval{}
				interval.Requests = values[ind]
				interval.TimePeak = times[ind]
				interval.TimeStart = times[ind-1]
				interval.TimeEnd = times[ind ]
				intervals = append(intervals, interval)
			}
		}

		if(i-1 > 0) {
			//Interval to the left before the peak
			interval := types.CriticalInterval{}
			interval.Requests = values[i]
			interval.TimePeak = times[i]
			interval.TimeStart = times[i-1]
			interval.TimeEnd = times[i]
			intervals = append(intervals, interval)
		}

		if(i+1 < nValues) {
			//Interval to the right after the peak
			interval := types.CriticalInterval{}
			interval.Requests = values[i]
			interval.TimePeak = times[i]
			interval.TimeStart = times[i]
			interval.TimeEnd = times[i+1]
			intervals = append(intervals, interval)
		}


		for _,ind := range it.Index_in_interval_right {
			if ((ind+1) < nValues) {
				interval := types.CriticalInterval{}
				interval.Requests = values[ind]
				interval.TimePeak = times[ind]
				interval.TimeStart = times[ind]
				interval.TimeEnd = times[ind+1]
				intervals = append(intervals, interval)
			}
		}
	}

	derivationStrategy.NIntervals = len(intervals)
	processedForecast := types.ProcessedForecast{}
	processedForecast.CriticalIntervals = intervals

	return  processedForecast
}

func (derivationStrategy SmallStepOverProvision) NumberIntervals() int{
	return derivationStrategy.NIntervals
}

