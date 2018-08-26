package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"gopkg.in/mgo.v2/bson"
	"math"
	"sort"
	"github.com/Cloud-Pie/SPDT/config"
	"strconv"
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
	policies := []types.Policy{}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}
	configurations := []types.Configuration{}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	containerResizeEnabled := true
	currentContainerLimits := p.currentContainerLimits()

	for i, it := range processedForecast.CriticalIntervals {
		//Load in terms of number of requests
		totalLoad := it.Requests
		resourcesConfiguration := ContainersConfig{}

		profileCurrentLimits := selectProfileWithLimits(totalLoad, currentContainerLimits, false)
		vmSetTLoadCurrentLimits := p.findOptimalVMSet(profileCurrentLimits.TRNConfiguration[0].NumberReplicas, profileCurrentLimits.Limit)
		rConfigTLoadCurrentLimits := ContainersConfig {
			ResourceLimits:profileCurrentLimits.Limit,
			PerformanceProfile:profileCurrentLimits.TRNConfiguration[0],
			VMSet:vmSetTLoadCurrentLimits,
		}

		profileNewLimits := selectProfile(totalLoad, underProvisionAllowed)
		vmSetTotalLoadNewLimits := p.findOptimalVMSet(profileNewLimits.TRNConfiguration[0].NumberReplicas, profileNewLimits.Limit)
		rConfigTLoadNewLimits := ContainersConfig {
			ResourceLimits:profileNewLimits.Limit,
			PerformanceProfile:profileNewLimits.TRNConfiguration[0],
			VMSet:vmSetTotalLoadNewLimits,
		}
		//Compute deltaLoad
		currentLoadCapacity := p.currentLoadCapacity()
		deltaLoad := totalLoad - currentLoadCapacity

		if deltaLoad > 0 {
			//Need to increase resources
			computeVMsCapacity(profileCurrentLimits.Limit, &p.mapVMProfiles)
			replicasCapacity := p.currentState.VMs.ReplicasCapacity(p.mapVMProfiles)

			//Validate if the current configuration is able to handle the new replicas
			//Using the current Resource Limits configuration for the containers
			if replicasCapacity > profileCurrentLimits.TRNConfiguration[0].NumberReplicas {
				resourcesConfiguration.VMSet = p.currentState.VMs
				resourcesConfiguration.ResourceLimits = profileCurrentLimits.Limit
				resourcesConfiguration.PerformanceProfile = profileCurrentLimits.TRNConfiguration[0]
			} else {
				computeVMsCapacity(profileNewLimits.Limit, &p.mapVMProfiles)
				replicasCapacity := p.currentState.VMs.ReplicasCapacity(p.mapVMProfiles)
				//Validate if the current configuration is able to handle the new replicas
				//Using a new Resource Limits configuration for the containers
				if replicasCapacity > profileNewLimits.TRNConfiguration[0].NumberReplicas {
					resourcesConfiguration.VMSet = p.currentState.VMs
					resourcesConfiguration.ResourceLimits = profileNewLimits.Limit
					resourcesConfiguration.PerformanceProfile = profileNewLimits.TRNConfiguration[0]
				}
			}

			//Find vmSet to handle new replicas that supply deltaLoad
			profileCurrentLimits = selectProfileWithLimits(deltaLoad, currentContainerLimits, false)
			profileNewLimits = selectProfile(deltaLoad, underProvisionAllowed)
			trnProfile,_ := p.selectContainersConfig(profileCurrentLimits.Limit, profileCurrentLimits.TRNConfiguration[0],
				 									profileNewLimits.Limit, profileNewLimits.TRNConfiguration[0], containerResizeEnabled)
			vmSetDeltaLoad := p.findOptimalVMSet(trnProfile.NumberReplicas,trnProfile.ResourceLimits)
			vmSetDeltaLoad.Merge(p.currentState.VMs)
			rConfigDLoad := ContainersConfig {
				ResourceLimits:trnProfile.ResourceLimits,
				PerformanceProfile:types.TRNConfiguration{NumberReplicas:trnProfile.NumberReplicas, TRN:trnProfile.TRN},
				VMSet:vmSetDeltaLoad,
			}
			resourcesConfiguration = rConfigDLoad

			//Find VM set for totalLoad and validate if a complete migration is better
			if newConfig,ok := p.shouldRepackVMSet(rConfigDLoad, rConfigTLoadCurrentLimits,i,processedForecast.CriticalIntervals); ok {
				resourcesConfiguration = newConfig
				if newConfig,ok := p.shouldRepackVMSet(newConfig, rConfigTLoadNewLimits,i,processedForecast.CriticalIntervals); ok {
					resourcesConfiguration = newConfig
				}
			}

		} else if deltaLoad < 0 {
			//Need to decrease resources
			deltaLoad *= -1
			trnProfile,_ := p.selectContainersConfig(profileCurrentLimits.Limit, profileCurrentLimits.TRNConfiguration[0],
													profileNewLimits.Limit, profileNewLimits.TRNConfiguration[0], containerResizeEnabled)
			vmSetDeltaLoad := p.releaseResources(trnProfile.NumberReplicas,p.currentState.VMs)

			rConfigDLoad := ContainersConfig {
				ResourceLimits:trnProfile.ResourceLimits,
				PerformanceProfile:types.TRNConfiguration{NumberReplicas:trnProfile.NumberReplicas, TRN:trnProfile.TRN},
				VMSet:vmSetDeltaLoad,
			}
			// Test if reconfigure the complete VM set for the totalLoad is better
			//Find VM set for totalLoad and validate if a complete migration is better
			if newConfig,ok := p.shouldRepackVMSet(rConfigDLoad, rConfigTLoadCurrentLimits,i,processedForecast.CriticalIntervals); ok {
				resourcesConfiguration = newConfig
				if newConfig,ok := p.shouldRepackVMSet(newConfig, rConfigTLoadNewLimits,i,processedForecast.CriticalIntervals); ok {
					resourcesConfiguration = newConfig
				}
			}
		}

		services := make(map[string]types.ServiceInfo)
		services[p.sysConfiguration.ServiceName] = types.ServiceInfo {
			Scale:  resourcesConfiguration.PerformanceProfile.NumberReplicas,
			CPU:    resourcesConfiguration.ResourceLimits.NumberCores,
			Memory: resourcesConfiguration.ResourceLimits.MemoryGB,
		}

		//Create a new state
		state := types.State{}
		state.Services = services
		vmSet := resourcesConfiguration.VMSet
		cleanKeys(vmSet)
		state.VMs = vmSet
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		totalServicesBootingTime := resourcesConfiguration.PerformanceProfile.BootTimeSec
		stateLoadCapacity := resourcesConfiguration.PerformanceProfile.TRN
		setConfiguration(&configurations,state,timeStart,timeEnd,p.sysConfiguration.ServiceName, totalServicesBootingTime, p.sysConfiguration,stateLoadCapacity)
		//Update current state
		p.currentState = state
	}

		//Add new policy
		parameters := make(map[string]string)
		parameters[types.METHOD] = "horizontal"
		parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(p.sysConfiguration.PolicySettings.HetereogeneousAllowed)
		parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)
		if underProvisionAllowed {
			parameters[types.MAXUNDERPROVISION] = strconv.FormatFloat(p.sysConfiguration.PolicySettings.MaxUnderprovision, 'f', -1, 64)
		}
		numConfigurations := len(configurations)
		newPolicy.Configurations = configurations
		newPolicy.Algorithm = p.algorithm
		newPolicy.ID = bson.NewObjectId()
		newPolicy.Status = types.DISCARTED	//State by default
		newPolicy.Parameters = parameters
		newPolicy.Metrics.NumberConfigurations = numConfigurations
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
func (p DeltaRepackedPolicy) findOptimalVMSet(numberReplicas int, resourceLimits types.Limit) types.VMScale {
	//nVMProfiles := len(p.sortedVMProfiles)
	candidateSolutions := []types.VMScale{}
	//currentVMType, isHomogeneous := p.isCurrentlyHomogeneous()
	//heterogeneousAllowed := p.sysConfiguration.PolicySettings.HetereogeneousAllowed

	for _,v := range p.sortedVMProfiles {
		vmScale :=  make(map[string]int)
		replicasCapacity :=  maxReplicasCapacityInVM(v, resourceLimits)
		if replicasCapacity > 0 {
			numVMs := math.Ceil(float64(numberReplicas) / float64(replicasCapacity))
			vmScale[v.Type] = int(numVMs)
			candidateSolutions = append(candidateSolutions, vmScale)
		}
	}


	/*TODO
	if heterogeneousAllowed {
		//heterogeneous candidate set of two types
		//The current type and the next type in size order
		if i < nVMProfiles-1 {
			vmScale := buildHeterogeneousSet(numberReplicas, v.Type, capacity,  p.sortedVMProfiles[i+1].Type,  p.sortedVMProfiles[i+1].ReplicasCapacity)
			set := VMSet{VMSet:vmScale}
			set.setValues(p.mapVMProfiles)
			candidateSolutions = append(candidateSolutions, set)
		}
	}else if isHomogeneous && v.Type == currentVMType {
		return set.VMSet
	}*/

	sort.Slice(candidateSolutions, func(i, j int) bool {
		costi := candidateSolutions[i].Cost(p.mapVMProfiles)
		costj := candidateSolutions[j].Cost(p.mapVMProfiles)
		if costi < costj {
			return true
		} else if costi ==  costj {
			return candidateSolutions[i].TotalVMs() >= candidateSolutions[j].TotalVMs()
		}
		return false
	})

	return candidateSolutions[0]
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
	newVMSet := copyMap(currentVMSet)

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

	return  newVMSet
}

