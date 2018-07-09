package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/util"
	"strconv"
	db "github.com/Cloud-Pie/SPDT/storage/policies"
	"fmt"
	"sort"
)

type TreePolicy struct {
	algorithm  		string               //Algorithm's name
	limitNVMS  		int                  //Max number of vms of the same type in a cluster
	timeWindow 		TimeWindowDerivation //Algorithm used to process the forecasted time serie
	currentState	types.State			 //Current State
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



func (policy TreePolicy) CreatePolicies(processedForecast types.ProcessedForecast, mapVMProfiles map[string]types.VmProfile, serviceProfile types.ServiceProfile) []types.Policy {

	policies := []types.Policy {}
	newPolicy := types.Policy{}
	newPolicy.StartTimeDerivation = time.Now()
	configurations := []types.Configuration {}
	totalOverProvision := float32(0.0)
	totalUnderProvision := float32(0.0)
	peaksInConf := float32(0.0)
	avgOver := float32(0.0)

	for _, it := range processedForecast.CriticalIntervals {
		requests := it.Requests
		services := [] types.Service{}

		//Select the performance profile that fits better
		performanceProfile := policy.selectProfile(serviceProfile.PerformanceProfiles)
		//Compute number of replicas needed depending on requests
		nProfileCopies := int(math.Ceil(float64(requests) / float64(performanceProfile.TRN)))
		nServiceReplicas := nProfileCopies * performanceProfile.NumReplicas
		services = append(services, types.Service{Name:serviceProfile.Name,Scale:nServiceReplicas})

		//Compute under/over provision
		diff := (nProfileCopies * performanceProfile.TRN) - requests		//TODO:Fix Wrong calculation
		var over, under float32
		if diff >= 0 {
			over = float32(diff*100/requests)
			under = 0
		}else{
			over = 0
			under = -1*float32(diff*100/requests)
		}
		peaksInConf+=1
		avgOver+=over

		//Set total resource limit needed
		limit := types.Limit{}
		limit.Memory = performanceProfile.Limit.Memory * float64(nServiceReplicas)
		limit.NumCores = performanceProfile.Limit.NumCores * float64(nServiceReplicas)

		state := types.State{}
		state.Services = services

		//Compare with current state
		currentNReplicas := policy.currentState.Services[0].Scale
		diffReplicas := nServiceReplicas - currentNReplicas
		var vms []types.VmScale
		if diffReplicas == 0 {
			break	//no need to scale
		}else if diffReplicas > 0 {
			//Need to increase resources
			var totalPossibleReplica int
			for _, vm := range policy.currentState.VMs {
				totalPossibleReplica += policy.maxReplicasInVM(mapVMProfiles[vm.Type], performanceProfile.Limit) * vm.Scale
			}
			//The new replicas fit into the current set of VMs
			if diffReplicas <= totalPossibleReplica - currentNReplicas {
				vms = policy.currentState.VMs
				state.VMs = vms
			}else {
				//Find new suitable Vm(s) depending on resources limit and current state
				vms = policy.findSuitableVMs(mapVMProfiles, diffReplicas,performanceProfile.Limit)
				vms = append(vms, policy.currentState.VMs...)
				state.VMs = vms
			}

		}else {
			//Need to decrease resources

		}

		totalServicesBootingTime := performanceProfile.BootTimeSec //TODO: It should include a booting rate

		//Adjust termination times for resources configuration
		terminationTime := ComputeVMTerminationTime(mapVMProfiles,vms)
		finishTime := it.TimeEnd.Add(time.Duration(terminationTime) * time.Second)

		nConfigurations := len(configurations)
		if nConfigurations >= 1 && state.Equal(configurations[nConfigurations-1].State) {
			configurations[nConfigurations-1].TimeEnd = finishTime
			configurations[nConfigurations-1].OverProvision = avgOver/peaksInConf
		} else {
			//Adjust booting times for resources configuration
			transitionTime := ComputeVMBootingTime(mapVMProfiles, vms)                            //TODO: It should include a booting rate
			startTime := it.TimeStart.Add(-1 * time.Duration(transitionTime) * time.Second)       //Booting time VM
			startTime = startTime.Add(-1 * time.Duration(totalServicesBootingTime) * time.Second) //Start time containers
			state.LaunchTime = startTime
			state.Name = strconv.Itoa(nConfigurations) + "__" + serviceProfile.Name + "__" + startTime.Format(util.TIME_LAYOUT)
			configurations = append(configurations,
				types.Configuration{
					State:          state,
					TimeStart:      it.TimeStart,
					TimeEnd:        finishTime,
					OverProvision:  over,
					UnderProvision: under,
				})
			peaksInConf = 0
			avgOver = 0
		}
		totalOverProvision += over
		totalUnderProvision += under
	}

	totalConfigurations := len(processedForecast.CriticalIntervals)
	//Add new policy
	newPolicy.Configurations = configurations
	newPolicy.FinishTimeDerivation = time.Now()
	newPolicy.Algorithm = policy.algorithm
	newPolicy.ID = bson.NewObjectId()
	newPolicy.TotalOverProvision = totalOverProvision/float32(totalConfigurations)
	newPolicy.TotalUnderProvision = totalUnderProvision/float32(totalConfigurations)
	//store policy
	db.Store(newPolicy)
	policies = append(policies, newPolicy)
	return policies
}

func (policy TreePolicy) selectProfile(performanceProfiles []types.PerformanceProfile) types.PerformanceProfile {
	//In a naive case, select the one with rank 1
	for _,p := range performanceProfiles {
		if p.RankWithLimits == 1 {
			return p
		}
	}
	return performanceProfiles[0]
}

func (policy TreePolicy) findSuitableVMs(mapVMProfiles map[string]types.VmProfile, nReplicas int, limit types.Limit) []types.VmScale {
	vmscale := []types.VmScale{}
	tree := &Tree{}
	node := new(Node)
	node.NReplicas = nReplicas
	node.vmScale = make(map[string]int)
	tree.Root = node
	mapVMScaleList := []map[string]int {}
	policy.buildTree(tree.Root, mapVMProfiles,nReplicas,limit, &mapVMScaleList)

	sort.Slice(mapVMScaleList, func(i, j int) bool {
		return MapPrice(mapVMScaleList[i], mapVMProfiles) <  MapPrice(mapVMScaleList[j], mapVMProfiles)
	})
	for k,v := range mapVMScaleList[0]{
		vmscale = append(vmscale, types.VmScale{Type: k, Scale:v})
	}
	return vmscale
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

func (policy TreePolicy) maxReplicasInVM(vmProfile types.VmProfile, limit types.Limit) int {
	m := float64(vmProfile.NumCores) / float64(limit.NumCores)
	n := float64(vmProfile.Memory) / float64(limit.Memory)
	nScale := math.Min(n,m)
	return int(nScale)
}

func (policy TreePolicy) buildTree(node *Node, mapVMProfiles map[string]types.VmProfile, nReplicas int, limit types.Limit, vmScaleList *[]map[string]int) *Node {

	if node.NReplicas == 0 {
		return node
	}
	for k,v := range mapVMProfiles {
		maxReplicas := policy.maxReplicasInVM(mapVMProfiles[v.Type], limit)
		if maxReplicas >= nReplicas {
			newNode := new(Node)
			newNode.vmType = k
			newNode.NReplicas = 0
			newNode.vmScale = CopyMap(node.vmScale)
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
			newNode.vmScale = CopyMap(node.vmScale)
			if _, ok := newNode.vmScale[newNode.vmType]; ok {
				newNode.vmScale[newNode.vmType] = newNode.vmScale[newNode.vmType] + 1
			} else {
				newNode.vmScale[newNode.vmType] = 1
			}
			newNode = policy.buildTree(newNode, mapVMProfiles, nReplicas-maxReplicas, limit, vmScaleList)
			node.children = append(node.children, newNode)
		}
	}
	return node
}

func CopyMap( m map[string]int) map[string]int {
	newM := make(map[string] int)
	for k,v := range m {
		newM[k]=v
	}
	return newM
}

func MapPrice( m map[string]int, mapVMProfiles map [string]types.VmProfile ) float64{
	price := float64(0.0)
	for k,v := range m {
		price += mapVMProfiles[k].Pricing.Price * float64(v)
	}
	return price
}