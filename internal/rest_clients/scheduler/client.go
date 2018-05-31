package scheduler

import (
	"github.com/Cloud-Pie/SPDT/internal/types"
	"github.com/Cloud-Pie/SPDT/internal/util"
	"encoding/json"
	"net/http"
	"bytes"
)

func CreateState(state types.State) error{
	jsonValue, _ := json.Marshal(state)
	_, err := http.Post(util.URL_SCHEDULER, "application/json", bytes.NewBuffer(jsonValue))
	return err
}