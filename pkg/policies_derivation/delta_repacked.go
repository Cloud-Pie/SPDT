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

type DeltaRepackedPolicy struct {
	algorithm        string              			 //Algorithm's name
	timeWindow       TimeWindowDerivation 			//Algorithm used to process the forecasted time serie
	currentState     types.State          			//Current State
	sortedVMProfiles []types.VmProfile    			//List of VM profiles sorted by price
	mapVMProfiles    map[string]types.VmProfile		//Map with VM profiles with VM.Type as key
	sysConfiguration	config.SystemConfiguration
}

//Create scaling policies
func (p DeltaRepackedPolicy) CreatePolicies(processedForecast types.ProcessedForecast, serviceProfile types.ServiceProfile) [] types.Policy {

	policies := []types.Policy{}
		newPolicy := types.Policy{}
		newPolicy.Metrics = types.PolicyMetrics {
			StartTimeDerivation:time.Now(),
		}
		configurations := []types.Configuration{}
		underprovisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
		//Select the performance profile that fits better
		performanceProfile := selectProfile(serviceProfile.PerformanceProfiles)
		//calculate the capacity of services replicas to each VM type
		computeCapacity(&p.sortedVMProfiles, performanceProfile, &p.mapVMProfiles)

		for i, it := range processedForecast.CriticalIntervals {
			var newNumServiceReplicas 	int
			var vmSet 					types.VMScale
			var currentNumberReplicas int

			//Load in terms of number of requests
			totalLoad := it.Requests
			services := make(map[string]types.ServiceInfo)
			for _,v := range p.currentState.Services {	currentNumberReplicas = v.Scale }

			//Compute deltaLoad
			currentLoadCapacity := float64(currentNumberReplicas/performanceProfile.NumReplicas) * performanceProfile.TRN
			deltaLoad := totalLoad - currentLoadCapacity
			//By default keep current configuration
			newNumServiceReplicas = currentNumberReplicas
			vmSet = p.currentState.VMs

			if deltaLoad > 0 {
				//Need to increase resources
				//Compute number of replicas needed only to handle delta load
				numReplicasDeltaLoad := int(math.Ceil(deltaLoad / performanceProfile.TRN)) * performanceProfile.NumReplicas
				//increases number of replicas
				newNumServiceReplicas = currentNumberReplicas + numReplicasDeltaLoad

				//Validate if the current configuration is able to handle the new replicas
				currentOpt := VMSet{VMSet:p.currentState.VMs}
				currentOpt.setValues(p.mapVMProfiles)
				if currentOpt.TotalReplicasCapacity >= newNumServiceReplicas {
					vmSet = p.currentState.VMs
				} else {
					//Increase number of VMs and find Optimal VM Set to deploy services required to handle deltaLoad
					vmSet = p.findOptimalVMSet(numReplicasDeltaLoad)
					//Merge the current configuration with configuration for the new replicas
					vmSet.Merge(p.currentState.VMs)
				}
				// Test if reconfigure the complete VM set for the totalLoad is better
				opt := VMSet{VMSet: vmSet}
				opt.setValues(p.mapVMProfiles)
				//Find VM set for totalLoad and validate if a complete migration is better
				if newVMSet,ok := p.repackVMSet(totalLoad,opt,i,processedForecast.CriticalIntervals,performanceProfile); ok {
					vmSet = newVMSet
				}

			} else if deltaLoad < 0 {
				//Need to decrease resources
				deltaLoad *= -1
				//Calculate number of replicas that should be removed to decrease -deltaLoad
				numReplicasDeltaLoad := int(math.Floor(deltaLoad / performanceProfile.TRN)) * performanceProfile.NumReplicas
				newNumServiceReplicas = currentNumberReplicas - numReplicasDeltaLoad
				//Build new set of VMs and release some if necessary
				if numReplicasDeltaLoad > 0 {
					vmSet = p.releaseResources(numReplicasDeltaLoad, p.currentState.VMs)
				}

				if underprovisionAllowed {
					underProvReplicas,newVMset:= p.considerIfUnderprovision(vmSet, performanceProfile, deltaLoad)
					if newVMset != nil {
						newNumServiceReplicas = underProvReplicas
						vmSet = newVMset
					}
				}

				// Test if reconfigure the complete VM set for the totalLoad is better
				opt := VMSet{VMSet: vmSet}
				opt.setValues(p.mapVMProfiles)
				if rep,ok := p.repackVMSet(totalLoad,opt, i, processedForecast.CriticalIntervals,performanceProfile); ok {
						vmSet = rep
				}
			}

			services[serviceProfile.Name] = types.ServiceInfo {
				Scale:  newNumServiceReplicas,
				CPU:    performanceProfile.Limit.NumCores,
				Memory: performanceProfile.Limit.Memory,
			}

			//Create a new state
			state := types.State{}
			state.Services = services
			state.VMs = vmSet
			stateLoadCapacity := float64(newNumServiceReplicas/performanceProfile.NumReplicas) * performanceProfile.TRN
			timeStart := it.TimeStart
			timeEnd := it.TimeEnd
			totalServicesBootingTime := performanceProfile.BootTimeSec
			setConfiguration(&configurations,state,timeStart,timeEnd,serviceProfile.Name, totalServicesBootingTime, p.sysConfiguration,stateLoadCapacity)

			//Update current state
			p.currentState = state
		}

		//Add new policy
		parameters := make(map[string]string)
		parameters[types.METHOD] = "horizontal"
		parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(p.sysConfiguration.PolicySettings.HetereogeneousAllowed)
		parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underprovisionAllowed)
		if underprovisionAllowed {
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

//Find an optimal set of virtual machines to support the deployment of of n number of service replicas
func (p DeltaRepackedPolicy) findOptimalVMSet(nReplicas int) types.VMScale {
	nVMProfiles := len(p.sortedVMProfiles)
	solutions := []VMSet{}
	currentVMType, isHomogeneous := p.isCurrentlyHomogeneous()
	hetereougeneouseAllowed := p.sysConfiguration.PolicySettings.HetereogeneousAllowed
		for i,v := range p.sortedVMProfiles {
			vmScale :=  make(map[string]int)
			capacity := v.ReplicasCapacity
			if capacity > nReplicas {
				vmScale[v.Type] = 1
				set := VMSet{VMSet:vmScale}
				set.setValues(p.mapVMProfiles)
				solutions = append(solutions, set)
				if !hetereougeneouseAllowed && isHomogeneous && v.Type == currentVMType {
					return set.VMSet
				}
			} else if capacity > 0 {
				//Homogeneous candidate set of type v
				vmScale[v.Type] = int(nReplicas / capacity)
				set := VMSet{VMSet:vmScale}
				set.setValues(p.mapVMProfiles)
				solutions = append(solutions, set)

				if hetereougeneouseAllowed {
					//heterogeneous candidate set of two types
					//The current type and the next type in size order
					if i < nVMProfiles-1 {
						vmScale := buildHeterogeneousSet(nReplicas, v.Type, capacity,  p.sortedVMProfiles[i+1].Type,  p.sortedVMProfiles[i+1].ReplicasCapacity)
						set := VMSet{VMSet:vmScale}
						set.setValues(p.mapVMProfiles)
						solutions = append(solutions, set)
					}
				}else if isHomogeneous && v.Type == currentVMType {
					return set.VMSet
				}
			}
		}

	sort.Slice(solutions, func(i, j int) bool {
		if solutions[i].Cost < solutions[j].Cost {
			return true
		} else if solutions[i].Cost ==  solutions[j].Cost {
			return solutions[i].TotalNVMs >= solutions[j].TotalNVMs
		}
		return false
	})

	return solutions[0].VMSet
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

//Evaluate if the current configuration of VMS should be changed to a new configuration
//searching a optimal vm set for the total Load
func(p DeltaRepackedPolicy) repackVMSet(totalLoad float64, currentOptimalSet VMSet, indexTimeInterval int, timeIntervals[]types.CriticalInterval, performanceProfile types.PerformanceProfile) (types.VMScale, bool) {
	numReplicasTotalLoad := int(math.Ceil(totalLoad / performanceProfile.TRN)) * performanceProfile.NumReplicas
	newSet := p.findOptimalVMSet(numReplicasTotalLoad)
	newOptimalSet := VMSet{VMSet:newSet}
	newOptimalSet.setValues(p.mapVMProfiles)

	if (newOptimalSet.Cost <= currentOptimalSet.Cost) {
		//By default the tranisition policy would be to shut down VMs after launch new ones
		//Calculate reconfiguration time
		timeStart := timeIntervals[indexTimeInterval].TimeStart
		var timeEnd time.Time
		idx := indexTimeInterval
		lenInterval := len(timeIntervals)
		//Compute duration for new set
		for idx < lenInterval {
			currentLoadCapacity := float64(newOptimalSet.TotalReplicasCapacity/performanceProfile.NumReplicas) * performanceProfile.TRN
			if timeIntervals[idx].Requests > currentLoadCapacity {
				timeEnd = timeIntervals[idx].TimeStart
				break
			}
			idx+=1
		}
		durationNewVMSet :=  timeEnd.Sub(timeStart).Seconds()
		reconfigCostNew := p.calculateReconfigurationCost(newSet)

		//Compute duration for current set
		jdx := indexTimeInterval
		for jdx < lenInterval {
			currentLoadCapacity := float64(currentOptimalSet.TotalReplicasCapacity/performanceProfile.NumReplicas) * performanceProfile.TRN
			if timeIntervals[jdx].Requests > currentLoadCapacity{
				timeEnd = timeIntervals[jdx].TimeStart
				break
			}
			jdx+=1
		}
		durationCurrentVMSet :=  timeEnd.Sub(timeStart).Seconds()
		reconfigCostCurrent := p.calculateReconfigurationCost(currentOptimalSet.VMSet)

		if newOptimalSet.Cost*durationNewVMSet + reconfigCostNew < currentOptimalSet.Cost * durationCurrentVMSet + reconfigCostCurrent {
			return newSet, true
		}
	}
	return nil, false
}



//Calculate the cost of a reconfiguration
func(p DeltaRepackedPolicy) calculateReconfigurationCost(newSet types.VMScale) float64 {
	//Compute reconfiguration cost
	_, deletedVMS := deltaVMSet(p.currentState.VMs, newSet)
	reconfigTime := computeVMTerminationTime(deletedVMS, p.sysConfiguration)

	deletedSet := VMSet{VMSet : deletedVMS}
	deletedSet.setValues(p.mapVMProfiles)
	return deletedSet.Cost * float64(reconfigTime)
}

//Compares if according to the minimum percentage of underprovisioning is possible to find a cheaper VM set
//by decreasing the number of replicas and comparing the capacity of a VM set for overprovisioning against the new one
//found for underprovision
func (p DeltaRepackedPolicy) considerIfUnderprovision(overVmSet types.VMScale, performanceProfile types.PerformanceProfile, requests float64)(int, types.VMScale){
	var newNumServiceReplicas int

	//Compute number of replicas that leads to  underprovision
	underNumServiceReplicas := int(math.Ceil(requests / performanceProfile.TRN)) * performanceProfile.NumReplicas
	underProvisionTRN := float64(underNumServiceReplicas / performanceProfile.NumReplicas)*performanceProfile.TRN
	percentageUnderProvisioned := underProvisionTRN * requests / 100.0
	//Compare if underprovision in terms of number of request is acceptable
	if percentageUnderProvisioned <= p.sysConfiguration.PolicySettings.MaxUnderprovision {
		vmSet := p.releaseResources(underNumServiceReplicas, p.currentState.VMs)
		//Compare vm sets for underprovisioning and overprovisioning of service replicas
		overVMSet := VMSet{VMSet:overVmSet}
		overVMSet.setValues(p.mapVMProfiles)
		underVMSet := VMSet{VMSet:vmSet}
		underVMSet.setValues(p.mapVMProfiles)
		//Compare if the change allowing underprovisioning really affect the selected vm set
		if underVMSet.TotalReplicasCapacity < overVMSet.TotalReplicasCapacity {
			newNumServiceReplicas = underNumServiceReplicas
			return newNumServiceReplicas,vmSet
		}
	}
	return newNumServiceReplicas,nil
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