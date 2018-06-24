package policies_derivation

import (
	"time"
	"github.com/Cloud-Pie/SPDT/types"
)

type SmallStepUnderProvision struct {
	NIntervals	int
}

func (derivationStrategy SmallStepUnderProvision) WindowDerivation(values []int, times [] time.Time) (types.ProcessedForecast) {
	intervals := []types.CriticalInterval{}

	for ind, requests := range values {
		if (ind == 0){
			break
		}
		interval := types.CriticalInterval{}
		interval.Requests = requests
		interval.TimePeak = times[ind]
		interval.TimeStart = times[ind-1]
		interval.TimeEnd = times[ind]
		intervals = append(intervals, interval)
	}
	derivationStrategy.NIntervals = len(intervals)
	processedForecast := types.ProcessedForecast{}
	processedForecast.CriticalIntervals = intervals

	return  processedForecast
}

func (derivationStrategy SmallStepUnderProvision) NumberIntervals() int{
	return derivationStrategy.NIntervals
}
