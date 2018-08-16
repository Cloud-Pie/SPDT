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
	threshold := 1200		//TODO: Get current TRN
	poiList,err := SProcessing.ProcessData(values, threshold, util.URL_SERIE_PROCESSING)
	return  poiList,values,times, err
}
