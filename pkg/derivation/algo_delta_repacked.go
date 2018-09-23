package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"gopkg.in/mgo.v2/bson"
	"math"
	"sort"
	"github.com/Cloud-Pie/SPDT/config"
	"strconv"
	"github.com/Cloud-Pie/SPDT/util"
)

/*
After each change in the workload it calculates the number of VMs of a predefined size needed
Repeat the process for all the vm types available
*/
type DeltaRepackedPolicy struct {
	algorithm        string              			 //Algorithm's name
	timeWindow       TimeWindowDerivation 			//Algorithm used to process the forecasted time serie
	currentState     types.State          			//Current State
	sortedVMProfiles []types.VmProfile    			//List of VM profiles sorted by price
	mapVMProfiles    map[string]types.VmProfile		//Map with VM profiles with VM.Type as key
	sysConfiguration	config.SystemConfiguration
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
func (p DeltaRepackedPolicy) CreatePolicies(processedForecast types.ProcessedForecast) [] types.Policy {
	log.Info("Derive policies with %s algorithm", p.algorithm)
	policies := []types.Policy{}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}
	configurations := []types.ScalingStep{}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	//containerResizeEnabled := p.sysConfiguration.PolicySettings.PodsResizeAllowed
	//percentageUnderProvision := p.sysConfiguration.PolicySettings.MaxUnderprovisionPercentage

	for i, it := range processedForecast.CriticalIntervals {
		resourcesConfiguration := types.ContainersConfig{}

		//Load in terms of number of requests
		totalLoad := it.Requests
		serviceToScale := p.currentState.Services[p.sysConfiguration.MainServiceName]
		currentContainerLimits := types.Limit{ MemoryGB:serviceToScale.Memory, CPUCores:serviceToScale.CPU }
		currentNumberReplicas := serviceToScale.Scale
		currentLoadCapacity := getStateLoadCapacity(currentNumberReplicas,currentContainerLimits)
		deltaLoad := totalLoad - currentLoadCapacity

		if deltaLoad == 0 {
			//case 0: Keep current resource configuration
			resourcesConfiguration.VMSet = p.currentState.VMs
			resourcesConfiguration.Limits = currentContainerLimits
			resourcesConfiguration.MSCSetting = types.MSCSimpleSetting{MSCPerSecond:currentLoadCapacity, Replicas:currentNumberReplicas,}
		} else 	{
			if deltaLoad > 0 {
				//case 1: Need to increase resources
				//Candidate option to handle total load with current limits
				var onlyScaleContainers bool
				profileCurrentLimits, onlyScaleContainers := p.onlyScaleOutContainers(p.currentState.VMs, totalLoad, currentContainerLimits)
				if onlyScaleContainers {
					//case 1: Only scale containers but keep the same VM set
					resourcesConfiguration.VMSet = p.currentState.VMs
					resourcesConfiguration.Limits = profileCurrentLimits.Limits
					resourcesConfiguration.MSCSetting = types.MSCSimpleSetting{MSCPerSecond:profileCurrentLimits.MSCSetting.MSCPerSecond,
																				Replicas:profileCurrentLimits.MSCSetting.Replicas,}
				} else {
					//case 2: Increases number of VMS. Find new suitable Vm(s) to cover the number of replicas missing
					//The replicas have the current limits configuration
					deltaNumberReplicas := profileCurrentLimits.MSCSetting.Replicas - currentNumberReplicas
					vmSetDeltaLoad := p.FindSuitableVMs(deltaNumberReplicas,profileCurrentLimits.Limits)
					//Merge the current configuration with configuration for the new replicas
					vmSetDeltaLoad.Merge(p.currentState.VMs)
					rConfigDeltaLoad := types.ContainersConfig {
						Limits:     profileCurrentLimits.Limits,
						MSCSetting: profileCurrentLimits.MSCSetting,
						VMSet:      vmSetDeltaLoad,
					}

					//Find VM a new VM set that possibly changes the current one
					vmSetTLoad := p.FindSuitableVMs(profileCurrentLimits.MSCSetting.Replicas, profileCurrentLimits.Limits)
					resourceConfigTLoad := types.ContainersConfig {
						Limits:     profileCurrentLimits.Limits,
						MSCSetting: profileCurrentLimits.MSCSetting,
						VMSet:      vmSetTLoad,
					}

					//Test if reconfigure the complete VM set for the totalLoad is better
					//Validate if a complete migration is better
					if newConfig,ok := p.shouldRepackVMSet(rConfigDeltaLoad, resourceConfigTLoad,i,processedForecast.CriticalIntervals); ok {
						resourcesConfiguration = newConfig
					} else {
						resourcesConfiguration = rConfigDeltaLoad
					}
				}
			} else if deltaLoad < 0 {
					//case 2: Need to decrease resources
					deltaLoad *= -1
					profileCurrentLimits := selectProfileByLimits(totalLoad, currentContainerLimits, false)
					deltaNumberReplicas := currentNumberReplicas - profileCurrentLimits.MSCSetting.Replicas

					//Build new VM set releasing resources used by extra container replicas
					vmSetDeltaLoad := p.releaseResources(deltaNumberReplicas,p.currentState.VMs)
					rConfigDeltaLoad := types.ContainersConfig {
						Limits:     profileCurrentLimits.Limits,
						MSCSetting: profileCurrentLimits.MSCSetting,
						VMSet:      vmSetDeltaLoad,
					}

					//Find VM a new VM set that possibly changes the current one
					vmSetTLoad := p.FindSuitableVMs(profileCurrentLimits.MSCSetting.Replicas, profileCurrentLimits.Limits)
					resourceConfigTLoad := types.ContainersConfig {
						Limits:     profileCurrentLimits.Limits,
						MSCSetting: profileCurrentLimits.MSCSetting,
						VMSet:      vmSetTLoad,
					}

					//Validate if a complete migration is better
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
		setScalingSteps(&configurations,p.currentState, state,timeStart,timeEnd, totalServicesBootingTime,stateLoadCapacity)
		//Update current state
		p.currentState = state
	}

		//Add new policy
		parameters := make(map[string]string)
		parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
		parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(true)
		parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)
		parameters[types.ISRESIZEPODS] = strconv.FormatBool(false)
		numConfigurations := len(configurations)
		newPolicy.ScalingActions = configurations
		newPolicy.Algorithm = p.algorithm
		newPolicy.ID = bson.NewObjectId()
		newPolicy.Status = types.DISCARTED	//State by default
		newPolicy.Parameters = parameters
		newPolicy.Metrics.NumberScalingActions = numConfigurations
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
	@resourceLimits = Resources (CPU, Memory) constraints to configure the containers.
 out:
	@VMScale with the suggested number of VMs
*/
func (p DeltaRepackedPolicy) FindSuitableVMs(numberReplicas int, resourceLimits types.Limit) types.VMScale {
	vmSet, _ := buildHomogeneousVMSet(numberReplicas,resourceLimits, p.mapVMProfiles)
	hetVMSet,_ := buildHeterogeneousVMSet(numberReplicas, resourceLimits, p.mapVMProfiles)
	costi := hetVMSet.Cost(p.mapVMProfiles)
	costj := vmSet.Cost(p.mapVMProfiles)
	if costi < costj {
		vmSet = hetVMSet
	}
	return vmSet
}

//Use a greedy approach to build a VM cluster of two types  to support the
// deployment of of n number of service replicas
func buildHeterogeneousSet(nReplicas int, typeVM1 string, capacityVM1 int, typeVM2 string, capacityVM2 int) types.VMScale {
	totalReplicas := 0
	c:=capacityVM1
	cn:=capacityVM2
	for totalReplicas <= nReplicas {
		totalReplicas = c + cn
		if totalReplicas >= nReplicas {break}
		c += capacityVM1
		totalReplicas = c + cn
		cn += capacityVM2
	}
	vmScale :=  make(map[string]int)
	vmScale[typeVM1] = c / capacityVM1
	vmScale[typeVM2] = (totalReplicas - c) / capacityVM2
	return  vmScale
}

//Remove the virtual machines that are supporting the deployment of nReplicas from the current configuration.
//If is not possible to remove vms without risk of high underprovisioning, no vm is released
func (p DeltaRepackedPolicy) releaseResources(nReplicas int, currentVMSet types.VMScale) types.VMScale {
	var newVMSet types.VMScale
	newVMSet = copyMap(currentVMSet)

	type keyValue struct {
		Key   string
		Value int
	}
	//Creates a list sorted by the number of machines per type
	var ss []keyValue
	for k, v := range newVMSet {
		ss = append(ss, keyValue{k, v})
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].Value > ss[j].Value })

	for _,kv := range ss {
		cap := p.mapVMProfiles[kv.Key].ReplicasCapacity
		if  newVMSet.TotalVMs() > 1 {
			if nReplicas == cap && kv.Value > 0{
				//Remove 1 VM of this type
				newVMSet[kv.Key]= newVMSet[kv.Key] - 1
				break
			} else if nReplicas > cap && kv.Value * cap > nReplicas {
				rmvVM := int(math.Floor(float64(nReplicas/cap)))
				newVMSet[kv.Key]= newVMSet[kv.Key] - rmvVM
				break
			} else if nReplicas > cap && kv.Value > 0 {
				newVMSet[kv.Key]= newVMSet[kv.Key] - 1
				nReplicas -= cap
			}
		}
	}
	return  newVMSet
}

