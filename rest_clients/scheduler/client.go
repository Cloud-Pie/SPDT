package scheduler

import (
	"github.com/Cloud-Pie/SPDT/types"
	"encoding/json"
	"net/http"
	"bytes"
	"io/ioutil"
	"time"
	"github.com/Cloud-Pie/SPDT/util"
)

type StateToSchedule struct {
	LaunchTime time.Time 						`json:"ISODate"`
	Services   map[string]ServiceToSchedule     `json:"Services"`
	Name       string    						`json:"Name"`
	VMs        types.VMScale   					`json:"VMs"`
	ExpectedStart time.Time 					`json:"ExpectedTime"`
}

type ServiceToSchedule struct {
	Scale 	int			`json:"Replicas"`
	CPU 	string		`json:"Cpu"`
	Memory	int64		`json:"Memory"`
}

type InfrastructureState struct {
	ActiveState				StateToSchedule	`json:"active" bson:"active"`
	LastDeployedState		StateToSchedule	`json:"lastDeployed" bson:"lastDeployed"`
	isStateTrue				bool	`json:"isStateTrue" bson:"isStateTrue"`
}

func CreateState(stateToSchedule StateToSchedule, endpoint string) error {
	jsonValue, _ := json.Marshal(stateToSchedule)
	_, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonValue))
	return err
}


func InfraCurrentState(endpoint string) (StateToSchedule, error) {
	currentState := StateToSchedule{}
	infrastructureState := InfrastructureState{}
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

func InvalidateStates(timestamp time.Time,endpoint string) (error) {
	parameters := make(map[string]string)
	parameters["timestamp"] = timestamp.Format(util.UTC_TIME_LAYOUT)
	endpoint = util.ParseURL(endpoint,parameters )
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

