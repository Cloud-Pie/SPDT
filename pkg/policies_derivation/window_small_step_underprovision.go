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
	lenValues := len(values)
	for i:=1; i<lenValues-1;i++ {
		interval := types.CriticalInterval{}
		interval.Requests = float64(values[i])
		interval.TimePeak = times[i]
		interval.TimeStart = times[i-1]
		interval.TimeEnd = times[i]
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
