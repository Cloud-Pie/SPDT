package reconfiguration

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"encoding/json"
	"net/http"
	"bytes"
	"fmt"
	"io/ioutil"
	"github.com/yemramirezca/SPDT/internal/util"
)

func TriggerScheduler(policy types.Policy) bool{

	name := "service-name"
	start_time := "05-01-2020, 13:24:52 UTC"
	var services  = [] types.Service{{"ui-service", "3"}, {"backend-service", "2"}}
	var state = types.State{start_time, services, name}

	jsonValue, _ := json.Marshal(state)
	response, err := http.Post(util.URL_SCHEDULER, "application/json", bytes.NewBuffer(jsonValue))

	if err != nil {
		fmt.Printf("The scheduler request failed with error %s\n", err)
		panic(err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		fmt.Println(string(data))
	}

	return true
}
