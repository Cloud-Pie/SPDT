package main

import (
	"github.com/Cloud-Pie/SPDT/types"
	"io/ioutil"
	"fmt"
	"encoding/json"
)


func CurrentState() types.State {
	var state types.State

	data, err := ioutil.ReadFile("cmd/spd/mock_current_state.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	err = json.Unmarshal(data, &state)
	return state
}