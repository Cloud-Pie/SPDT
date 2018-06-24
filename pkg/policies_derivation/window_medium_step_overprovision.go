package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
)

type MediumStepOverprovision struct {
	PoIList []types.PoI
	NIntervals	int
}

func (derivationStrategy MediumStepOverprovision) WindowDerivation(values []int, times [] time.Time) (types.ProcessedForecast){

	intervals := []types.CriticalInterval{}

	for _,item := range derivationStrategy.PoIList {
		interval := types.CriticalInterval{}
		interval.Requests = values[item.Index]
		interval.TimePeak = times[item.Index]

		interSize := len(intervals)
		if (interSize >= 1){
			//start time is equal to the end time from previous interval
			interval.TimeStart = intervals[interSize-1].TimeEnd
		}else {
			interval.TimeStart = times[int(item.Start.Index)]
		}
		//Calculate End Time using the ips_left
		timeValleyIpsLeft := adjustTime(times[int(item.End.Index)], item.End.Left_ips - math.Floor(item.End.Left_ips))
		if timeValleyIpsLeft.After(interval.TimePeak) {
			interval.TimeEnd = timeValleyIpsLeft
		} else {
			interval.TimeEnd = times[int(item.End.Index)]
		}
		interval.AboveThreshold = item.Peak
		intervals = append(intervals, interval)
	}
	derivationStrategy.NIntervals = len(intervals)
	processedForecast := types.ProcessedForecast{}
	processedForecast.CriticalIntervals = intervals

	return processedForecast
}

func (derivationStrategy MediumStepOverprovision) NumberIntervals() int{
	return derivationStrategy.NIntervals
}

