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
	//Expected forecast json
	router.GET("/api/forecast", forecast)
	//Expected profiles json
	router.GET("/api/profiles", profiles)
	//Expected VMs json
	router.GET("/api/vms", vms)
	router.Run(":8081")
}

func forecast(c *gin.Context){
	// Open jsonFile
	data, err := ioutil.ReadFile("test/mock_forecast_test.json")
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
	data, err := ioutil.ReadFile("test/performance_profiles_test.json")
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
	data, err := ioutil.ReadFile("test/mock_vms.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	var vms interface{}
	err = json.Unmarshal(data, &vms)

	c.JSON(http.StatusOK, vms)
}