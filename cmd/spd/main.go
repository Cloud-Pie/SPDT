package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"fmt"
	"github.com/yemramirezca/SPDT/pkg/performance_profiles"
	"github.com/yemramirezca/SPDT/pkg/policies_derivation"
	"github.com/yemramirezca/SPDT/pkg/policy_selection"
	"github.com/yemramirezca/SPDT/pkg/reconfiguration"
	"github.com/yemramirezca/SPDT/pkg/forecast_processing"
)

func main () {
	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20 // 8 MiB

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
	c.String(http.StatusOK, fmt.Sprintf("File %s uploaded successfully.", file.Filename))


	forecast := forecast_processing.ProcessData()
	if(forecast.Need_to_scale) {
		vmProfiles := performance_profiles.GetPerformanceProfiles()
		policies := policies_derivation.CreatePolicies(forecast, vmProfiles)
		policy := policy_selection.SelectPolicy(policies)
		reconfiguration.TriggerScheduler(policy)
	} else {
		//No need to scale in the next time window.
		//write logs
	}
}