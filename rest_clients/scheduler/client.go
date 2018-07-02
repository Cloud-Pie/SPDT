package scheduler

import (
	"github.com/Cloud-Pie/SPDT/types"
	"encoding/json"
	"net/http"
	"bytes"
	"io/ioutil"
)



func CreateState(state types.State, endpoint string) error {
	jsonValue, _ := json.Marshal(state)
	_, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonValue))
	return err
}


func CurrentState(endpoint string) (types.State, error) {
	currentState := types.State{}
	response, err := http.Get(endpoint)
	if err != nil {
		return currentState, err
	}

	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return  currentState, err
	}
	err = json.Unmarshal(data, &currentState)
	if err != nil {
		return  currentState, err
	}
	return  currentState, err
}

