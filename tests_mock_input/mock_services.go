package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"fmt"
	"io/ioutil"
	"encoding/json"
)

func main () {

	router := gin.Default()
	//Expected VMs json
	router.POST("/subscribe", subscribe)
	//Expected forecast json
	router.GET("/predict", forecast)
	//Expected profiles json
	router.GET("/getRegressionTRNsMongoDBAll/:apptype/:appname", profiles)
	//Expected VMs json
	router.GET("/api/vms", vms)
	//Expected Current DesiredState json
	router.GET("/api/current", currentState)
	//Expected  All VM  Times
	router.GET("/getAllVMTypesBootShutDownDataAvg",allVMsTimes )
	//Expected   VM  Times
	router.GET("/getPerVMTypeOneBootShutDownData", vmsTimes)
	router.Run(":8081")
}

func forecast(c *gin.Context){
	// Open jsonFile
	data, err := ioutil.ReadFile("tests_mock_input/mock_forecast_test.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	var forecast interface{}
	err = json.Unmarshal(data, &forecast)
	c.JSON(http.StatusOK, forecast)
}


func profiles(c *gin.Context){
	// Open jsonFile
	data, err := ioutil.ReadFile("tests_mock_input/performance_profiles_test.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	var profile interface{}
	err = json.Unmarshal(data, &profile)

	c.JSON(http.StatusOK, profile)
}

func vms(c *gin.Context){
	// Open jsonFile
	data, err := ioutil.ReadFile("tests_mock_input/vm_profiles.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	var vms interface{}
	err = json.Unmarshal(data, &vms)

	c.JSON(http.StatusOK, vms)
}

func currentState(c *gin.Context){
	// Open jsonFile
	data, err := ioutil.ReadFile("tests_mock_input/mock_current_state.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	var state interface{}
	err = json.Unmarshal(data, &state)

	c.JSON(http.StatusOK, state)
}

func allVMsTimes(c *gin.Context){
	appraoch := c.DefaultQuery("appraoch", "")
	csp := c.DefaultQuery("csp","")
	instanceType := c.DefaultQuery("instanceType","")
	numInstances := c.DefaultQuery("numInstances","")
	region := c.DefaultQuery("region","")

	// Open jsonFile
	data, err := ioutil.ReadFile("tests_mock_input/mock_vms_all_times.json")
	if err != nil {
		fmt.Println(err)
		fmt.Printf(appraoch + csp + instanceType + numInstances + region)
	}

	var state interface{}
	err = json.Unmarshal(data, &state)

	c.JSON(http.StatusOK, state)
}

func vmsTimes(c *gin.Context){
	// Open jsonFile
	data, err := ioutil.ReadFile("tests_mock_input/mock_vms_times.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	var state interface{}
	err = json.Unmarshal(data, &state)

	c.JSON(http.StatusOK, state)
}

func subscribe(c *gin.Context){
	r := c.Request.Body
	c.JSON(http.StatusOK, r)
}