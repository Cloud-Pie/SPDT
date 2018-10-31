package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"gopkg.in/mgo.v2/bson"
	"sort"
	"strconv"
	"github.com/Cloud-Pie/SPDT/util"
)

/*
After each change in the workload it calculates the number of VMs of a predefined size needed
Repeat the process for all the vm types available
*/
type ResizeWhenBeneficialPolicy struct {
	algorithm        string              			 //Algorithm's name
	currentState     types.State          			//Current State
	sortedVMProfiles []types.VmProfile    			//List of VM profiles sorted by price
	mapVMProfiles    map[string]types.VmProfile		//Map with VM profiles with VM.Type as key
	sysConfiguration	util.SystemConfiguration
}

/* Derive a list of policies
   Add vmSet to handle delta load and compare the reconfiguration cost against the vmSet
   optimized for a total load.
	in:
		@processedForecast
		@serviceProfile
	out:
		[] Policy. List of type Policy
*/
func (p ResizeWhenBeneficialPolicy) CreatePolicies(processedForecast types.ProcessedForecast) [] types.Policy {
	log.Info("Derive policies with %s algorithm", p.algorithm)
	policies := []types.Policy{}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}
	configurations := []types.ScalingAction{}
	biggestVM := p.sortedVMProfiles[len(p.sortedVMProfiles)-1]
	vmLimits := types.Limit{ MemoryGB:biggestVM.Memory, CPUCores:biggestVM.CPUCores}

	for i, it := range processedForecast.CriticalIntervals {
		resourcesConfiguration := types.ContainersConfig{}

		//Load in terms of number of requests
		totalLoad := it.Requests
		serviceToScale := p.currentState.Services[p.sysConfiguration.MainServiceName]
		currentPodLimits := types.Limit{ MemoryGB:serviceToScale.Memory, CPUCores:serviceToScale.CPU }
		currentNumPods := serviceToScale.Scale
		currentLoadCapacity := getStateLoadCapacity(currentNumPods, currentPodLimits).MSCPerSecond
		deltaLoad := totalLoad - currentLoadCapacity

		if deltaLoad == 0 {
			//case 0: Keep current resource configuration
			resourcesConfiguration.VMSet = p.currentState.VMs
			resourcesConfiguration.Limits = currentPodLimits
			resourcesConfiguration.MSCSetting = types.MSCSimpleSetting{MSCPerSecond:currentLoadCapacity, Replicas: currentNumPods,}
		} else 	{
			if deltaLoad > 0 {
				//case 1: Need to increase resources
				rConfigDeltaLoad := p.onlyDeltaScaleOut(totalLoad, currentPodLimits)
				resourceConfigTLoad := p.resize(totalLoad, currentPodLimits, vmLimits)

				//Test if reconfigure the complete VM set for the totalLoad is better
				if newConfig,ok := p.shouldRepackVMSet(rConfigDeltaLoad, resourceConfigTLoad,i,processedForecast.CriticalIntervals); ok {
					resourcesConfiguration = newConfig
				} else {
					resourcesConfiguration = rConfigDeltaLoad
				}

			} else if deltaLoad < 0 {
				//case 2: Need to decrease resources
				rConfigDeltaLoad := p.onlyDeltaScaleIn(totalLoad, currentPodLimits, currentNumPods)
				resourceConfigTLoad := p.resize(totalLoad, currentPodLimits, vmLimits)

				//Test if reconfigure the complete VM set for the totalLoad is better
				if newConfig,ok := p.shouldRepackVMSet(rConfigDeltaLoad, resourceConfigTLoad,i,processedForecast.CriticalIntervals); ok {
					resourcesConfiguration = newConfig
				} else {
					resourcesConfiguration = rConfigDeltaLoad
				}
			}
		}
		services := make(map[string]types.ServiceInfo)
		services[p.sysConfiguration.MainServiceName] = types.ServiceInfo {
			Scale:  resourcesConfiguration.MSCSetting.Replicas,
			CPU:    resourcesConfiguration.Limits.CPUCores,
			Memory: resourcesConfiguration.Limits.MemoryGB,
		}

		//Create a new state
		state := types.State{}
		state.Services = services
		vmSet := resourcesConfiguration.VMSet
		cleanKeys(vmSet)
		state.VMs = vmSet
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		totalServicesBootingTime := resourcesConfiguration.MSCSetting.BootTimeSec
		stateLoadCapacity := resourcesConfiguration.MSCSetting.MSCPerSecond
		stateLoadCapacity = adjustGranularity(systemConfiguration.ForecastComponent.Granularity, stateLoadCapacity)
		setScalingSteps(&configurations,p.currentState, state,timeStart,timeEnd, totalServicesBootingTime,stateLoadCapacity)
		//Update current state
		p.currentState = state
	}

	//Add new policy
	parameters := make(map[string]string)
	parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
	parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(true)
	parameters[types.ISRESIZEPODS] = strconv.FormatBool(true)
	numConfigurations := len(configurations)
	newPolicy.ScalingActions = configurations
	newPolicy.Algorithm = p.algorithm
	newPolicy.ID = bson.NewObjectId()
	newPolicy.Status = types.DISCARTED	//State by default
	newPolicy.Parameters = parameters
	newPolicy.Metrics.FinishTimeDerivation = time.Now()
	newPolicy.TimeWindowStart = configurations[0].TimeStart
	newPolicy.TimeWindowEnd = configurations[numConfigurations -1].TimeEnd
	policies = append(policies, newPolicy)

	return policies
}

