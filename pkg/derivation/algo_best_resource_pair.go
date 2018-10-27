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
	"sort"
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
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	containerResizeEnabled := p.sysConfiguration.PolicySettings.PodsResizeAllowed
	percentageUnderProvision := p.sysConfiguration.PolicySettings.MaxUnderprovisionPercentage
	selectedLimits, selectedVMType := p.findBestPair(processedForecast.CriticalIntervals )

	vm := p.mapVMProfiles[selectedVMType]
	vmLimits := types.Limit{ MemoryGB:vm.Memory, CPUCores:vm.CPUCores}
	_, newPolicy := p.deriveCandidatePolicy(processedForecast.CriticalIntervals,containerResizeEnabled, selectedLimits, vmLimits, selectedVMType, underProvisionAllowed, percentageUnderProvision )

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
	maxReplicas := maxReplicasCapacityInVM(profile, resourcesLimit)
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
func (p BestResourcePairPolicy) deriveCandidatePolicy(criticalIntervals []types.CriticalInterval, containerResizeEnabled bool,
	containerLimits types.Limit, vmLimits types.Limit, vmType string, underProvisionAllowed bool,percentageUnderProvision float64 ) (bool, types.Policy) {

	vmTypeSuitable := true
	scalingSteps := []types.ScalingStep{}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}

	for _, it := range criticalIntervals {
		servicePerformanceProfile,_ := estimatePodsConfiguration(it.Requests, containerLimits)
		vmSet,err := p.FindSuitableVMs(servicePerformanceProfile.MSCSetting.Replicas, servicePerformanceProfile.Limits, vmType)
		if err !=  nil {
			vmTypeSuitable = false
		}

		costVMSetOverProvision := vmSet.Cost(p.mapVMProfiles)
		//Update values if the configuration that leads to under provisioning is cheaper
		if underProvisionAllowed {
			containerConfigUnder := p.optionWithUnderProvision(it.Requests, containerLimits, percentageUnderProvision, vmType)
			if containerConfigUnder.Cost >0 && containerConfigUnder.Cost < costVMSetOverProvision &&
				isUnderProvisionInRange(it.Requests, containerConfigUnder.MSCSetting.MSCPerSecond, percentageUnderProvision) {
				vmSet = containerConfigUnder.VMSet
				servicePerformanceProfile = containerConfigUnder
			}
		}

		newNumServiceReplicas := servicePerformanceProfile.MSCSetting.Replicas
		stateLoadCapacity := servicePerformanceProfile.MSCSetting.MSCPerSecond
		totalServicesBootingTime := servicePerformanceProfile.MSCSetting.BootTimeSec
		limits := servicePerformanceProfile.Limits

		services :=  make(map[string]types.ServiceInfo)
		services[ p.sysConfiguration.MainServiceName] = types.ServiceInfo {
			Scale:  newNumServiceReplicas,
			CPU:    limits.CPUCores,
			Memory: limits.MemoryGB,
		}

		state := types.State{}
		state.Services = services
		state.VMs = vmSet
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		stateLoadCapacity = adjustGranularity(systemConfiguration.ForecastComponent.Granularity, stateLoadCapacity)
		setScalingSteps(&scalingSteps,p.currentState, state,timeStart,timeEnd, totalServicesBootingTime, stateLoadCapacity)
		p.currentState = state
	}

	numScalingSteps := len(scalingSteps)
	if vmTypeSuitable && numScalingSteps > 0 {
		//Add new policy
		parameters := make(map[string]string)
		parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
		parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(false)
		parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)
		parameters[types.ISRESIZEPODS] = strconv.FormatBool(containerResizeEnabled)
		newPolicy.ScalingActions = scalingSteps
		newPolicy.Algorithm = p.algorithm
		newPolicy.ID = bson.NewObjectId()
		newPolicy.Status = types.DISCARTED	//State by default
		newPolicy.Parameters = parameters
		newPolicy.Metrics.NumberScalingActions = numScalingSteps
		newPolicy.Metrics.FinishTimeDerivation = time.Now()
		newPolicy.Metrics.DerivationDuration = newPolicy.Metrics.FinishTimeDerivation.Sub(newPolicy.Metrics.StartTimeDerivation).Seconds()
		newPolicy.TimeWindowStart = scalingSteps[0].TimeStart
		newPolicy.TimeWindowEnd = scalingSteps[numScalingSteps-1].TimeEnd
	}
	return vmTypeSuitable, newPolicy
}

/*Search a container configuration (limits & n replicas) which has under provision but leads to a cheaper VM set
	in:
		@totalLoad - float64 = Number of requests
		@containerLimits -Limit = Resource limits for the container
		@percentageUnderProvision -float64 = percentage of the max under provision allowed
	out:
		ContainersConfig -
*/
func (p BestResourcePairPolicy) optionWithUnderProvision(totalLoad float64, containerLimits types.Limit, percentageUnderProvision float64, vmType string) types.ContainersConfig {
	containerConfigUnder,_ := estimatePodsConfiguration(totalLoad, containerLimits)
	vmSetUnder,_ := p.FindSuitableVMs(containerConfigUnder.MSCSetting.Replicas, containerConfigUnder.Limits, vmType)
	costVMSetUnderProvision := vmSetUnder.Cost(p.mapVMProfiles)
	containerConfigUnder.VMSet = vmSetUnder
	containerConfigUnder.Cost = costVMSetUnderProvision
	return  containerConfigUnder
}

/* Finds the cheapest combination between container limits(cpu,mem) and vm type
	in:
		@forecastedValues
	out:
		Limit, string - Best pair found
*/
func (p BestResourcePairPolicy) findBestPair(forecastedValues []types.CriticalInterval ) (types.Limit, string){
	max := 0.0
	for _,v := range forecastedValues {
		if v.Requests > max {
			max = v.Requests
		}
	}

	vmProfiles := p.sortedVMProfiles
	sort.Slice(vmProfiles, func(i, j int) bool {
		return vmProfiles[i].Pricing.Price >  vmProfiles[j].Pricing.Price
	})
	biggestVMType := p.mapVMProfiles[vmProfiles[0].Type]
	allLimits,_ := storage.GetPerformanceProfileDAO(p.sysConfiguration.MainServiceName).FindAllUnderLimits(biggestVMType.CPUCores, biggestVMType.Memory)

	var bestLimit types.Limit
	var bestType string
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
					bestType = vmType
					bestCost = vmSetCost
					numberReplicas = replicas
				} else if vmSetCost == bestCost && replicas < numberReplicas{
					bestLimit = vl.Limit
					bestType = vmType
					bestCost = vmSetCost
					numberReplicas = replicas
				}
			}
		}
	}
	return bestLimit,bestType
}

