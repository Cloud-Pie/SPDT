package main

import (
	"github.com/gin-gonic/gin"
	db "github.com/Cloud-Pie/SPDT/storage/policies"
	"net/http"
	"github.com/Cloud-Pie/SPDT/util"
	"log"
	"github.com/Cloud-Pie/SPDT/types"
)

var forecastChannel chan types.Forecast

func SetUpServer( fc chan types.Forecast ) *gin.Engine{
	forecastChannel = fc
	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20 // 8 MiB
	router.GET("/api/forecast", serverCall)
	router.PUT("/api/forecast", updateForecast)
	router.GET("/api/policy", policy)
	router.GET("/api/policies", policies)
	return router
}

func policy(c *gin.Context) {
	id := c.Param("id")

	policyDAO := db.PolicyDAO{
		util.DEFAULT_DB_SERVER_POLICIES,
		util.DEFAULT_DB_POLICIES,
	}
	policyDAO.Connect()
	policy,err := policyDAO.FindByID(id)

	if err != nil {
		log.Fatalf(err.Error())
	}
	c.JSON(http.StatusOK, policy)
}

func policies(c *gin.Context) {
	policyDAO := db.PolicyDAO{
		util.DEFAULT_DB_SERVER_POLICIES,
		util.DEFAULT_DB_POLICIES,
	}
	policyDAO.Connect()
	policies,err := policyDAO.FindAll()

	if err != nil {
		log.Fatalf(err.Error())
	}
	c.JSON(http.StatusOK, policies)
}

func serverCall(c *gin.Context) {
	startPolicyDerivation()
	c.JSON(http.StatusOK, "server")
}

func updateForecast(c *gin.Context) {
	forecast := &types.Forecast{}
	c.Bind(forecast)

	forecastChannel <- *forecast
	c.JSON(http.StatusOK,policies)
}