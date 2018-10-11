package server

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
func SetUpServer( fc chan types.Forecast ) *gin.Engine {
	forecastChannel = fc
	router := gin.Default()
	//router.Static("/assets", "./ui/assets")
	router.Static("/ui", "./ui")
	router.LoadHTMLGlob("ui/*.html")
	router.POST("/api/policies", serverCall)
	router.GET("/ui", homeUI)
	router.POST("/api/forecast", updateForecast)
	router.GET("/api/:service/policies/:id", policyByID)
	router.GET("/api/:service/policies", getPolicies)
	router.DELETE("/api/:service/policies/:id", deletePolicyByID)
	router.DELETE("/api/:service/policies", deletePolicyWindow)
	router.PUT("/api/:service/policies/:id", invalidatePolicyByID)
	router.GET("/api/:service/forecast", getForecast)


	return router
}

// This handler will match /api/:id/:service
// Retrieves information of a policy with the correspondent :id
func policyByID(c *gin.Context) {
	id := c.Param("id")
	serviceName := c.Param("service")
	policyDAO := db.GetPolicyDAO(serviceName)
	policyDAO.Connect()
	policy,err := policyDAO.FindByID(id)

	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	}
	c.JSON(http.StatusOK, policy)
}

// Delete policy with the correspondent :id
func deletePolicyByID(c *gin.Context) {
	id := c.Param("id")
	serviceName := c.Param("service")
	policyDAO := db.GetPolicyDAO(serviceName)
	err := policyDAO.DeleteById(id)

	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	}
	c.JSON(http.StatusOK,"Policy removed")
}

// This handler retrieve information of all policies that match the query paramenters
// The request responds to a policiesEndpoint matching:  /api/policies?start=2018-08-07T20:28:20&end=2018-08-07T20:28:20
func getPolicies(c *gin.Context) {
	windowTimeStart := c.DefaultQuery("start", "")
	windowTimeEnd := c.DefaultQuery("end","")
	serviceName := c.Param("service")
	policyDAO := db.GetPolicyDAO(serviceName)

	policies := []types.Policy{}
	if windowTimeStart == "" && windowTimeEnd == "" {
		policies,_ = policyDAO.FindAll()

	} else if windowTimeStart == "" && windowTimeEnd != "" {
		time, _ := time.Parse(util.UTC_TIME_LAYOUT, windowTimeEnd)
		policies,_ = policyDAO.FindByEndTime(time)

	} else if windowTimeStart != "" && windowTimeEnd == "" {
		time, _ := time.Parse(util.UTC_TIME_LAYOUT, windowTimeStart)
		policies,_ = policyDAO.FindByStartTime(time)

	} else if windowTimeStart != "" && windowTimeEnd != "" {
		startTime, _ := time.Parse(util.UTC_TIME_LAYOUT, windowTimeStart)
		endTime, _ := time.Parse(util.UTC_TIME_LAYOUT, windowTimeEnd)
		policies,_ = policyDAO.FindAllByTimeWindow(startTime,endTime)

	}
	if len(policies) == 0 {
		policies = make([]types.Policy,0)
	}
	c.JSON(http.StatusOK, policies)
}

// This handler delete policy that match the query parameters
// The request responds to a policiesEndpoint matching:  /api/policies?start=2018-08-07T20:28:20&end=2018-08-07T20:28:20
func deletePolicyWindow(c *gin.Context) {
	windowTimeStart := c.DefaultQuery("start", "")
	windowTimeEnd := c.DefaultQuery("end","")
	serviceName := c.Param("service")
	policyDAO := db.GetPolicyDAO(serviceName)

	if windowTimeStart != "" && windowTimeEnd != "" {
		startTime, err := time.Parse(util.UTC_TIME_LAYOUT, windowTimeStart)
		endTime, err := time.Parse(util.UTC_TIME_LAYOUT, windowTimeEnd)
		err = policyDAO.DeleteAllByTimeWindow(startTime,endTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		}
	}else {
		c.JSON(http.StatusBadRequest, "Missing parameters [start,end]")
	}
	c.JSON(http.StatusOK,"Policy removed")
}

// This handler will match /api/:id
// Invalidate policy with the correspondent :id
func invalidatePolicyByID(c *gin.Context) {
	id := c.Param("id")
	serviceName := c.Param("service")
	policyDAO := db.GetPolicyDAO(serviceName)
	policyDAO.Connect()
	err := policyDAO.DeleteById(id)

	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	}
	c.JSON(http.StatusOK,"Policy removed")
}

func serverCall(c *gin.Context) {
	sysConfiguration := ReadSysConfigurationFile("config.yml")
	selectedPolicy, err := StartPolicyDerivation(timeStart,timeEnd,"config.yml")

	if err != nil {
		c.JSON(http.StatusOK, err.Error())
	}else {
		ScheduleScaling(sysConfiguration,selectedPolicy)
		c.JSON(http.StatusOK, testJSON)
	}
}

//Listener to receive forecasting updates
func updateForecast(c *gin.Context) {
	forecast := &types.Forecast{}
	c.Bind(forecast)
	forecastChannel <- *forecast
	c.JSON(http.StatusOK,testJSON)
}

//This handler return the home page of the user interface
func homeUI(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

func getForecast(c *gin.Context) {
	windowTimeStart := c.DefaultQuery("start", "")
	windowTimeEnd := c.DefaultQuery("end","")
	serviceName := c.Param("service")

	type data struct {
		Timestamp 	[]time.Time
		Requests 	[]float64
	}

	forecastDAO := db.GetForecastDAO(serviceName)


	timestamps :=[]time.Time{}
	forecastedValues :=[]float64{}

	if windowTimeStart != "" && windowTimeEnd != "" {
		startTime, err := time.Parse(util.UTC_TIME_LAYOUT, windowTimeStart)
		endTime, err := time.Parse(util.UTC_TIME_LAYOUT, windowTimeEnd)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		} else {
			forecast,_ := forecastDAO.FindOneByTimeWindow(startTime, endTime)

			for _,v := range forecast.ForecastedValues {
				timestamps = append(timestamps, v.TimeStamp)
				forecastedValues = append(forecastedValues, v.Requests)
			}
			output := data{
				Timestamp:timestamps,
				Requests:forecastedValues,
			}
			c.JSON(http.StatusOK, output)
		}

	}else {
		c.JSON(http.StatusBadRequest, "Missing parameters [start,end]")
	}
}
