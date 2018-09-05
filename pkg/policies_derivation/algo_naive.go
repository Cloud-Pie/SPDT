package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/config"
	"time"
	"strconv"
	"gopkg.in/mgo.v2/bson"
	"math"
	"github.com/Cloud-Pie/SPDT/util"
)

/*
It assumes that the current VM set where the microservice is deployed is a homogeneous set
Based on the unique VM type and its capacity to host a number of replicas it increases or decreases the number of VMs
*/
type NaivePolicy struct {
	algorithm  		string               //Algorithm's name
	timeWindow 		TimeWindowDerivation //Algorithm used to process the forecasted time serie
	currentState	types.State			 //Current State
	mapVMProfiles   map[string]types.VmProfile
	sysConfiguration	config.SystemConfiguration

}

/* Derive a list of policies using the Naive approach
	in:
		@processedForecast
		@serviceProfile
	out:
		[] Policy. List of type Policy
*/
func (p NaivePolicy) CreatePolicies(processedForecast types.ProcessedForecast) []types.Policy {
	policies := []types.Policy {}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}

	configurations := []types.ScalingConfiguration{}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	percentageUnderProvision := p.sysConfiguration.PolicySettings.MaxUnderprovisionPercentage
	serviceToScale := p.currentState.Services[p.sysConfiguration.ServiceName]
	currentContainerLimits := types.Limit{ MemoryGB:serviceToScale.Memory, CPUCores:serviceToScale.CPU }

	for _, it := range processedForecast.CriticalIntervals {
		var resourceLimits types.Limit

		//Select the performance profile that fits better
		containerConfigOver := selectProfileWithLimits(it.Requests, currentContainerLimits, false)
		newNumServiceReplicas := containerConfigOver.TRNConfiguration.NumberReplicas
		vmSet := p.FindSuitableVMs(newNumServiceReplicas, containerConfigOver.Limits)
		costOver := vmSet.Cost(p.mapVMProfiles)
		stateLoadCapacity := containerConfigOver.TRNConfiguration.TRN
		totalServicesBootingTime := containerConfigOver.TRNConfiguration.BootTimeSec
		resourceLimits = containerConfigOver.Limits

		if underProvisionAllowed {
			containerConfigUnder := selectProfileWithLimits(it.Requests, currentContainerLimits, underProvisionAllowed)
			vmSetUnder := p.FindSuitableVMs(containerConfigUnder.TRNConfiguration.NumberReplicas, containerConfigUnder.Limits)
			costUnder := vmSetUnder.Cost(p.mapVMProfiles)
			//Update values if the configuration that leads to under provisioning is cheaper
			if costUnder < costOver && isUnderProvisionInRange(it.Requests, containerConfigUnder.TRNConfiguration.TRN, percentageUnderProvision){
				vmSet = vmSetUnder
				newNumServiceReplicas = containerConfigUnder.TRNConfiguration.NumberReplicas
				stateLoadCapacity = containerConfigUnder.TRNConfiguration.TRN
				totalServicesBootingTime = containerConfigUnder.TRNConfiguration.BootTimeSec
				resourceLimits = containerConfigUnder.Limits
			}
		}

		services := make(map[string]types.ServiceInfo)
		services[p.sysConfiguration.ServiceName] = types.ServiceInfo{
			Scale:  newNumServiceReplicas,
			CPU:    resourceLimits.CPUCores,
			Memory: resourceLimits.MemoryGB,
		}
		state := types.State{
			Services: services,
			VMs:      vmSet,
		}

		//update state before next iteration
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		setConfiguration(&configurations, state, timeStart, timeEnd, p.sysConfiguration.ServiceName, totalServicesBootingTime, p.sysConfiguration, stateLoadCapacity)
	}
	parameters := make(map[string]string)
	parameters[types.VMTYPES] = p.currentVMType()
	parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
	parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(false)
	parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)
	parameters[types.ISRESIZEPODS] = strconv.FormatBool(false)
	//Add new policy
	numConfigurations := len(configurations)
	newPolicy.Configurations = configurations
	newPolicy.Algorithm = p.algorithm
	newPolicy.ID = bson.NewObjectId()
	newPolicy.Status = types.DISCARTED	//State by default
	newPolicy.Parameters = parameters
	newPolicy.Metrics.NumberConfigurations = numConfigurations
	newPolicy.Metrics.FinishTimeDerivation = time.Now()
	newPolicy.Metrics.DerivationDuration = newPolicy.Metrics.FinishTimeDerivation.Sub(newPolicy.Metrics.StartTimeDerivation).Seconds()
	newPolicy.TimeWindowStart = configurations[0].TimeStart
	newPolicy.TimeWindowEnd = configurations[numConfigurations -1].TimeEnd
	policies = append(policies, newPolicy)


	return policies
}

/*Calculate VM set able to host the required number of replicas
 in:
	@numberReplicas = Amount of replicas that should be hosted
	@limits = Resources (CPU, Memory) constraints to configure the containers.
 out:
	@VMScale with the suggested number of VMs for that type
*/
func (p NaivePolicy) FindSuitableVMs(numberReplicas int, limits types.Limit) types.VMScale {
	vmScale := make(types.VMScale)
	vmType := p.currentVMType()
	profile := p.mapVMProfiles[vmType]
	maxReplicas := maxReplicasCapacityInVM(profile, limits)
	if maxReplicas > 0 {
		numVMs := math.Ceil(float64(numberReplicas) / float64(maxReplicas))
		vmScale[vmType] = int(numVMs)
	}
	return vmScale
}

/*Return the VM type used by the current Homogeneous VM cluster
	out:
		String with the name of the current VM type
*/
func (p NaivePolicy) currentVMType() string {
	//Assumption for p approach: There is only 1 vm Type in current state
	var vmType string
	for k := range p.currentState.VMs {
		vmType = k
	}
	if len(p.currentState.VMs) > 1 {
		log.Warning("Current config has more than one VM type, type %s was selected to continue", vmType)
	}
	return vmType
}
