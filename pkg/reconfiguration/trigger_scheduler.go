package reconfiguration

import (
	"fmt"

	"github.com/Cloud-Pie/SPDT/internal/util"

	"github.com/Cloud-Pie/Passa/server"
	"github.com/Cloud-Pie/SPDT/internal/types"
)

//TriggerScheduler sends the policies to Scheduler
func TriggerScheduler(policy types.Policy) {

	for _, conf := range policy.Configurations {
		serverMessenger := server.Communication{
			SchedulerURL: util.URL_SCHEDULER,
		}
		err := serverMessenger.CreateState(conf.State)

		if err != nil {
			fmt.Printf("The scheduler request failed with error %s\n", err)
			panic(err)
		}
	}
}
