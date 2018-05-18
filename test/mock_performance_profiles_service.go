package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"fmt"
	"os"
	"io/ioutil"
)

func main () {

	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20 // 8 MiB

	router.GET("/api/profiles", profiles)
	router.Run(":8082")
}

func profiles(c *gin.Context){
	// Open jsonFile
	jsonFile, err := os.Open("test/performance_profiles_test.json")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	c.JSON(http.StatusOK, byteValue)
}