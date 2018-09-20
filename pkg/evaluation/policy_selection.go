package evaluation
import (
	"github.com/Cloud-Pie/SPDT/types"
	"sort"
	"errors"
	"github.com/Cloud-Pie/SPDT/config"
	misc "github.com/Cloud-Pie/SPDT/pkg/derivation"
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
		nForecast := len(forecast.ForecastedValues)
		over, under := computeMetricsCapacity(&(*policies)[i].ScalingActions,forecast.ForecastedValues, nForecast)
		(*policies)[i].Metrics.OverProvision = math.Ceil(over*100)/100
		(*policies)[i].Metrics.UnderProvision = math.Ceil(under*100)/100

		numScaledContainers, numScaledVMS, vmTypes := computeMetricsScalingActions(&(*policies)[i].ScalingActions, mapVMProfiles, sysConfig)
		(*policies)[i].Metrics.NumberContainerScalingActions = numScaledContainers
		(*policies)[i].Metrics.NumberVMScalingActions = numScaledVMS
		(*policies)[i].Parameters[types.VMTYPES] = misc.MapKeystoString(vmTypes)

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
func computeMetricsCapacity(configurations *[]types.ScalingStep, forecast []types.ForecastedValue, nForecastedValues int) (float64, float64){
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
		for  fi < nForecastedValues && (*configurations)[i].TimeEnd.After(forecast[fi].TimeStamp) {
			deltaLoad := (*configurations)[i].Metrics.RequestsCapacity - forecast[fi].Requests
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
	@scalingActions *[]types.ScalingStep
				- List of scaling actions or configurations
	@sysConfig config.SystemConfiguration
				- Configuration specified by the user in the config file
out:
	@int numberContainerScalingActions
		- Number of times where the containers where scaled
	@int numberVMScalingActions
		- Number of times where the vms where scaled
	@map[string]bool vmTypes
		- Map with the vm types used in the scaling actions
*/

func computeMetricsScalingActions (scalingActions *[]types.ScalingStep, mapVMProfiles map[string] types.VmProfile,  sysConfiguration config.SystemConfiguration) (int,int, map[string]bool) {
	numberVMScalingActions := 0
	numberContainerScalingActions := 1
	numberScalingActions := len(*scalingActions)
	vmTypes := make(map[string] bool)

	for i,_ := range *scalingActions {
		vmSetToScale := (*scalingActions)[i].State.VMs
		serviceToScale := (*scalingActions)[i].State.Services[sysConfiguration.ServiceName]

		if i< numberScalingActions - 1 {
			vmSetScaled := (*scalingActions)[i+1].State.VMs
			if !vmSetToScale.Equal(vmSetScaled) {
				numberVMScalingActions += 1
			}
			serviceScaled := (*scalingActions)[i+1].State.Services[sysConfiguration.ServiceName]
			if !serviceToScale.Equal(serviceScaled) {
				numberContainerScalingActions += 1
			}
		}
		totalCPUCores := 0.0
		totalMemGB := 0.0
		for k,v := range vmSetToScale {
			vmTypes[k] = true
			totalCPUCores += mapVMProfiles[k].CPUCores * float64(v)
			totalMemGB += mapVMProfiles[k].Memory * float64(v)
		}
		if totalMemGB > 0 {
			percentageMemUtilization := serviceToScale.Memory * float64(serviceToScale.Scale) * 100.0 / totalMemGB
			(*scalingActions)[i].Metrics.MemoryUtilization = percentageMemUtilization
		}
		if totalCPUCores > 0 {
			percentageCPUUtilization := serviceToScale.CPU * float64(serviceToScale.Scale)  * 100.0 / totalCPUCores
			(*scalingActions)[i].Metrics.CPUUtilization = percentageCPUUtilization
		}

		if i >= 1 {
			previousStateEndTime := (*scalingActions)[i-1].TimeEnd
			(*scalingActions)[i].Metrics.ShadowTimeSec = previousStateEndTime.Sub((*scalingActions)[i].TimeStart).Seconds()
		}

	}
	return numberContainerScalingActions, numberVMScalingActions, vmTypes
}

