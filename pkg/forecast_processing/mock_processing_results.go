package forecast_processing

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"encoding/json"
	"fmt"
)

func getMockData()  [] types.CriticalInterval{

	intervals := [] types.CriticalInterval{}

	i := `[
    {
      "TimeStart" : "10",
      "Requests" : 300,
      "Trend" :	-1,
      "TimeEnd" : "50"
    },
    {
      "TimeStart" : "60",
      "Requests" : 800,
      "Trend" :	1,
      "TimeEnd" : "80"
    },
    {
      "TimeStart" : "90",
      "Requests" : 400,
      "Trend" :	-1,
      "TimeEnd" : "100"
    }
  ]`
	err := json.Unmarshal([]byte(i), &intervals)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	return intervals
}
