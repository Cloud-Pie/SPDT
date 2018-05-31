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
	router.GET("/api/forecast", forecast)
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