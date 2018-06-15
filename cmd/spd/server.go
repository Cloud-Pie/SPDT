package main

import (
	"github.com/gin-gonic/gin"
)

func SetUpServer() *gin.Engine{
	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20 // 8 MiB
	router.GET("/api/forecast", serverCall)
	return router
}


