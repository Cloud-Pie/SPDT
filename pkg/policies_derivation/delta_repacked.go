package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"strconv"
	"github.com/Cloud-Pie/SPDT/util"
	"gopkg.in/mgo.v2/bson"
	db "github.com/Cloud-Pie/SPDT/storage/policies"
	"math"
	"sort"
)

type DeltaRepackedPolicy struct {
	algorithm        string              			 //Algorithm's name
	timeWindow       TimeWindowDerivation 			//Algorithm used to process the forecasted time serie
	currentState     types.State          			//Current State
	sortedVMProfiles []types.VmProfile    			//List of VM profiles sorted by price
	mapVMProfiles    map[string]types.VmProfile		//Map with VM profiles with VM.Type as key
}

type OptimalVMSet struct {
	VMSet                 types.VMScale
	Cost                  float64
	TotalNVMs             int
	TotalReplicasCapacity int
}

//Create scaling policies
func (p DeltaRepackedPolicy) CreatePolicies(processedForecast types.ProcessedForecast, serviceProfile types.ServiceProfile) [] types.Policy {

	policies := []types.Policy{}
		newPolicy := types.Policy{}
		newPolicy.StartTimeDerivation = time.Now()
		configurations := []types.Configuration{}
		totalOverProvision := float32(0.0)
		totalUnderProvision := float32(0.0)
		nPeaksInConfiguration := float32(0.0)
		avgOver := float32(0.0)
		avgUnder := float32(0.0)

		//Select the performance profile that fits better
		performanceProfile := selectProfile(serviceProfile.PerformanceProfiles)

		//calculate the capacity of services replicas to each VM type
		for i,v := range p.sortedVMProfiles {
			cap := MaxReplicasInVM(v,performanceProfile.Limit)
			p.sortedVMProfiles[i].ReplicasCapacity = cap
			profile := p.mapVMProfiles[v.Type]
			profile.ReplicasCapacity = cap
			p.mapVMProfiles[v.Type] = profile
		}

		for i, it := range processedForecast.CriticalIntervals {
			var nProfileCopies 		int
			var nServiceReplicas	int
			var vms 				types.VMScale
			var currentNServices 	int
			totalLoad := it.Requests
			services := make(map[string]int)

			for _,v := range p.currentState.Services {	currentNServices = v }

			//Compute deltaLoad
			currentCapacity := (currentNServices/performanceProfile.NumReplicas) * performanceProfile.TRN
			deltaLoad := totalLoad - currentCapacity

			//By default keep current configuration
			services = p.currentState.Services
			vms = p.currentState.VMs

			if deltaLoad > 0 {
				//Need to increase resources
				//Number of replicas to supply deltaLoad
				nProfileCopies = int(math.Ceil(float64(deltaLoad) / float64(performanceProfile.TRN)))
				nServiceReplicas = nProfileCopies * performanceProfile.NumReplicas

				//increases number of replicas
				services = make(map[string]int)
				services[serviceProfile.Name] = currentNServices + nServiceReplicas

				//Validate if the current configuration is able to handle the new replicas
				currentOpt := OptimalVMSet{VMSet:p.currentState.VMs}
				currentOpt.setValues(p.mapVMProfiles)
				if (currentOpt.TotalReplicasCapacity >= currentNServices + nServiceReplicas) {
					vms = p.currentState.VMs
				} else {
					//Increase number of VMs
					//Find Optimal Set for deltaLoad
					vms = p.findOptimalVMSet(nServiceReplicas)
					//Merge the current configuration with configuration for the new replicas
					for k,v :=  range  p.currentState.VMs {
						if _,ok := vms[k]; ok {
							vms[k] += v
						}else {
							vms[k] = v
						}
					}
				}

				// Test if reconfigure the complete VM set for the totalLoad is better
				opt := OptimalVMSet{VMSet:vms}
				opt.setValues(p.mapVMProfiles)

				//Find VM set for totalLoad
				nProfileCopies = int(math.Ceil(float64(totalLoad) / float64(performanceProfile.TRN)))
				nServiceReplicas = nProfileCopies * performanceProfile.NumReplicas

				if rep,ok := p.repackVMSet(nServiceReplicas,opt,i,processedForecast.CriticalIntervals,performanceProfile); ok {
					vms = rep
				}

			} else if deltaLoad < 0 {
				//Need to decrease resources
				deltaLoad *= -1
				//Calculate number of replicas that should be removed to decrease -deltaLoad
				nProfileCopies = int(math.Floor(float64(deltaLoad) / float64(performanceProfile.TRN)))

				//Validate condition to avoid underprovisioning
				if (nProfileCopies > 0) {
					nServiceReplicas = nProfileCopies * performanceProfile.NumReplicas
					tmp := currentNServices - nServiceReplicas
					//validate if after removing those replicas still SLA is met
					if tmp*(performanceProfile.TRN/performanceProfile.NumReplicas) >= totalLoad {

						//Decreases number of replicas
						services = make(map[string]int)
						services[serviceProfile.Name] = currentNServices - nServiceReplicas
						//Build new set of VMs and release some if necessary
						vms = p.releaseResources(nServiceReplicas, p.currentState.VMs)

						// Test if reconfigure the complete VM set for the totalLoad is better
						opt := OptimalVMSet{VMSet:vms}
						opt.setValues(p.mapVMProfiles)
						nProfileCopies = int(math.Ceil(float64(totalLoad) / float64(performanceProfile.TRN)))
						nServiceReplicas = nProfileCopies * performanceProfile.NumReplicas
						if rep,ok := p.repackVMSet(nServiceReplicas,opt, i, processedForecast.CriticalIntervals,performanceProfile); ok {
							vms = rep
						}
					}
				}
			}

			//Compute under/over provision
			var ns int
			for _,v := range services {	ns = v }
			cc := ns*(performanceProfile.TRN/performanceProfile.NumReplicas)
			diff := cc - totalLoad

			var over, under float32
			if diff >= 0 {
				over = float32(diff * 100) / float32(totalLoad)
				under = 0
			} else {
				over = 0
				under = -1 * float32(diff*100) / float32(totalLoad)
			}
			nPeaksInConfiguration +=1
			avgOver += over
			avgUnder += under

			//Create a new state
			state := types.State{}
			state.Services = services
			state.VMs = vms

			/*
			timeStart := it.TimeStart
			timeEnd := it.TimeEnd
			totalServicesBootingTime := performanceProfile.BootTimeSec
			p.setConfiguration(&configurations,state,timeStart,timeEnd,serviceProfile.Name, totalServicesBootingTime)
			*/

			//Adjust termination times for resources configuration
			terminationTime := ComputeVMTerminationTime(p.mapVMProfiles, vms)
			finishTime := it.TimeEnd.Add(time.Duration(terminationTime) * time.Second)

			nConfigurations := len(configurations)
			if nConfigurations >= 1 && state.Equal(configurations[nConfigurations-1].State) {
				configurations[nConfigurations-1].TimeEnd = finishTime
				configurations[nConfigurations-1].Metrics.OverProvision = avgOver/ nPeaksInConfiguration
				p.currentState = state
			} else {
				//Adjust booting times for resources configuration
				transitionTime := ComputeVMBootingTime(p.mapVMProfiles, vms)
				totalServicesBootingTime := performanceProfile.BootTimeSec 								//TODO: It should include a booting rate
				startTime := it.TimeStart.Add(-1 * time.Duration(transitionTime) * time.Second)       //Booting time VM
				startTime = startTime.Add(-1 * time.Duration(totalServicesBootingTime) * time.Second) //Start time containers
				state.LaunchTime = startTime
				state.Name = strconv.Itoa(nConfigurations) + "__" + serviceProfile.Name + "__" + startTime.Format(util.TIME_LAYOUT)
				configurations = append(configurations,
					types.Configuration {
						State:     state,
						TimeStart: it.TimeStart,
						TimeEnd:   finishTime,
						Metrics: types.Metrics{
							OverProvision:  over,
							UnderProvision: under,
						},
					})
				nPeaksInConfiguration = 0
				avgOver = 0
				p.currentState = state
			}

			totalOverProvision += over
			totalUnderProvision += under
		}

		totalConfigurations := len(configurations)
		totalPeaks := len(processedForecast.CriticalIntervals)
		//Add new policy
		newPolicy.Configurations = configurations
		newPolicy.FinishTimeDerivation = time.Now()
		newPolicy.Algorithm = p.algorithm
		newPolicy.ID = bson.NewObjectId()
		newPolicy.Metrics = types.Metrics{
			OverProvision:        totalOverProvision / float32(totalPeaks),
			UnderProvision:       totalUnderProvision / float32(totalPeaks),
			NumberConfigurations: totalConfigurations,
		}
		//store policy
		db.Store(newPolicy)
		policies = append(policies, newPolicy)

		return policies
}

