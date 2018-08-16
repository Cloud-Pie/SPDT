package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
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
	sortedVMProfiles []types.VmProfile    			//List of VM profiles sorted by price
	mapVMProfiles map[string]types.VmProfile
	sysConfiguration	config.SystemConfiguration
}

type Node struct {
	NReplicas	int
	vmType	string
	children []*Node
	vmScale types.VMScale
}

type Tree struct {
	Root *Node
}

func (p TreePolicy) CreatePolicies(processedForecast types.ProcessedForecast, serviceProfile types.ServiceProfile) []types.Policy {

	policies := []types.Policy {}
	configurations := []types.Configuration {}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}
	underprovisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	//Select the performance profile that fits better
	performanceProfile := selectProfile(serviceProfile.PerformanceProfiles)
	//calculate the capacity of services replicas to each VM type
	computeCapacity(&p.sortedVMProfiles, performanceProfile, &p.mapVMProfiles)
	for _, it := range processedForecast.CriticalIntervals {
		requests := it.Requests
		services :=  make(map[string]types.ServiceInfo)
		//Compute number of replicas needed depending on requests
		newNumServiceReplicas := int(math.Ceil(requests / performanceProfile.TRN)) * performanceProfile.NumReplicas

		//Compare with current state . It Assumes that there is only 1 service
		var currentNumberReplicas int
		for _,v := range p.currentState.Services { currentNumberReplicas = v.Scale }
		var vmSet types.VMScale
		deltaNumberReplicas := newNumServiceReplicas - currentNumberReplicas

		if deltaNumberReplicas == 0 {
				vmSet = p.currentState.VMs
		} else if deltaNumberReplicas > 0 {
			//Need to increase resources
			currentOpt := VMSet{VMSet:p.currentState.VMs}
			currentOpt.setValues(p.mapVMProfiles)
			//Validate if the current configuration is able to handle the new replicas
			if currentOpt.TotalReplicasCapacity >= currentNumberReplicas + deltaNumberReplicas {
				vmSet = p.currentState.VMs
			} else {
				//Find new suitable Vm(s) to cover the number of replicas missing.
				vmSet = p.findSuitableVMs(deltaNumberReplicas,performanceProfile.Limit)
				vmsold := p.currentState.VMs
				//Merge the current configuration with configuration for the new replicas
				vmSet.Merge(vmsold)
			}
		} else {
			//Need to decrease resources
			//vmSet = p.currentState.VMs
			vmSet = p.findSuitableVMs(newNumServiceReplicas,performanceProfile.Limit)
			//vmSet = p.removeVMs(mapVMProfiles, p.currentState.VMs, -1*deltaNumberReplicas,performanceProfile.Limit)
		}

		services[serviceProfile.Name] = types.ServiceInfo{
			Scale:  newNumServiceReplicas,
			CPU:    performanceProfile.Limit.NumCores,
			Memory: performanceProfile.Limit.Memory,
		}
		state := types.State{}
		state.Services = services
		state.VMs = vmSet
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		totalServicesBootingTime := performanceProfile.BootTimeSec
		stateLoadCapacity := float64(newNumServiceReplicas/performanceProfile.NumReplicas) * performanceProfile.TRN
		setConfiguration(&configurations,state,timeStart,timeEnd,serviceProfile.Name, totalServicesBootingTime, p.sysConfiguration, stateLoadCapacity)
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

func (p TreePolicy) findSuitableVMs(nReplicas int, limit types.Limit) types.VMScale {
	tree := &Tree{}
	node := new(Node)
	node.NReplicas = nReplicas
	node.vmScale = make(map[string]int)
	tree.Root = node
	mapVMScaleList := []types.VMScale {}
	p.buildTree(tree.Root,nReplicas,&mapVMScaleList)

	sort.Slice(mapVMScaleList, func(i, j int) bool {
		map1 := VMSet{VMSet:mapVMScaleList[i]}
		map1.setValues(p.mapVMProfiles)
		map2 := VMSet{VMSet:mapVMScaleList[j]}
		map2.setValues(p.mapVMProfiles)
		return map1.Cost <=  map2.Cost
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

func (p TreePolicy) buildTree(node *Node, nReplicas int, vmScaleList *[]types.VMScale) *Node {
	if node.NReplicas == 0 {
		return node
	}
	for k,v := range p.mapVMProfiles {
		maxReplicas := v.ReplicasCapacity
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
			newNode = p.buildTree(newNode,nReplicas-maxReplicas, vmScaleList)
			node.children = append(node.children, newNode)
		}
	}
	return node
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