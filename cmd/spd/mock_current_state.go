package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/Cloud-Pie/Passa/ymlparser"
)

func CurrentState() ymlparser.State {
	var state ymlparser.State

	data, err := ioutil.ReadFile("cmd/spd/mock_current_state.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	err = json.Unmarshal(data, &state)
	return state
}
