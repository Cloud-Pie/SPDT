package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
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
	currentState	types.State
	mapVMProfiles   map[string]types.VmProfile
	sysConfiguration	util.SystemConfiguration
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
	serviceToScale := p.currentState.Services[p.sysConfiguration.MainServiceName]
	currentPodLimits := types.Limit{CPUCores:serviceToScale.CPU, MemoryGB:serviceToScale.Memory}

	newPolicy := types.Policy{}
	state := types.State{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}

	scalingActions := []types.ScalingAction{}
	for _, it := range processedForecast.CriticalIntervals {
		var resourceLimits types.Limit
		//Select the performance profile that fits better
		containerConfigOver,_ := estimatePodsConfiguration(it.Requests, currentPodLimits)
		newNumPods := containerConfigOver.MSCSetting.Replicas
		vmSet := p.FindSuitableVMs(newNumPods, containerConfigOver.Limits)
		stateLoadCapacity := containerConfigOver.MSCSetting.MSCPerSecond
		totalServicesBootingTime := containerConfigOver.MSCSetting.BootTimeSec
		resourceLimits = containerConfigOver.Limits

		services := make(map[string]types.ServiceInfo)
		services[p.sysConfiguration.MainServiceName] = types.ServiceInfo{
			Scale:  newNumPods,
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
		stateLoadCapacity = adjustGranularity(systemConfiguration.ForecastComponent.Granularity, stateLoadCapacity)
		setScalingSteps(&scalingActions, p.currentState, state, timeStart, timeEnd, totalServicesBootingTime, stateLoadCapacity)
		p.currentState = state
	}

	parameters := make(map[string]string)
	parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
	parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(false)
	parameters[types.ISRESIZEPODS] = strconv.FormatBool(false)
	//Add new policy
	numConfigurations := len(scalingActions)
	newPolicy.ScalingActions = scalingActions
	newPolicy.Algorithm = p.algorithm
	newPolicy.ID = bson.NewObjectId()
	newPolicy.Status = types.DISCARTED //State by default
	newPolicy.Parameters = parameters
	newPolicy.Metrics.NumberScalingActions = numConfigurations
	newPolicy.Metrics.FinishTimeDerivation = time.Now()
	newPolicy.Metrics.DerivationDuration = newPolicy.Metrics.FinishTimeDerivation.Sub(newPolicy.Metrics.StartTimeDerivation).Seconds()
	newPolicy.TimeWindowStart = scalingActions[0].TimeStart
	newPolicy.TimeWindowEnd = scalingActions[numConfigurations-1].TimeEnd
	policies = append(policies, newPolicy)

	return policies
}

/*Calculate VM set able to host the required number of replicas
 in:
	@numberPods = Amount of pods that should be hosted
	@limits = Resources (CPU, Memory) constraints to configure the containers.
 out:
	@VMScale with the suggested number of VMs for that type
*/
func (p NaivePolicy) FindSuitableVMs(numberPods int, limits types.Limit) types.VMScale {
	vmScale := make(types.VMScale)
	vmType := p.currentVMType()
	profile := p.mapVMProfiles[vmType]
	podsCapacity := maxPodsCapacityInVM(profile, limits)
	if podsCapacity > 0 {
		numVMs := math.Ceil(float64(numberPods) / float64(podsCapacity))
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
