package forecast_processing

import (
	"github.com/Cloud-Pie/SPDT/types"
	"math"
)

func RMSE(forecast []types.ForecastedValue, newForecast []types.ForecastedValue) float64{
	 rmserrors := 0.0
	 sum := 0.0
	 for i,k := range newForecast {
		 sum += ((k.Requests - forecast[i].Requests) * (k.Requests - forecast[i].Requests))
	 }
 	n := float64(len(forecast))
	rmserrors = math.Sqrt(sum/n)
	return rmserrors
}