/*Calculate VM set able to host the required number of replicas
 in:
	@numberReplicas = Amount of replicas that should be hosted
	@resourceLimits = Resources (CPU, Memory) constraints to configure the containers.
 out:
	@VMScale with the suggested number of VMs
*/
func (p ResizeWhenBeneficialPolicy) FindSuitableVMs(numberReplicas int, resourceLimits types.Limit) types.VMScale {
	vmSet, _ := buildHomogeneousVMSet(numberReplicas,resourceLimits, p.mapVMProfiles)
	/*hetVMSet,_ := buildHeterogeneousVMSet(numberReplicas, resourceLimits, p.mapVMProfiles)
	costi := hetVMSet.Cost(p.mapVMProfiles)
	costj := vmSet.Cost(p.mapVMProfiles)
	if costi < costj {
		vmSet = hetVMSet
	}*/
	return vmSet
}

/* Remove the virtual machines that are supporting the deployment of pods from the current configuration.
 in:
	@vmSet = Current VM set
	@numberPods = Amount of pod replicas that should be hosted
	@limits = Resources (CPU, Memory) constraints to configure the containers.
 out:
	@VMScale with the suggested number of VMs
*/
func (p ResizeWhenBeneficialPolicy) releaseVMs(vmSet types.VMScale, numberPods int, limits types.Limit) types.VMScale {
	computeVMsCapacity(limits, &p.mapVMProfiles)

	var currentVMSet types.VMScale
	currentVMSet = copyMap(vmSet)
	newVMSet :=  make(map[string] int)

	type mapTypeCapacity struct {
		Key   string
		Value int
	}
	//Creates a list sorted by the number of machines per type
	var listMaps []mapTypeCapacity
	for k, v := range currentVMSet {
		listMaps = append(listMaps, mapTypeCapacity{k, v})
	}
	sort.Slice(listMaps, func(i, j int) bool { return listMaps[i].Value < listMaps[j].Value })

	for _,v := range listMaps {
		i:=0
		cap := p.mapVMProfiles[v.Key].ReplicasCapacity
		for i < v.Value && numberPods > 0{
			numberPods = numberPods - cap
			newVMSet[v.Key] = newVMSet[v.Key] + 1
			i+=1
		}
		if numberPods <= 0 {
			break
		}
	}

	return newVMSet
}

/* Calculate the cost of a reconfiguration
 in:
	@newSet =  Target VM set
 out:
	@float64 Cost of the transition from the  current vmSet to the newSet
*/
func(p ResizeWhenBeneficialPolicy) calculateReconfigurationCost(newSet types.VMScale) float64 {
	//Compute reconfiguration cost
	_, deletedVMS := DeltaVMSet(p.currentState.VMs, newSet)
	reconfigTime := computeVMTerminationTime(deletedVMS, p.sysConfiguration)

	return deletedVMS.Cost(p.mapVMProfiles) * float64(reconfigTime)
}



/* Evaluate if the current configuration of VMS should be changed to a new configuration
 in:
	@currentOption =  Configuration to handle extra delta load
	@candidateOption = Configuration to handle total load
	@indexTimeInterval = index in the time window
	@timeIntervals = forecasted values
 out:
	@ContainersConfig = chosen configuration
	@bool = flag to indicate whether reconfiguration should be performed
*/
func(p ResizeWhenBeneficialPolicy) shouldRepackVMSet(currentOption types.ContainersConfig, candidateOption types.ContainersConfig, indexTimeInterval int, timeIntervals[]types.CriticalInterval) (types.ContainersConfig, bool) {
	currentCost := currentOption.VMSet.Cost(p.mapVMProfiles)
	candidateCost := candidateOption.VMSet.Cost(p.mapVMProfiles)

	if candidateCost <= currentCost {
		//By default the transition policy would be to shut down VMs after launch new ones
		//Calculate reconfiguration time
		timeStart := timeIntervals[indexTimeInterval].TimeStart
		var timeEnd time.Time
		idx := indexTimeInterval
		lenInterval := len(timeIntervals)
		//Compute duration for new set
		candidateLoadCapacity := candidateOption.MSCSetting.MSCPerSecond
		for idx < lenInterval {
			if timeIntervals[idx].Requests > candidateLoadCapacity {
				timeEnd = timeIntervals[idx].TimeStart
				break
			}
			idx+=1
		}
		durationNewVMSet :=  timeEnd.Sub(timeStart).Seconds()
		reconfigCostNew := p.calculateReconfigurationCost(candidateOption.VMSet)

		//Compute duration for current set
		jdx := indexTimeInterval
		currentLoadCapacity := currentOption.MSCSetting.MSCPerSecond
		for jdx < lenInterval {
			if timeIntervals[jdx].Requests > currentLoadCapacity{
				timeEnd = timeIntervals[jdx].TimeStart
				break
			}
			jdx+=1
		}
		durationCurrentVMSet :=  timeEnd.Sub(timeStart).Seconds()
		reconfigCostCurrent := p.calculateReconfigurationCost(currentOption.VMSet)

		if candidateCost*durationNewVMSet + reconfigCostNew < currentCost * durationCurrentVMSet + reconfigCostCurrent {
			return candidateOption, true
		}
	}
	return types.ContainersConfig{}, false
}

