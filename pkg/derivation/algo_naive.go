package derivation

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
	algorithm  		string
	timeWindow 		TimeWindowDerivation
	currentState	types.State
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
	log.Info("Derive policies with %s algorithm", p.algorithm)
	policies := []types.Policy {}

	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	percentageUnderProvision := p.sysConfiguration.PolicySettings.MaxUnderprovisionPercentage
	serviceToScale := p.currentState.Services[p.sysConfiguration.MainServiceName]
	currentContainerLimits := types.Limit{CPUCores:serviceToScale.CPU, MemoryGB:serviceToScale.Memory}

	newPolicy := types.Policy{}
	state := types.State{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}

	configurations := []types.ScalingStep{}
	for _, it := range processedForecast.CriticalIntervals {
		var resourceLimits types.Limit
		//Select the performance profile that fits better
		containerConfigOver := selectProfileByLimits(it.Requests, currentContainerLimits, false)
		newNumServiceReplicas := containerConfigOver.MSCSetting.Replicas
		vmSet := p.FindSuitableVMs(newNumServiceReplicas, containerConfigOver.Limits)
		costVMSetOverProvision := vmSet.Cost(p.mapVMProfiles)
		stateLoadCapacity := containerConfigOver.MSCSetting.MSCPerSecond
		totalServicesBootingTime := containerConfigOver.MSCSetting.BootTimeSec
		resourceLimits = containerConfigOver.Limits

		//Update values if the configuration that leads to under provisioning is cheaper
		if underProvisionAllowed {
			containerConfigUnder := p.optionWithUnderProvision(it.Requests, currentContainerLimits, percentageUnderProvision)
			if containerConfigUnder.Cost >0 && containerConfigUnder.Cost < costVMSetOverProvision &&
				isUnderProvisionInRange(it.Requests, containerConfigUnder.MSCSetting.MSCPerSecond, percentageUnderProvision) {
				vmSet = containerConfigUnder.VMSet
				newNumServiceReplicas = containerConfigUnder.MSCSetting.Replicas
				stateLoadCapacity = containerConfigUnder.MSCSetting.MSCPerSecond
				totalServicesBootingTime = containerConfigUnder.MSCSetting.BootTimeSec
				resourceLimits = containerConfigUnder.Limits
			}
		}

		services := make(map[string]types.ServiceInfo)
		services[p.sysConfiguration.MainServiceName] = types.ServiceInfo{
			Scale:  newNumServiceReplicas,
			CPU:    resourceLimits.CPUCores,
			Memory: resourceLimits.MemoryGB,
		}
		state = types.State{
			Services: services,
			VMs:      vmSet,
		}

		//update state before next iteration
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		setScalingSteps(&configurations, p.currentState, state, timeStart, timeEnd, totalServicesBootingTime, stateLoadCapacity)
		p.currentState = state
	}

	parameters := make(map[string]string)
	parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
	parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(false)
	parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)
	parameters[types.ISRESIZEPODS] = strconv.FormatBool(false)
	//Add new policy
	numConfigurations := len(configurations)
	newPolicy.ScalingActions = configurations
	newPolicy.Algorithm = p.algorithm
	newPolicy.ID = bson.NewObjectId()
	newPolicy.Status = types.DISCARTED //State by default
	newPolicy.Parameters = parameters
	newPolicy.Metrics.NumberScalingActions = numConfigurations
	newPolicy.Metrics.FinishTimeDerivation = time.Now()
	newPolicy.Metrics.DerivationDuration = newPolicy.Metrics.FinishTimeDerivation.Sub(newPolicy.Metrics.StartTimeDerivation).Seconds()
	newPolicy.TimeWindowStart = configurations[0].TimeStart
	newPolicy.TimeWindowEnd = configurations[numConfigurations-1].TimeEnd
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
	//It selects teh VM with more resources in case there is more than onw vm type
	var vmType string
	memGB := 0.0
	for k,_ := range p.currentState.VMs {
		if p.mapVMProfiles[k].Memory > memGB {
			vmType = k
			memGB =  p.mapVMProfiles[k].Memory
		}
	}
	if len(p.currentState.VMs) > 1 {
		log.Warning("Current config has more than one VM type, type %s was selected to continue", vmType)
	}
	return vmType
}

/*Search a container configuration (limits & n replicas) which has under provision but leads to a cheaper VM set
	in:
		@totalLoad - float64 = Number of requests
		@containerLimits -Limit = Resource limits for the container
		@percentageUnderProvision -float64 = percentage of the max under provision allowed
	out:
		ContainersConfig -
*/
func (p NaivePolicy) optionWithUnderProvision(totalLoad float64, containerLimits types.Limit, percentageUnderProvision float64) types.ContainersConfig {
	containerConfigUnder := selectProfileByLimits(totalLoad, containerLimits, true)
	vmSetUnder := p.FindSuitableVMs(containerConfigUnder.MSCSetting.Replicas, containerConfigUnder.Limits)
	costVMSetUnderProvision := vmSetUnder.Cost(p.mapVMProfiles)
	containerConfigUnder.VMSet = vmSetUnder
	containerConfigUnder.Cost = costVMSetUnderProvision
	return  containerConfigUnder
}
