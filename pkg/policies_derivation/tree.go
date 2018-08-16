package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/util"
	"strconv"
	"fmt"
	"sort"
	"github.com/Cloud-Pie/SPDT/config"
)

type TreePolicy struct {
	algorithm  		string               //Algorithm's name
	limitNVMS  		int                  //Max number of vms of the same type in a cluster
	timeWindow 		TimeWindowDerivation //Algorithm used to process the forecasted time serie
	currentState	types.State			 //Current State
	mapVMProfiles map[string]types.VmProfile
	sysConfiguration	config.SystemConfiguration
}

type Node struct {
	NReplicas	int
	vmType	string
	children []*Node
	vmScale map[string]int
}

type Tree struct {
	Root *Node
}

func (p TreePolicy) CreatePolicies(processedForecast types.ProcessedForecast, serviceProfile types.ServiceProfile) []types.Policy {

	policies := []types.Policy {}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}
	configurations := []types.Configuration {}
	underprovisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	for _, it := range processedForecast.CriticalIntervals {
		requests := it.Requests
		services :=  make(map[string]types.ServiceInfo)
		shouldScale := true

		//Select the performance profile that fits better
		performanceProfile := selectProfile(serviceProfile.PerformanceProfiles)
		//Compute number of replicas needed depending on requests
		nProfileCopies := int(math.Ceil(float64(requests) / float64(performanceProfile.TRN)))
		newNumServiceReplicas := nProfileCopies * performanceProfile.NumReplicas
		services[serviceProfile.Name] = types.ServiceInfo{
										Scale:  newNumServiceReplicas,
										CPU:    performanceProfile.Limit.NumCores,
										Memory: performanceProfile.Limit.Memory, }

		state := types.State{}
		state.Services = services

		//Compare with current state . It Assumes that there is only 1 service
		var currentNReplicas int
		for _,v := range p.currentState.Services {
			currentNReplicas = v.Scale
		}

		diffReplicas := newNumServiceReplicas - currentNReplicas
		var vms types.VMScale
		if diffReplicas == 0 {
				shouldScale = false	//no need to scale
				vms = p.currentState.VMs
				state.VMs = vms
		} else if diffReplicas > 0 {
			//Need to increase resources
			var totalPossibleReplica int
			for k, v := range p.currentState.VMs {
				totalPossibleReplica += maxReplicasCapacityInVM(p.mapVMProfiles[k],performanceProfile.Limit) * v
			}
			//The new replicas fit into the current set of VMs
			if diffReplicas <= totalPossibleReplica - currentNReplicas {
				vms = p.currentState.VMs
				state.VMs = vms
			} else {
				//Find new suitable Vm(s) to cover the number of replicas missing.
				vms = p.findSuitableVMs(diffReplicas,performanceProfile.Limit)
				vmsold := p.currentState.VMs
				//Merge the current configuration with configuration for the new replicas
				for k,v :=  range vmsold{
					if _,ok := vms[k]; ok {
						vms[k] += v
					}else {
						vms[k] = v
					}
				}
				state.VMs = vms
			}

		} else {
			//Need to decrease resources
			//vms = p.currentState.VMs
			vms = p.findSuitableVMs(newNumServiceReplicas,performanceProfile.Limit)
			//vms = p.removeVMs(mapVMProfiles, p.currentState.VMs, -1*diffReplicas,performanceProfile.Limit)
			state.VMs = vms
		}

		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		totalServicesBootingTime := performanceProfile.BootTimeSec
		stateLoadCapacity := float64(newNumServiceReplicas/performanceProfile.NumReplicas) * performanceProfile.TRN
		setConfiguration(&configurations,state,timeStart,timeEnd,serviceProfile.Name, totalServicesBootingTime, p.sysConfiguration, stateLoadCapacity)

		//Adjust termination times for resources configuration
		terminationTime := computeVMTerminationTime(vms, p.sysConfiguration)
		finishTime := it.TimeEnd.Add(time.Duration(terminationTime) * time.Second)

		nConfigurations := len(configurations)
		if nConfigurations >= 1 && state.Equal(configurations[nConfigurations-1].State) || !shouldScale {
			configurations[nConfigurations-1].TimeEnd = finishTime
			p.currentState = state
		} else {
			//Adjust booting times for resources configuration
			transitionTime := computeVMBootingTime(vms, p.sysConfiguration)                       //TODO: It should include a booting rate
			totalServicesBootingTime := performanceProfile.BootTimeSec                            //TODO: It should include a booting rate
			startTime := it.TimeStart.Add(-1 * time.Duration(transitionTime) * time.Second)       //Booting time VM
			startTime = startTime.Add(-1 * time.Duration(totalServicesBootingTime) * time.Second) //Start time containers
			state.LaunchTime = startTime
			state.Name = strconv.Itoa(nConfigurations) + "__" + serviceProfile.Name + "__" + startTime.Format(util.TIME_LAYOUT)
			configurations = append(configurations,
				types.Configuration {
					State:          state,
					TimeStart:      it.TimeStart,
					TimeEnd:        finishTime,
				})
			p.currentState = state
		}
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

func (p TreePolicy) findSuitableVMs(nReplicas int, limit types.Limit) types.VMScale {
	tree := &Tree{}
	node := new(Node)
	node.NReplicas = nReplicas
	node.vmScale = make(map[string]int)
	tree.Root = node
	mapVMScaleList := []map[string]int {}
	p.buildTree(tree.Root, p.mapVMProfiles,nReplicas,limit, &mapVMScaleList)

	sort.Slice(mapVMScaleList, func(i, j int) bool {
		return MapPrice(mapVMScaleList[i], p.mapVMProfiles) <=  MapPrice(mapVMScaleList[j], p.mapVMProfiles)
	})
	return mapVMScaleList[0]
}

//TODO: Delete after debugging
func Drucken (n *Node, level int) {
	fmt.Printf("%d - %s --%d\n", level,n.vmType, n.NReplicas)
	for k,v := range n.vmScale {
		fmt.Printf("%s : %d; ", k,v)
	}
	fmt.Printf("\n")
	for _,v := range n.children {
		Drucken(v,level+1)
	}
}

func (p TreePolicy) buildTree(node *Node, mapVMProfiles map[string]types.VmProfile, nReplicas int, limit types.Limit, vmScaleList *[]map[string]int) *Node {
	if node.NReplicas == 0 {
		return node
	}
	for k,v := range mapVMProfiles {
		maxReplicas := maxReplicasCapacityInVM(mapVMProfiles[v.Type], limit)
		if maxReplicas >= nReplicas {
			newNode := new(Node)
			newNode.vmType = k
			newNode.NReplicas = 0
			newNode.vmScale = copyMap(node.vmScale)
			if _, ok := newNode.vmScale[newNode.vmType]; ok {
				newNode.vmScale[newNode.vmType] = newNode.vmScale[newNode.vmType]+1
			} else {
				newNode.vmScale[newNode.vmType] = 1
			}
			node.children = append(node.children, newNode)
			*vmScaleList = append(*vmScaleList, newNode.vmScale)
			//return node
		} else if maxReplicas > 0 {
			newNode := new(Node)
			newNode.vmType = k
			newNode.NReplicas = nReplicas-maxReplicas
			newNode.vmScale = copyMap(node.vmScale)
			if _, ok := newNode.vmScale[newNode.vmType]; ok {
				newNode.vmScale[newNode.vmType] = newNode.vmScale[newNode.vmType] + 1
			} else {
				newNode.vmScale[newNode.vmType] = 1
			}
			newNode = p.buildTree(newNode, mapVMProfiles, nReplicas-maxReplicas, limit, vmScaleList)
			node.children = append(node.children, newNode)
		}
	}
	return node
}



func MapPrice( m map[string]int, mapVMProfiles map [string]types.VmProfile ) float64{
	price := float64(0.0)
	for k,v := range m {
		price += mapVMProfiles[k].Pricing.Price * float64(v)
	}
	return price
}

func (p TreePolicy)removeVMs(mapVMProfiles map[string] types.VmProfile, currentConfig map[string] int, nReplicas int, limit types.Limit) types.VMScale{
	//Find the type with more machines
	nVMS := 0
	var typeVM string
	for k,v := range currentConfig {
		if v > nVMS {
			nVMS = v
			typeVM = k
		}
	}

	maxNReplicas := maxReplicasCapacityInVM(mapVMProfiles[typeVM],limit)
	if maxNReplicas == nReplicas {
		//Remove only 1 VM
		currentConfig[typeVM] = nVMS - 1
		return currentConfig
	} else if maxNReplicas > nReplicas {
		//I cannot remove any machine, keep the same configuration
		return currentConfig
	}

	totalMaxNReplicas := maxNReplicas * nVMS
	if totalMaxNReplicas > nReplicas {
		currentConfig[typeVM] = nVMS - nReplicas / maxNReplicas
	}

	/*var totalPossibleReplica int
	for k, v := range p.currentState.VMs {
		totalPossibleReplica += maxReplicasCapacityInVM(mapVMProfiles[k],limit) * v
	}*/

	return currentConfig
}