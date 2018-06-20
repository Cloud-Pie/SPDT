package scheduler

import (
	"github.com/Cloud-Pie/SPDT/types"
	"encoding/json"
	"net/http"
	"bytes"
)

func CreateState(state types.State, endpoint string) error{
	jsonValue, _ := json.Marshal(state)
	_, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonValue))
	return err
}