package main

import (
	"github.com/gin-gonic/gin"
)

func SetUpServer() *gin.Engine{
	router := gin.Default()
	router.POST("/api/forecast", processForecast)
	return router
}

func processForecast(c *gin.Context){

}

