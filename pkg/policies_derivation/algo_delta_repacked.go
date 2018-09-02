package policies_derivation

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
	policies := []types.Policy{}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}
	configurations := []types.ScalingConfiguration{}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	containerResizeEnabled := p.sysConfiguration.PolicySettings.PodsResizeAllowed


	for i, it := range processedForecast.CriticalIntervals {
		//Load in terms of number of requests
		totalLoad := it.Requests
		resourcesConfiguration := types.ContainersConfig{}
		serviceToScale := p.currentState.Services[p.sysConfiguration.ServiceName]
		currentContainerLimits := types.Limit{ MemoryGB:serviceToScale.Memory, NumberCores:serviceToScale.CPU }
		currentNumberReplicas := serviceToScale.Scale

		//Candidate option to handle total load
		profileCurrentLimits := selectProfileWithLimits(totalLoad, currentContainerLimits, false)
		vmSetTLoadCurrentLimits := p.FindSuitableVMs(profileCurrentLimits.PerformanceProfile.NumberReplicas, profileCurrentLimits.Limits)
		rConfigTLoadCurrentLimits := types.ContainersConfig {
			Limits:profileCurrentLimits.Limits,
			PerformanceProfile:profileCurrentLimits.PerformanceProfile,
			VMSet:vmSetTLoadCurrentLimits,
		}

		//Compute deltaLoad
		currentLoadCapacity := configurationLoadCapacity(currentNumberReplicas,currentContainerLimits)
		deltaLoad := totalLoad - currentLoadCapacity
		if deltaLoad == 0 {
			resourcesConfiguration.VMSet = p.currentState.VMs
			resourcesConfiguration.Limits = currentContainerLimits
			resourcesConfiguration.PerformanceProfile = types.TRNConfiguration{TRN:currentLoadCapacity, NumberReplicas:currentNumberReplicas,}

		} else 	if deltaLoad > 0 {
			//Need to increase resources
			computeVMsCapacity(profileCurrentLimits.Limits, &p.mapVMProfiles)
			replicasCapacity := p.currentState.VMs.ReplicasCapacity(p.mapVMProfiles)

			//Validate if the current configuration is able to handle the new replicas
			//Using the current Resource Limits configuration for the containers
			if replicasCapacity > profileCurrentLimits.PerformanceProfile.NumberReplicas {
				//case 1: Increases number of replicas but VMS remain the same
				resourcesConfiguration.VMSet = p.currentState.VMs
				resourcesConfiguration.Limits = profileCurrentLimits.Limits
				resourcesConfiguration.PerformanceProfile = profileCurrentLimits.PerformanceProfile

			} else if underProvisionAllowed && containerResizeEnabled{
				//case 2: search a new service profile with underprovisioning that possible fit into the
				//current VM set
				profileNewLimits := selectProfile(totalLoad, underProvisionAllowed)
				computeVMsCapacity(profileNewLimits.Limits, &p.mapVMProfiles)
				replicasCapacity := p.currentState.VMs.ReplicasCapacity(p.mapVMProfiles)
				//Validate if the current configuration is able to handle the new replicas
				//Using a new Resource Limits configuration for the containers
				if replicasCapacity > profileNewLimits.PerformanceProfile.NumberReplicas {
					resourcesConfiguration.VMSet = p.currentState.VMs
					resourcesConfiguration.Limits = profileNewLimits.Limits
					resourcesConfiguration.PerformanceProfile = profileNewLimits.PerformanceProfile
				}
			} else {
				//case 3: Increases number of VMS. Find new suitable Vm(s) to cover the number of replicas missing.
				deltaNumberReplicas := profileCurrentLimits.PerformanceProfile.NumberReplicas - currentNumberReplicas

				//Find VM set for totalLoad and validate if a complete migration is better
				if deltaNumberReplicas > 0 {
					vmSetDeltaLoad := p.FindSuitableVMs(deltaNumberReplicas,profileCurrentLimits.Limits)
					vmSetDeltaLoad.Merge(p.currentState.VMs)
					rConfigDeltaLoad := types.ContainersConfig {
						Limits:profileCurrentLimits.Limits,
						PerformanceProfile:profileCurrentLimits.PerformanceProfile,
						VMSet:vmSetDeltaLoad,
					}

					if newConfig,ok := p.shouldRepackVMSet(rConfigDeltaLoad, rConfigTLoadCurrentLimits,i,processedForecast.CriticalIntervals); ok {
						resourcesConfiguration = newConfig
					} else {
						resourcesConfiguration = rConfigDeltaLoad
					}
				}else {
					//TODO: Fix
					resourcesConfiguration = rConfigTLoadCurrentLimits
				}
			}

		} else if deltaLoad < 0 {
			//Need to decrease resources
			deltaLoad *= -1
			deltaReplicas := currentNumberReplicas - profileCurrentLimits.PerformanceProfile.NumberReplicas

				 //Build new VM set releasing resources used by extra container replicas
				 vmSetDeltaLoad := p.releaseResources(deltaReplicas,p.currentState.VMs)
				 rConfigDLoad := types.ContainersConfig {
					 Limits:profileCurrentLimits.Limits,
					 PerformanceProfile:profileCurrentLimits.PerformanceProfile,
					 VMSet:vmSetDeltaLoad,
				 }
				 // Test if reconfigure the complete VM set for the totalLoad is better
				 //Find VM set for totalLoad and validate if a complete migration is better
				 if newConfig,ok := p.shouldRepackVMSet(rConfigDLoad, rConfigTLoadCurrentLimits,i,processedForecast.CriticalIntervals); ok {
					 resourcesConfiguration = newConfig
				 }else {
					 resourcesConfiguration = rConfigDLoad
				 }
		}

		services := make(map[string]types.ServiceInfo)
		services[p.sysConfiguration.ServiceName] = types.ServiceInfo {
			Scale:  resourcesConfiguration.PerformanceProfile.NumberReplicas,
			CPU:    resourcesConfiguration.Limits.NumberCores,
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
		totalServicesBootingTime := resourcesConfiguration.PerformanceProfile.BootTimeSec
		stateLoadCapacity := resourcesConfiguration.PerformanceProfile.TRN
		setConfiguration(&configurations,state,timeStart,timeEnd,p.sysConfiguration.ServiceName, totalServicesBootingTime, p.sysConfiguration,stateLoadCapacity)
		//Update current state
		p.currentState = state
	}

		//Add new policy
		parameters := make(map[string]string)
		parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
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
func (p DeltaRepackedPolicy) FindSuitableVMs(numberReplicas int, resourceLimits types.Limit) types.VMScale {
	//currentVMType, isHomogeneous := p.isCurrentlyHomogeneous()
	heterogeneousAllowed := p.sysConfiguration.PolicySettings.HetereogeneousAllowed
	vmSet, _ := buildHomogeneousVMSet(numberReplicas,resourceLimits, p.mapVMProfiles)

	if heterogeneousAllowed {
		hetVMSet,_ := buildHeterogeneousVMSet(numberReplicas, resourceLimits, p.mapVMProfiles)
		costi := hetVMSet.Cost(p.mapVMProfiles)
		costj := vmSet.Cost(p.mapVMProfiles)
		if costi < costj {
			vmSet = hetVMSet
		}
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
	return types.ContainersConfig{}, false
}