//Calculate the cost of a reconfiguration
func(p DeltaRepackedPolicy) calculateReconfigurationCost(newSet types.VMScale) float64 {
	//Compute reconfiguration cost
	_, deletedVMS := DeltaVMSet(p.currentState.VMs, newSet)
	reconfigTime := computeVMTerminationTime(deletedVMS, p.sysConfiguration)

	return deletedVMS.Cost(p.mapVMProfiles) * float64(reconfigTime)
}

/*
	in:
		@currentLimits
		@profileCurrentLimits
		@newLimits
		@profileNewLimits
		@vmType
		@containerResize
	out:
		@ContainersConfig
		@error

*/
func (p DeltaRepackedPolicy) selectContainersConfig(currentLimits types.Limit, profileCurrentLimits types.MSCSimpleSetting,
	newLimits types.Limit, profileNewLimits types.MSCSimpleSetting, containerResize bool) (MSCProfile, error) {

	currentNumberReplicas := float64(profileCurrentLimits.Replicas)
	utilizationCurrent := (currentNumberReplicas * currentLimits.CPUCores)+(currentNumberReplicas * currentLimits.MemoryGB)

	newNumberReplicas := float64(profileNewLimits.Replicas)
	utilizationNew := (newNumberReplicas * newLimits.CPUCores)+(newNumberReplicas * newLimits.MemoryGB)

	if utilizationNew < utilizationCurrent && containerResize {
		return MSCProfile{ResourceLimits:newLimits,
							NumberReplicas:int(newNumberReplicas),
							MSC:profileNewLimits.MSCPerSecond,}, nil
	} else {
		return MSCProfile{ResourceLimits:currentLimits,
			NumberReplicas:int(currentNumberReplicas),
			MSC:profileCurrentLimits.MSCPerSecond,}, nil
	}
}

