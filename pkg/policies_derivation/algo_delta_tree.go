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


	for _, it := range processedForecast.CriticalIntervals {
		//Compute number of replicas needed depending on requests
		performanceProfile := selectProfile(serviceProfile.PerformanceProfiles, it.Requests, underprovisionAllowed)
		computeCapacity(&p.sortedVMProfiles, performanceProfile, &p.mapVMProfiles)

		newNumServiceReplicas := performanceProfile.TRNConfiguration[0].NumberReplicas
		currentNumberReplicas := p.currentState.Services[p.sysConfiguration.ServiceName].Scale

		var vmSet types.VMScale
		deltaNumberReplicas := newNumServiceReplicas - currentNumberReplicas

		if deltaNumberReplicas == 0 {
				vmSet = p.currentState.VMs
		} else if deltaNumberReplicas > 0 {
			//Need to increase resources
			currentReplicasCapacity := p.currentState.VMs.ReplicasCapacity(p.mapVMProfiles)
			//Validate if the current configuration is able to handle the new replicas
			if currentReplicasCapacity >= currentNumberReplicas + deltaNumberReplicas {
				vmSet = p.currentState.VMs
			} else {
				//Find new suitable Vm(s) to cover the number of replicas missing.
				vmSet = p.findSuitableVMs(deltaNumberReplicas)
				if underprovisionAllowed {
					//TODO:implement
				}
				//Merge the current configuration with configuration for the new replicas
				vmSet.Merge(p.currentState.VMs)
			}
		} else {
			//Need to decrease resources
			vmSet = p.removeVMs(p.currentState.VMs, -1*deltaNumberReplicas)
		}

		services :=  make(map[string]types.ServiceInfo)
		services[serviceProfile.Name] = types.ServiceInfo{
			Scale:  newNumServiceReplicas,
			CPU:    performanceProfile.Limit.NumberCores,
			Memory: performanceProfile.Limit.MemoryGB,
		}
		state := types.State{}
		state.Services = services
		cleanKeys(vmSet)
		state.VMs = vmSet
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		totalServicesBootingTime := performanceProfile.TRNConfiguration[0].BootTimeSec
		stateLoadCapacity := performanceProfile.TRNConfiguration[0].TRN
		setConfiguration(&configurations,state,timeStart,timeEnd,serviceProfile.Name, totalServicesBootingTime, p.sysConfiguration, stateLoadCapacity)
		p.currentState = state
	}

	//Add new policy
	parameters := make(map[string]string)
	parameters[types.METHOD] = "horizontal"
	parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(true)
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

func (p TreePolicy) findSuitableVMs(nReplicas int) types.VMScale {
	tree := &Tree{}
	node := new(Node)
	node.NReplicas = nReplicas
	node.vmScale = make(map[string]int)
	tree.Root = node
	mapVMScaleList := []types.VMScale {}
	p.buildTree(tree.Root,nReplicas,&mapVMScaleList)

	//Drucken (node, 1)
	sort.Slice(mapVMScaleList, func(i, j int) bool {
		costi := mapVMScaleList[i].Cost(p.mapVMProfiles)
		costj := mapVMScaleList[j].Cost(p.mapVMProfiles)
		if costi < costj {
			return true
		} else if costi ==  costj {
			return mapVMScaleList[i].TotalVMs() >= mapVMScaleList[j].TotalVMs()
		}
		return false
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

func (p TreePolicy)removeVMs(currentVMSet types.VMScale, nReplicas int) types.VMScale{
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
