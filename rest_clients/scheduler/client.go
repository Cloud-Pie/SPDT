package scheduler

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"encoding/json"
	"net/http"
	"bytes"
)

func CreateState(state types.State) error{
	jsonValue, _ := json.Marshal(state)
	_, err := http.Post(util.URL_SCHEDULER, "application/json", bytes.NewBuffer(jsonValue))
	return err
}