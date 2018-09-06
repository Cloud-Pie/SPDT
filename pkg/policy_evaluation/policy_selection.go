package policy_evaluation
import (
	"github.com/Cloud-Pie/SPDT/types"
	"sort"
	"errors"
	"github.com/Cloud-Pie/SPDT/config"
	misc "github.com/Cloud-Pie/SPDT/pkg/policies_derivation"
	"math"
)

/*Evaluates and select the most suitable policy for the given system configurations and forecast
 in:
	@policies *[]types.Policy
				- List of derived policies
	@sysConfig config.SystemConfiguration
				- Configuration specified by the user in the config file
	@vmProfiles []types.VmProfile
				- List of virtual machines profiles
	@forecast types.Forecast
				- Forecast of the expected load
 out:
	@types.Policy
			- Selected policy
	@error
			- Error in case of any
*/
func SelectPolicy(policies *[]types.Policy, sysConfig config.SystemConfiguration, vmProfiles []types.VmProfile, forecast types.Forecast)(types.Policy, error) {

	mapVMProfiles := misc.VMListToMap(vmProfiles)
	//Calculate total cost of the policy
	for i := range *policies {
		cost := computePolicyCost((*policies)[i],sysConfig.PricingModel.BillingUnit, mapVMProfiles)
		(*policies)[i].Metrics.Cost = math.Ceil(cost*100)/100
	}
	//Sort policies based on price
	sort.Slice(*policies, func(i, j int) bool {
		return (*policies)[i].Metrics.Cost < (*policies)[j].Metrics.Cost
	})

	//Calculate Metrics of the policies
	for i := range *policies {
		over, under := computeMetricsCapacity(&(*policies)[i].Configurations,forecast.ForecastedValues)
		(*policies)[i].Metrics.OverProvision = math.Ceil(over*100)/100
		(*policies)[i].Metrics.UnderProvision = math.Ceil(under*100)/100

		numScaledContainers, numScaledVMS := computeMetricsScalingActions(&(*policies)[i].Configurations, sysConfig)
		(*policies)[i].Metrics.NumberContainerScalingActions = numScaledContainers
		(*policies)[i].Metrics.NumberVMScalingActions = numScaledVMS
	}

	if len(*policies) >0 {
		remainBudget, time := isEnoughBudget(sysConfig.PricingModel.Budget, (*policies)[0])
		if remainBudget {
			(*policies)[0].Status = types.SELECTED
			return (*policies)[0], nil
		} else {
			return (*policies)[0], errors.New("Budget is not enough for time window, you should increase the budget to ensure resources after " +time.String())
		}
	} else {
		return types.Policy{}, errors.New("No suitable policy found")
	}
}


//Calculate overprovisioning and underprovisioning of a state
func computeMetricsCapacity(configurations *[]types.ScalingConfiguration, forecast []types.ForecastedValue) (float64, float64){
	var avgOver float64
	var avgUnder float64
	fi := 0
	totalOver := 0.0
	totalUnder := 0.0
	numConfigurations := float64(len(*configurations))
	for i,_ := range *configurations {
		confOver := 0.0
		confUnder := 0.0
		numSamplesOver := 0.0
		numSamplesUnder := 0.0
		for  (*configurations)[i].TimeEnd.After(forecast[fi].TimeStamp){
			deltaLoad := (*configurations)[i].Metrics.CapacityTRN - forecast[fi].Requests
			if deltaLoad > 0 {
				confOver += deltaLoad*100.0/ forecast[fi].Requests
				numSamplesOver++
			} else if deltaLoad < 0 {
				confUnder += -1*deltaLoad*100.0/ forecast[fi].Requests
				numSamplesUnder++
			}
			fi++
		}
		if numSamplesUnder > 0 {
			(*configurations)[i].Metrics.UnderProvision = math.Ceil(confUnder /numSamplesUnder*100)/100
			totalUnder += confUnder /numSamplesUnder
		}
		if numSamplesOver > 0 {
			(*configurations)[i].Metrics.OverProvision = math.Ceil(confOver /numSamplesOver*100)/100
			totalOver += confOver /numSamplesOver
		}
	}
	avgOver = totalOver/numConfigurations
	avgUnder = totalUnder /numConfigurations
	return avgOver,avgUnder
}

/*Compute number of scaling steps per VM and containers
 in:
	@scalingActions *[]types.ScalingConfiguration
				- List of scaling actions or configurations
	@sysConfig config.SystemConfiguration
				- Configuration specified by the user in the config file
out:
	@int numberContainerScalingActions
		- Number of times where the containers where scaled
	@int numberVMScalingActions
		- Number of times where the vms where scaled
*/

func computeMetricsScalingActions (scalingActions *[]types.ScalingConfiguration, sysConfiguration config.SystemConfiguration) (int,int) {
	numberVMScalingActions := 0
	numberContainerScalingActions := 1
	numberScalingActions := len(*scalingActions)
	for i,_ := range *scalingActions {
		if i< numberScalingActions - 1{
			if !(*scalingActions)[i].State.VMs.Equal((*scalingActions)[i+1].State.VMs) {
				numberVMScalingActions += 1
			}
			serviceToScale := (*scalingActions)[i].State.Services[sysConfiguration.ServiceName]
			serviceScaled := (*scalingActions)[i+1].State.Services[sysConfiguration.ServiceName]
			if !serviceToScale.Equal(serviceScaled) {
				numberContainerScalingActions += 1
			}
		}
	}

	return numberContainerScalingActions, numberVMScalingActions
}