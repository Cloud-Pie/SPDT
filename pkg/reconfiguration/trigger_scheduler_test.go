package reconfiguration

import (
	"net/http"
	"testing"
	"time"

	"github.com/Cloud-Pie/Passa/ymlparser"

	"github.com/Cloud-Pie/SPDT/internal/types"
	"github.com/Cloud-Pie/SPDT/internal/util"
)

func TestScheduler_TriggerScheduler(t *testing.T) {
	if !isServerOpen() {
		t.Skip("Server is not open!")
	}
	type fields struct {
		SchedulerURL string
	}
	type args struct {
		policy types.Policy
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Valid State",
			fields: fields{
				SchedulerURL: util.URL_SCHEDULER,
			},
			args: args{
				policy: types.Policy{
					ID:        "myid",
					TotalCost: 40,
					Configurations: []types.Configuration{
						{
							TransitionCost: -3,
							TimeStart:      time.Now(),
							TimeEnd:        time.Now().Add(time.Duration(2) * time.Hour),
							State: ymlparser.State{
								ISODate: time.Now(),
								Name:    "test from SPDT",
								Services: []ymlparser.Service{
									{
										Name:  "myservice",
										Scale: 10,
									},
									{
										Name:  "myservice2",
										Scale: 10,
									},
								},
								VMs: []ymlparser.VM{
									{
										Type:  "myvm1",
										Scale: 2,
									},
									{
										Type:  "myvm2",
										Scale: 1,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			TriggerScheduler(tt.args.policy)
		})
	}
}

func isServerOpen() bool {
	timeout := time.Duration(2 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	_, err := client.Get(util.URL_SCHEDULER)
	if err == nil {
		return true
	}
	return false
}
