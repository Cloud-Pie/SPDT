package performance_profiles

import (
	"testing"
	"time"
	"net/http"
	"github.com/Cloud-Pie/SPDT/util"
)

func TestGetPerformanceProfiles(t *testing.T) {
	if !isServerAvailable() {
				t.Skip("Server is not available")
	}

	profile, err := GetPerformanceProfiles()
	if err != nil {
		t.Error(
			"For", "Performance Profile Service",
			"expected", nil,
			"got", err,
		)
	}

	if len (profile.PerformanceModels) == 0 {
		t.Error(
			"For", "Performance profiles values lenght",
			"expected", ">0",
			"got", 0,
		)
	}

}

func isServerAvailable() bool {
	timeout := time.Duration(time.Second)
	client := http.Client{Timeout: timeout}
	_, err := client.Get(util.URL_PROFILER)

	if err == nil {
		return true
	}
	return false
}