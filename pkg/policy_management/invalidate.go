package policy_management

import (
	"github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("spdt")

func ConflictTimeOldPolicy(forecast types.Forecast, timeConflict time.Time) (types.Policy, int){
	policyDAO := storage.GetPolicyDAO()

	_,err := policyDAO.Connect()
	if err != nil {
		log.Error(err.Error())
	}
	//Search policyByID created for the time window
	storedPolicy, err := policyDAO.FindOneByTimeWindow(forecast.TimeWindowStart, forecast.TimeWindowEnd)
	if err != nil {
		log.Error(err.Error())
	}
	var indexConflict int
	for i,c := range storedPolicy.ScalingActions {
		if timeConflict.Equal(c.TimeStart) || timeConflict.After(c.TimeStart) {
			indexConflict = i
		}
	}
	return storedPolicy, indexConflict
}