//Evaluate if the current configuration of VMS should be changed to a new configuration
//searching a optimal vm set for the total Load
func(p DeltaRepackedPolicy) shouldRepackVMSet(currentOption types.ContainersConfig, candidateOption types.ContainersConfig, indexTimeInterval int, timeIntervals[]types.CriticalInterval) (types.ContainersConfig, bool) {
	currentCost := currentOption.VMSet.Cost(p.mapVMProfiles)
	candidateCost := candidateOption.VMSet.Cost(p.mapVMProfiles)

	if candidateCost <= currentCost {
		//By default the tranisition policy would be to shut down VMs after launch new ones
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

/*
	Evaluate if only scaling out containers is enough to handle the workload
	either using the current limit constrains or finding other (cpu, mem) configuration that meet the requirement
	and still fit into the current VM set.
*/
func (p DeltaRepackedPolicy) onlyScaleOutContainers(currentVMSet types.VMScale, totalLoad float64, currentContainerLimits types.Limit) (types.ContainersConfig, bool){
	containersResourceConfig :=  types.ContainersConfig{}
	onlyScaleContainers := false

	//Search the number of replicas with current resource limits for the total load
	profileCurrentLimits := selectProfileByLimits(totalLoad, currentContainerLimits, false)
	computeVMsCapacity(currentContainerLimits, &p.mapVMProfiles)
	currentReplicasCapacity := currentVMSet.ReplicasCapacity(p.mapVMProfiles)

	containersResourceConfig.VMSet = p.currentState.VMs
	containersResourceConfig.Limits = profileCurrentLimits.Limits
	containersResourceConfig.MSCSetting = profileCurrentLimits.MSCSetting

	if currentReplicasCapacity > profileCurrentLimits.MSCSetting.Replicas {
		//case 1: Increases number of replicas but VMS remain the same
		onlyScaleContainers = true
	} else {
		//case 2: Search for an option changing containers limits (container hybrid scaling) but VMS remain the same
		configurationOptionFound, optionFound := findConfigOptionByContainerResize(currentVMSet,totalLoad, p.mapVMProfiles)
		if optionFound {
			containersResourceConfig.VMSet = p.currentState.VMs
			containersResourceConfig.Limits = configurationOptionFound.Limits
			containersResourceConfig.MSCSetting = configurationOptionFound.MSCSetting
			onlyScaleContainers = true
		}
	}

	return containersResourceConfig, onlyScaleContainers
}