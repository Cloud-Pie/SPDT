package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"github.com/Cloud-Pie/SPDT/util"
	"errors"
	"github.com/Cloud-Pie/SPDT/storage"
)

/*
After each change in the workload it calculates the number of VMs of a predefined size needed
Repeat the process for all the vm types available
*/
type BestResourcePairPolicy struct {
	algorithm  string
	currentState	types.State
	sortedVMProfiles []types.VmProfile
	mapVMProfiles   map[string]types.VmProfile
	sysConfiguration	util.SystemConfiguration
}

/* Derive a list of policies using the Best Instance Approach approach
	in:
		@processedForecast
		@serviceProfile
	out:
		[] Policy. List of type Policy
*/
func (p BestResourcePairPolicy) CreatePolicies(processedForecast types.ProcessedForecast) [] types.Policy {
	log.Info("Derive policies with %s algorithm", p.algorithm)
	policies := []types.Policy{}
	selectedLimits, selectedVMProfile := p.findBestPair(processedForecast.CriticalIntervals )

	_, newPolicy := p.deriveCandidatePolicy(processedForecast.CriticalIntervals,selectedLimits, selectedVMProfile.Type)

	policies = append(policies, newPolicy)
	return policies
}

/*Calculate VM set able to host the required number of replicas
 in:
	@numberReplicas = Amount of replicas that should be hosted
	@resourcesLimit = Resources (CPU, Memory) constraints to configure the containers.
 out:
	@VMScale with the suggested number of VMs for that type
*/
func (p BestResourcePairPolicy) FindSuitableVMs(numberReplicas int, resourcesLimit types.Limit, vmType string) (types.VMScale, error) {
	vmScale := make(map[string]int)
	var err error
	profile := p.mapVMProfiles[vmType]
	maxReplicas := maxPodsCapacityInVM(profile, resourcesLimit)
	if maxReplicas > 0 {
		numVMs := math.Ceil(float64(numberReplicas) / float64(maxReplicas))
		vmScale[vmType] = int(numVMs)
	} else {
		return vmScale,errors.New("No suitable VM set found")
	}
	return vmScale,err
}

/*
	Derive a policy
*/
func (p BestResourcePairPolicy) deriveCandidatePolicy(criticalIntervals []types.CriticalInterval,
	podLimits types.Limit, vmType string ) (bool, types.Policy) {

	vmTypeSuitable := true
	scalingActions := []types.ScalingAction{}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}

	for _, it := range criticalIntervals {
		servicePerformanceProfile,_ := estimatePodsConfiguration(it.Requests, podLimits)
		vmSet,err := p.FindSuitableVMs(servicePerformanceProfile.MSCSetting.Replicas, servicePerformanceProfile.Limits, vmType)
		if err !=  nil {
			vmTypeSuitable = false
		}
		newNumPods := servicePerformanceProfile.MSCSetting.Replicas
		stateLoadCapacity := servicePerformanceProfile.MSCSetting.MSCPerSecond
		totalServicesBootingTime := servicePerformanceProfile.MSCSetting.BootTimeSec
		limits := servicePerformanceProfile.Limits

		services :=  make(map[string]types.ServiceInfo)
		services[ p.sysConfiguration.MainServiceName] = types.ServiceInfo {
			Scale:  newNumPods,
			CPU:    limits.CPUCores,
			Memory: limits.MemoryGB,
		}

		state := types.State{}
		state.Services = services
		state.VMs = vmSet
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		stateLoadCapacity = adjustGranularity(systemConfiguration.ForecastComponent.Granularity, stateLoadCapacity)
		setScalingSteps(&scalingActions,p.currentState, state,timeStart,timeEnd, totalServicesBootingTime, stateLoadCapacity)
		p.currentState = state
	}

	numScalingSteps := len(scalingActions)
	if vmTypeSuitable && numScalingSteps > 0 {
		//Add new policy
		parameters := make(map[string]string)
		parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
		parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(false)
		parameters[types.ISRESIZEPODS] = strconv.FormatBool(true)
		newPolicy.ScalingActions = scalingActions
		newPolicy.Algorithm = p.algorithm
		newPolicy.ID = bson.NewObjectId()
		newPolicy.Status = types.DISCARTED	//State by default
		newPolicy.Parameters = parameters
		newPolicy.Metrics.NumberScalingActions = numScalingSteps
		newPolicy.Metrics.FinishTimeDerivation = time.Now()
		newPolicy.Metrics.DerivationDuration = newPolicy.Metrics.FinishTimeDerivation.Sub(newPolicy.Metrics.StartTimeDerivation).Seconds()
		newPolicy.TimeWindowStart = scalingActions[0].TimeStart
		newPolicy.TimeWindowEnd = scalingActions[numScalingSteps-1].TimeEnd
	}
	return vmTypeSuitable, newPolicy
}

/* Finds the cheapest combination between container limits(cpu,mem) and vm type
	in:
		@forecastedValues
	out:
		Limit, string - Best pair found
*/
func (p BestResourcePairPolicy) findBestPair(forecastedValues []types.CriticalInterval ) (types.Limit, types.VmProfile){
	max := 0.0
	for _,v := range forecastedValues {
		if v.Requests > max {
			max = v.Requests
		}
	}
	biggestVMType := p.sortedVMProfiles[len(p.sortedVMProfiles)-1]
	allLimits,_ := storage.GetPerformanceProfileDAO(p.sysConfiguration.MainServiceName).FindAllUnderLimits(biggestVMType.CPUCores, biggestVMType.Memory)

	var bestLimit types.Limit
	var bestVMProfile types.VmProfile
	bestCost :=  math.Inf(1)
	numberReplicas := 999
	for vmType, _ := range p.mapVMProfiles {
		for _,vl := range allLimits {
			servicePerformanceProfile,_ := estimatePodsConfiguration(max, vl.Limit)
			replicas := servicePerformanceProfile.MSCSetting.Replicas
			vmSetCandidate,_ := p.FindSuitableVMs(replicas,vl.Limit, vmType)
			vmSetCost := 0.0
			if len(vmSetCandidate) > 0 {
				for _,v := range vmSetCandidate {
					vmSetCost += p.mapVMProfiles[vmType].Pricing.Price * float64(v)
				}
				if vmSetCost < bestCost {
					bestLimit = vl.Limit
					bestVMProfile = p.mapVMProfiles[vmType]
					bestCost = vmSetCost
					numberReplicas = replicas
				} else if vmSetCost == bestCost && replicas < numberReplicas{
					bestLimit = vl.Limit
					bestVMProfile = p.mapVMProfiles[vmType]
					bestCost = vmSetCost
					numberReplicas = replicas
				}
			}
		}
	}
	return bestLimit, bestVMProfile
}

