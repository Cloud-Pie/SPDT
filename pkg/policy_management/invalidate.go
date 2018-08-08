package policy_management

import (
	"github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("spdt")

func ConflictTimeOldPolicy(forecast types.Forecast, timeConflict time.Time) (types.Policy, int){
	policyDAO := storage.PolicyDAO{
		Server:util.DEFAULT_DB_SERVER_POLICIES,
		Database:util.DEFAULT_DB_POLICIES,
	}

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
	for i,c := range storedPolicy.Configurations {
		if timeConflict.Equal(c.TimeStart) || timeConflict.After(c.TimeStart) {
			indexConflict = i
		}
	}
	return storedPolicy, indexConflict
}

