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
      "TimeStart" : "2020-01-05T13:24:52Z",
      "Requests" : 300,
      "Trend" :	-1,
      "TimeEnd" : "2020-01-05T13:54:52Z"
    },
    {
      "TimeStart" : "2020-01-05T13:54:52Z",
      "Requests" : 800,
      "Trend" :	1,
      "TimeEnd" : "2020-01-05T14:54:52Z"
    },
    {
      "TimeStart" : "2020-01-05T14:54:52Z",
      "Requests" : 400,
      "Trend" :	-1,
      "TimeEnd" : "2020-01-05T15:24:52Z"
    }
  ]`
	err := json.Unmarshal([]byte(i), &intervals)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	return intervals
}