//Calculate the cost of a reconfiguration
func(p DeltaRepackedPolicy) calculateReconfigurationCost(newSet types.VMScale) float64 {
	//Compute reconfiguration cost
	_, deletedVMS := deltaVMSet(p.currentState.VMs, newSet)
	reconfigTime := computeVMTerminationTime(deletedVMS, p.sysConfiguration)

	return deletedVMS.Cost(p.mapVMProfiles) * float64(reconfigTime)
}

//Return the VM type used by the current Homogeneous VM cluster
func (p DeltaRepackedPolicy) isCurrentlyHomogeneous() (string, bool) {
	//Assumption for p approach: There is only 1 vm Type in current state
	var vmType string
	isHomogeneous := true
	for k,_ := range p.currentState.VMs {
		vmType = k
	}
	if len(p.currentState.VMs) > 1 {
		isHomogeneous = false
	}
	return vmType, isHomogeneous
}

/*Return the Limit constraint of the current configuration
	out:
		Limit
*/
func (p DeltaRepackedPolicy) currentContainerLimits() types.Limit {
	var limits types.Limit
	for _,s := range p.currentState.Services {
		limits.MemoryGB = s.Memory
		limits.NumberCores = s.CPU
	}
	return limits
}

/*Return the load capacity of the current configuration
	out:
		Limit
*/
func (p DeltaRepackedPolicy) currentLoadCapacity() float64 {
	//TODO
	return 0.0
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
func (p DeltaRepackedPolicy) selectContainersConfig(currentLimits types.Limit, profileCurrentLimits types.TRNConfiguration,
	newLimits types.Limit, profileNewLimits types.TRNConfiguration, containerResize bool) (TRNProfile, error) {

	currentNumberReplicas := float64(profileCurrentLimits.NumberReplicas)
	utilizationCurrent := (currentNumberReplicas * currentLimits.NumberCores)+(currentNumberReplicas * currentLimits.MemoryGB)

	newNumberReplicas := float64(profileNewLimits.NumberReplicas)
	utilizationNew := (newNumberReplicas * newLimits.NumberCores)+(newNumberReplicas * newLimits.MemoryGB)

	if utilizationNew < utilizationCurrent && containerResize {
		return TRNProfile{ResourceLimits:newLimits,
							NumberReplicas:int(newNumberReplicas),
							TRN:profileNewLimits.TRN,}, nil
	} else {
		return TRNProfile{ResourceLimits:currentLimits,
			NumberReplicas:int(currentNumberReplicas),
			TRN:profileCurrentLimits.TRN,}, nil
	}
}

//Evaluate if the current configuration of VMS should be changed to a new configuration
//searching a optimal vm set for the total Load
func(p DeltaRepackedPolicy) shouldRepackVMSet(currentOption ContainersConfig, candidateOption ContainersConfig, indexTimeInterval int, timeIntervals[]types.CriticalInterval) (ContainersConfig, bool) {
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
		candidateLoadCapacity := candidateOption.PerformanceProfile.TRN
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
		currentLoadCapacity := currentOption.PerformanceProfile.TRN
		for jdx < lenInterval {
			if timeIntervals[jdx].Requests > currentLoadCapacity{
				timeEnd = timeIntervals[jdx].TimeStart
				break
			}
			jdx+=1
		}
		durationCurrentVMSet :=  timeEnd.Sub(timeStart).Seconds()
		reconfigCostCurrent := p.calculateReconfigurationCost(currentOption.VMSet)

		if candidateCost*durationNewVMSet + reconfigCostNew < candidateCost * durationCurrentVMSet + reconfigCostCurrent {
			return candidateOption, true
		}
	}
	return ContainersConfig{}, false
}