package schedule

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/rest_clients/scheduler"
	"time"
	"github.com/Cloud-Pie/SPDT/util"
)

func TriggerScheduler(policy types.Policy, endpoint string)error{
	for _, conf := range policy.ScalingActions {
		err := scheduler.CreateState(conf.State, endpoint)
		if err != nil {
			return err
		}
	}
	return nil
}

func InvalidateStates(timestamp time.Time, endpoint string)error {
	parameters := make(map[string]string)
	parameters["timestamp"] = timestamp.Format(util.UTC_TIME_LAYOUT)
	endpoint = util.ParseURL(endpoint,parameters )
		err := scheduler.InvalidateStates(endpoint)
		if err != nil {
			return err
		}
		return nil
}