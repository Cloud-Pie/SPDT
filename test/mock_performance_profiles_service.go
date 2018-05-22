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
	router.MaxMultipartMemory = 8 << 20 // 8 MiB

	router.GET("/api/profiles", profiles)
	router.Run(":8082")
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