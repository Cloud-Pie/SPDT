package reconfiguration

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"encoding/json"
	"net/http"
	"bytes"
	"fmt"
	"io/ioutil"
)

func TriggerScheduler(policy types.Policy) bool{

	name := "service-name"
	url := "https://localhost:8080/api/states"
	start_time := "05-01-2020, 13:24:52 UTC"
	var services  = [] types.Service{{"ui-service", "3"}, {"backend-service", "2"}}
	var state = types.State{start_time, services, name}

	jsonValue, _ := json.Marshal(state)
	response, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))

	if err != nil {
		fmt.Printf("The scheduler request failed with error %s\n", err)
		panic(err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		fmt.Println(string(data))
	}

	return true
}
