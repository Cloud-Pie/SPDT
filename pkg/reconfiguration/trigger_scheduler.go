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

func TriggerScheduler(policy types.Policy){

	for _, conf := range policy.Configurations {
		jsonValue, _ := json.Marshal(conf.State)
		response, err := http.Post(util.URL_SCHEDULER, "application/json", bytes.NewBuffer(jsonValue))

		if err != nil {
			fmt.Printf("The scheduler request failed with error %s\n", err)
			panic(err)
		} else {
			data, _ := ioutil.ReadAll(response.Body)
			fmt.Println(string(data))
		}
	}
}
