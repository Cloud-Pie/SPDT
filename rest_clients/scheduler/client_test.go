package scheduler

import (
	"testing"
	"time"
	"net/http"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/types"
)

func TestCreateState(t *testing.T) {
	if !isServerAvailable() {
				t.Skip("Server is not available")
	}

	state := types.State{
		ISOTime: time.Now(),
		Services: []types.Service{
			{
				Name:  "myservice",
				Scale: 10,
			},
			{
				Name:  "myservice2",
				Scale: 10,
			},
		},
		Name:    "test from SPDT",
		Vms: []types.VmScale{
			{
				Type:  "myvm1",
				Scale: 2,
			},
			{
				Type:  "myvm2",
				Scale: 1,
			},
		},
	}

	err := CreateState(state)
	if err != nil {
		t.Error(
			"For", "Forecast Service",
			"expected", nil,
			"got", err,
		)
	}
}

func isServerAvailable() bool {
	timeout := time.Duration(time.Second)
	client := http.Client{Timeout: timeout}
	_, err := client.Get(util.URL_SCHEDULER)

	if err == nil {
		return true
	}
	return false
}