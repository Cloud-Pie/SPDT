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


func InfraCurrentState(endpoint string) (types.State, error) {
	currentState := types.State{}
	infrastructureState := types.InfrastractureState{}
	response, err := http.Get(endpoint)
	if err != nil {
		return currentState, err
	}

	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return  currentState, err
	}
	err = json.Unmarshal(data, &infrastructureState)
	if err != nil {
		return  currentState, err
	}
	currentState = infrastructureState.ActiveState
	return  currentState, err
}

func InvalidateStates(endpoint string) (error) {
	response, err := http.Get(endpoint)
	if err != nil {
		return  err
	}
	defer response.Body.Close()
	_,err = ioutil.ReadAll(response.Body)
	if err != nil {
		return   err
	}
	return   err
}