// Set missing values for the OptimalVMSet structure
func (set *OptimalVMSet) setValues(mapVMProfiles map[string]types.VmProfile) {
	cost := float64(0.0)
	totalNVMs := 0
	totalCapacity :=0
	for k,v := range set.VMSet {
		cost += mapVMProfiles[k].Pricing.Price * float64(v)
		totalNVMs += v
		totalCapacity += mapVMProfiles[k].ReplicasCapacity * v
	}
	set.Cost = cost
	set.TotalNVMs = totalNVMs
	set.TotalReplicasCapacity = totalCapacity
}

//Find an optimal set of virtual machines to support the deployment of of n number of service replicas
func (p DeltaRepackedPolicy) findOptimalVMSet(nReplicas int) types.VMScale {
	nVMProfiles := len(p.sortedVMProfiles)
	solutions := []OptimalVMSet{}

		for i,v := range p.sortedVMProfiles {
			vmScale :=  make(map[string]int)
			capacity := v.ReplicasCapacity
			if capacity > nReplicas {
				vmScale[v.Type] = 1
				set := OptimalVMSet{VMSet:vmScale}
				set.setValues(p.mapVMProfiles)
				solutions = append(solutions, set)
			} else if capacity > 0 {
				//Homogeneous candidate set of type v
				vmScale[v.Type] = int(nReplicas / capacity)
				set := OptimalVMSet{VMSet:vmScale}
				set.setValues(p.mapVMProfiles)
				solutions = append(solutions, set)

				//heterogeneous candidate set of types v and v+1
				if i < nVMProfiles-1 {
					vmScale := hetCandidateSet(nReplicas, v.Type, capacity,  p.sortedVMProfiles[i+1].Type,  p.sortedVMProfiles[i+1].ReplicasCapacity)
					set := OptimalVMSet{VMSet:vmScale}
					set.setValues(p.mapVMProfiles)
					solutions = append(solutions, set)
				}
			}
		}

	sort.Slice(solutions, func(i, j int) bool {
		if solutions[i].Cost < solutions[j].Cost {
			return true
		} else if solutions[i].Cost ==  solutions[j].Cost {
			return solutions[i].TotalNVMs <= solutions[j].TotalNVMs
		}
		return false
	})

	return solutions[0].VMSet
}

