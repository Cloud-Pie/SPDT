package forecast

import (
	"github.com/Cloud-Pie/SPDT/types"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"bytes"
)

type Serie struct {
	Serie	[]int	 `json:"serie"`
	Threshold int 	`json:"threshold"`
}

type ResponsePoI struct {
	PoI	[]types.PoI	 `json:"PoI"`
}


func ProcessData(values []int, threshold int, endpoint string) ([]types.PoI, error){
	poiList:= []types.PoI{}

	serie := Serie{Serie:values, Threshold:threshold}
	jsonValue, err := json.Marshal(serie)
	if err != nil {
		return poiList,err
	}
	response, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonValue))

	responsePoI := ResponsePoI{}
	if err != nil {
		return poiList,err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return poiList,err
	}
	err = json.Unmarshal(data, &responsePoI)
	if err != nil {
		return poiList,err
	}

	return responsePoI.PoI,nil
}

