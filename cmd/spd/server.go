package main

import (
	"github.com/gin-gonic/gin"
	db "github.com/Cloud-Pie/SPDT/storage"
	"net/http"
	"github.com/Cloud-Pie/SPDT/util"
	"github.com/Cloud-Pie/SPDT/types"
	"time"
)

var forecastChannel chan types.Forecast

//Set up server routes
func SetUpServer( fc chan types.Forecast ) *gin.Engine{
	forecastChannel = fc
	router := gin.Default()
	router.GET("/api/forecast", serverCall)
	router.PUT("/api/forecast", updateForecast)
	router.GET("/api/policies/:id", policyByID)
	router.GET("/api/policies", getPolicies)
	router.DELETE("/api/policies/:id", deletePolicyByID)
	router.DELETE("/api/policies", deletePolicyWindow)
	return router
}

// This handler will match /api/:id
// Retrieves information of a policy with the correspondent :id
func policyByID(c *gin.Context) {
	id := c.Param("id")

	policyDAO := db.PolicyDAO{
		Server:util.DEFAULT_DB_SERVER_POLICIES,
		Database:util.DEFAULT_DB_POLICIES,
	}
	policyDAO.Connect()
	policy,err := policyDAO.FindByID(id)

	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	}
	c.JSON(http.StatusOK, policy)
}

// This handler will match /api/:id
// Delete policy with the correspondent :id
func deletePolicyByID(c *gin.Context) {
	id := c.Param("id")

	policyDAO := db.PolicyDAO{
		Server:util.DEFAULT_DB_SERVER_POLICIES,
		Database:util.DEFAULT_DB_POLICIES,
	}
	policyDAO.Connect()
	err := policyDAO.DeleteById(id)

	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	}
	c.JSON(http.StatusOK,"Policy removed")
}

// This handler retrieve information of all policies that match the query paramenters
// The request responds to a url matching:  /api/policies?start=2018-08-07T20:28:20&end=2018-08-07T20:28:20
func getPolicies(c *gin.Context) {
	windowTimeStart := c.DefaultQuery("start", "")
	windowTimeEnd := c.DefaultQuery("end","")

	policyDAO := db.PolicyDAO{
		Server:util.DEFAULT_DB_SERVER_POLICIES,
		Database:util.DEFAULT_DB_POLICIES,
	}
	policyDAO.Connect()

	var policies		[]types.Policy
	var err				error

	if windowTimeStart == "" && windowTimeEnd == "" {
		policies,err = policyDAO.FindAll()
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		}
	} else if windowTimeStart == "" && windowTimeEnd != "" {
		time, err := time.Parse(util.STD_TIME_LAYOUT, windowTimeEnd)
		policies,err = policyDAO.FindByEndTime(time)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		}
	} else if windowTimeStart != "" && windowTimeEnd == "" {
		time, err := time.Parse(util.STD_TIME_LAYOUT, windowTimeStart)
		policies,err = policyDAO.FindByStartTime(time)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		}
	} else if windowTimeStart != "" && windowTimeEnd != "" {
		startTime, err := time.Parse(util.STD_TIME_LAYOUT, windowTimeStart)
		endTime, err := time.Parse(util.STD_TIME_LAYOUT, windowTimeEnd)
		policies,err = policyDAO.FindAllByTimeWindow(startTime,endTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		}
	}
	c.JSON(http.StatusOK, policies)
}

// This handler delete policy that match the query paramenters
// The request responds to a url matching:  /api/policies?start=2018-08-07T20:28:20&end=2018-08-07T20:28:20
func deletePolicyWindow(c *gin.Context) {
	windowTimeStart := c.DefaultQuery("start", "")
	windowTimeEnd := c.DefaultQuery("end","")

	policyDAO := db.PolicyDAO{
		Server:util.DEFAULT_DB_SERVER_POLICIES,
		Database:util.DEFAULT_DB_POLICIES,
	}
	policyDAO.Connect()

	if windowTimeStart != "" && windowTimeEnd != "" {
		startTime, err := time.Parse(util.STD_TIME_LAYOUT, windowTimeStart)
		endTime, err := time.Parse(util.STD_TIME_LAYOUT, windowTimeEnd)
		err = policyDAO.DeleteOneByTimeWindow(startTime,endTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		}
	}else {
		c.JSON(http.StatusBadRequest, "Missing parameters [start,end]")
	}
	c.JSON(http.StatusOK,"Policy removed")
}

func serverCall(c *gin.Context) {
	startPolicyDerivation()
	c.JSON(http.StatusOK, "server")
}

//Listener to receive forecasting updates
func updateForecast(c *gin.Context) {
	forecast := &types.Forecast{}
	c.Bind(forecast)
	l := len(forecast.ForecastedValues)
	forecast.TimeWindowStart = forecast.ForecastedValues[0].TimeStamp
	forecast.TimeWindowEnd = forecast.ForecastedValues[l-1].TimeStamp
	forecastChannel <- *forecast
	c.JSON(http.StatusOK,policies)
}