//Use a greedy approach to build a VM cluster of two types  to support the deployment of of n number of service replicas
func hetCandidateSet(nReplicas int, typeVM1 string, capacityVM1 int, typeVM2 string, capacityVM2 int) types.VMScale {
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

//Remove the virtual machines that are supporting the deployment of nReplicas from the current configuration. if possible.
func (p DeltaRepackedPolicy) releaseResources(nReplicas int, currentVMSet types.VMScale) types.VMScale {
	newVMSet := CopyMap(currentVMSet)

	type kv struct {
		Key   string
		Value int
	}

	var ss []kv
	for k, v := range newVMSet {
		ss = append(ss, kv{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

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

//Evaluate if repack the current configuration to a new one is worth it
func(p DeltaRepackedPolicy) repackVMSet(requiredReplicasCapacity int, currentOptimalSet OptimalVMSet, indexTimeInterval int, timeIntervals[]types.CriticalInterval, performanceProfile types.PerformanceProfile) (types.VMScale, bool) {

	newSet := p.findOptimalVMSet(requiredReplicasCapacity)
	newOptimalSet := OptimalVMSet {VMSet:newSet}
	newOptimalSet.setValues(p.mapVMProfiles)

	factor := performanceProfile.TRN / performanceProfile.NumReplicas

	if (newOptimalSet.Cost < currentOptimalSet.Cost) {
		//By default the tranisition policy would be to shut down VMs after launch new ones
		//Calculate reconfiguration time

		timeStart := timeIntervals[indexTimeInterval].TimeStart
		var timeEnd time.Time
		idx := indexTimeInterval
		lenInterval := len(timeIntervals)

		//Compute duration for new set
		for idx < lenInterval {
			if timeIntervals[idx].Requests > newOptimalSet.TotalReplicasCapacity*int(factor) {
				timeEnd = timeIntervals[idx].TimeStart
			}
			idx+=1
		}
		durationNewVMSet :=  timeEnd.Sub(timeStart).Seconds()
		reconfigCostNew := p.calculateReconfigurationCost(newSet)

		//Compute duration for current set
		jdx := indexTimeInterval
		for jdx < lenInterval {
			if timeIntervals[jdx].Requests > currentOptimalSet.TotalReplicasCapacity*int(factor) {
				timeEnd = timeIntervals[jdx].TimeStart
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

//Compare 2 VM Sets and returns a set of the new machines and one set with the machines that were removed
func deltaVMSet(current types.VMScale, candidate types.VMScale) (types.VMScale, types.VMScale){
		delta := types.VMScale{}
		startSet := types.VMScale{}
		shutdownSet := types.VMScale{}

		for k,_ :=  range current{
			if _,ok := candidate[k]; ok {
				delta[k] = -1 * (current[k] - candidate[k])
				if (delta[k]> 0) {
					startSet[k] = delta[k]
				} else if (delta[k] < 0) {
					shutdownSet[k] = delta[k]
				}
			}else {
				delta[k] = -1 * current[k]
				shutdownSet[k] = current[k]
			}
		}

		for k,_ :=  range candidate {
			if _,ok := current[k]; !ok {
				delta[k] = candidate[k]
				startSet[k] = candidate[k]
			}
		}
		return startSet, shutdownSet
}

//Calculate the cost of a reconfiguration
func(p DeltaRepackedPolicy) calculateReconfigurationCost(newSet types.VMScale) float64 {
	//Compute reconfiguration cost
	_, deletedVMS := deltaVMSet(p.currentState.VMs, newSet)
	reconfigTime := ComputeVMTerminationTime(p.mapVMProfiles, deletedVMS)

	deletedSet := OptimalVMSet{VMSet : deletedVMS}
	deletedSet.setValues(p.mapVMProfiles)
	return deletedSet.Cost * float64(reconfigTime)
}

//TODO: Fix
func (p DeltaRepackedPolicy) setConfiguration(configurations *[]types.Configuration, state types.State, timeStart time.Time, timeEnd time.Time, name string, totalServicesBootingTime int){
	//Adjust termination times for resources configuration
	terminationTime := ComputeVMTerminationTime(p.mapVMProfiles, state.VMs)
	finishTime := timeEnd.Add(time.Duration(terminationTime) * time.Second)

	nConfigurations := len(*configurations)
	if nConfigurations >= 1 && state.Equal(((*configurations)[nConfigurations-1]).State) {
		(*configurations)[nConfigurations-1].TimeEnd = finishTime
		//configurations[nConfigurations-1].Metrics.OverProvision = avgOver/ nPeaksInConfiguration
		p.currentState = state
	} else {
		//Adjust booting times for resources configuration
		transitionTime := ComputeVMBootingTime(p.mapVMProfiles, state.VMs)
		startTime := timeStart.Add(-1 * time.Duration(transitionTime) * time.Second)       //Booting time VM
		startTime = startTime.Add(-1 * time.Duration(totalServicesBootingTime) * time.Second) //Start time containers
		state.LaunchTime = startTime
		state.Name = strconv.Itoa(nConfigurations) + "__" + name + "__" + startTime.Format(util.TIME_LAYOUT)
		*configurations = append(*configurations,
			types.Configuration {
				State:     state,
				TimeStart: timeStart,
				TimeEnd:   finishTime,
				/*Metrics: types.Metrics{
					OverProvision:  over,
					UnderProvision: under,
				},*/
			})
		//nPeaksInConfiguration = 0
		//avgOver = 0
		p.currentState = state
	}
}