/* Compute the resource configuration to handle extra delta load
 in:
	@totalLoad =  Configuration to handle extra delta load
	@currentPodLimits = Pod limits used in current configuration
 out:
	@ContainersConfig = Computed configuration
*/
func (p ResizeWhenBeneficialPolicy) onlyDeltaScaleOut(totalLoad float64, currentPodLimits types.Limit) types.ContainersConfig {
	var vmSet types.VMScale
	containersResourceConfig :=  types.ContainersConfig{}

	profileCurrentLimits,_ := estimatePodsConfiguration(totalLoad, currentPodLimits)
	computeVMsCapacity(currentPodLimits, &p.mapVMProfiles)
	currentPodsCapacity := p.currentState.VMs.ReplicasCapacity(p.mapVMProfiles)
	newNumPods := profileCurrentLimits.MSCSetting.Replicas

	if currentPodsCapacity >= newNumPods {
		vmSet = p.currentState.VMs
	} else {
		deltaNumPods := newNumPods - currentPodsCapacity
		vmSet = p.FindSuitableVMs(deltaNumPods, profileCurrentLimits.Limits)
		vmSet.Merge(p.currentState.VMs)
	}

	containersResourceConfig.VMSet = vmSet
	containersResourceConfig.Limits = profileCurrentLimits.Limits
	containersResourceConfig.MSCSetting = profileCurrentLimits.MSCSetting

	return  containersResourceConfig
}

/* Compute the resource configuration to remove resources given a negative delta load
 in:
	@totalLoad =  Configuration to handle extra delta load
	@currentPodLimits = Pod limits used in current configuration
	@currentNumPods = Number of pods that should be hosted
 out:
	@ContainersConfig = Computed configuration
*/
func (p ResizeWhenBeneficialPolicy) onlyDeltaScaleIn(totalLoad float64, currentPodLimits types.Limit, currentNumPods int) types.ContainersConfig {
	var vmSet types.VMScale
	containersResourceConfig :=  types.ContainersConfig{}

	profileCurrentLimits,_ := estimatePodsConfiguration(totalLoad, currentPodLimits)
	newNumPods := profileCurrentLimits.MSCSetting.Replicas
	deltaNumPods := currentNumPods - newNumPods
	if deltaNumPods > 0 {
		vmSet = p.releaseVMs(p.currentState.VMs,newNumPods, currentPodLimits)
	} else {
		vmSet = p.currentState.VMs
	}
	containersResourceConfig.VMSet = vmSet
	containersResourceConfig.Limits = profileCurrentLimits.Limits
	containersResourceConfig.MSCSetting = profileCurrentLimits.MSCSetting

	return  containersResourceConfig
}

/* Compute the resource configuration to by changing pod configurations and VM types
 in:
	@totalLoad =  Configuration to handle extra delta load
	@currentPodLimits = Pod limits used in current configuration
	@vmLimits = Limits of the biggest VM available
 out:
	@ContainersConfig = Computed configuration
*/
func (p ResizeWhenBeneficialPolicy) resize(totalLoad float64, currentPodLimits types.Limit, vmLimits types.Limit) types.ContainersConfig{
	containersResourceConfig :=  types.ContainersConfig{}
	performanceProfile, _ := selectProfileUnderVMLimits(totalLoad, vmLimits)
	vmSet := p.FindSuitableVMs(performanceProfile.MSCSetting.Replicas, performanceProfile.Limits)

	containersResourceConfig.VMSet = vmSet
	containersResourceConfig.Limits = performanceProfile.Limits
	containersResourceConfig.MSCSetting = performanceProfile.MSCSetting

	return  containersResourceConfig
}