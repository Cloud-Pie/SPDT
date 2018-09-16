package forecast_processing

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/storage"
	"time"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("spdt")

func UpdateForecast(forecast types.Forecast) (bool, types.Forecast, time.Time) {
		forecastDAO := storage.GetForecastDAO()
		_,err := forecastDAO.Connect()
		if err != nil {
			log.Fatalf(err.Error())
		}

		var shouldUpdate bool
		var timeConflict time.Time
		var indexTimeConflict int
		//Compare with the previous forecast if sth changed
		resultQuery, err := forecastDAO.FindAll() //TODO: Write better query
		if len(resultQuery) == 1 {
			oldForecast := resultQuery[0]
			if shouldUpdate, indexTimeConflict = isConflict(forecast, oldForecast); shouldUpdate {
				id := resultQuery[0].IDdb
				forecastDAO.Update(id, forecast)
			}
		} else {
			//Case when there was not previous forecast
			err = forecastDAO.Insert(forecast)
			if err != nil {
				log.Error(err.Error())
			}
		}

		timeConflict = forecast.ForecastedValues[indexTimeConflict].TimeStamp
		forecast.ForecastedValues = forecast.ForecastedValues[indexTimeConflict:]

	return shouldUpdate, forecast, timeConflict
}

func isConflict(current types.Forecast, old types.Forecast) (bool, int) {
	var indexTimeConflict 	int
	var iSsignificantChange bool

	rmse := RMSE(old.ForecastedValues, current.ForecastedValues)
	if len(current.ForecastedValues) == len(old.ForecastedValues) && current.TimeWindowStart.Equal(old.TimeWindowStart) {
		for i,in := range current.ForecastedValues {
			if in.Requests - old.ForecastedValues[i].Requests != 0{
				indexTimeConflict = i
				break
			}
		}
	}

	iSsignificantChange = rmse > 1.0
	return iSsignificantChange, indexTimeConflict
}

func GetMaxRequestCapacity(policy types.Policy) types.RequestCapacitySupply{
	var requestCapacitySupply types.RequestCapacitySupply
	statesLoadCapacity := []types.StateLoadCapacity{}
	 for _,v := range policy.ScalingActions {
	 	statesLoadCapacity = append(statesLoadCapacity, types.StateLoadCapacity{Requests:v.Metrics.RequestsCapacity,
	 																			TimeStamp:v.TimeStart})
	 }
	 requestCapacitySupply.StatesCapacity = statesLoadCapacity

	return requestCapacitySupply
}