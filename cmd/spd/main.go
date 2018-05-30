package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"fmt"
	"github.com/Cloud-Pie/SPDT/pkg/performance_profiles"
	"github.com/Cloud-Pie/SPDT/pkg/forecast_processing"
	"github.com/Cloud-Pie/SPDT/pkg/policies_derivation"
	"github.com/Cloud-Pie/SPDT/pkg/policy_selection"
	"github.com/Cloud-Pie/SPDT/config"
	"github.com/Cloud-Pie/SPDT/internal/util"
	"log"
)

func main () {

	defaultLogFile    := util.DAFAULT_LOGFILE //TODO:Change to flag arg
	var Log *Logger = NewLogger()
	Log.SetLogFile(defaultLogFile)

	systemConfiguration,err := config.ParseConfigFile(util.CONFIG_FILE)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	forecastEndpoint := systemConfiguration.ForecastingComponent.Endpoint
	log.Print(forecastEndpoint)

	router := gin.Default()
	router.POST("/api/forecast", processForecast)
	router.Run(":8081")
}

func processForecast(c *gin.Context){
	file,err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf(err.Error()))
		return
	}
	if err := c.SaveUploadedFile (file, "pkg/forecast_processing/"+file.Filename); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Upload file err: %s. ", err.Error()))
		return
	}

	forecast := forecast_processing.ProcessData()
	if(!forecast.NeedToScale) {
		c.String(http.StatusOK,fmt.Sprintf("No need to scale in the next time window"))
	//write logs
	} else 	{
		vmProfiles := performance_profiles.GetPerformanceProfiles()
		policies := policies_derivation.Policies(forecast, vmProfiles)
		//currentState := CurrentState()
		policy := policy_selection.SelectPolicy(policies)

		//reconfiguration.TriggerScheduler(policy)
		c.JSON(http.StatusOK, policy)
	}
}