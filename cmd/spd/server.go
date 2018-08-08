package main

import (
	"github.com/gin-gonic/gin"
	db "github.com/Cloud-Pie/SPDT/storage"
	"net/http"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/types"
)

var forecastChannel chan types.Forecast

func SetUpServer( fc chan types.Forecast ) *gin.Engine{
	forecastChannel = fc
	router := gin.Default()
	router.GET("/api/forecast", serverCall)
	router.PUT("/api/forecast", updateForecast)
	router.GET("/api/policy", policy)
	router.GET("/api/policies", getPolicies)
	return router
}

func policy(c *gin.Context) {
	id := c.Param("id")

	policyDAO := db.PolicyDAO{
		Server:util.DEFAULT_DB_SERVER_POLICIES,
		Database:util.DEFAULT_DB_POLICIES,
	}
	policyDAO.Connect()
	policy,err := policyDAO.FindByID(id)

	if err != nil {
		log.Fatalf(err.Error())
	}
	c.JSON(http.StatusOK, policy)
}

func getPolicies(c *gin.Context) {
	policyDAO := db.PolicyDAO{
		Server:util.DEFAULT_DB_SERVER_POLICIES,
		Database:util.DEFAULT_DB_POLICIES,
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