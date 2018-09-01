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
	"github.com/Cloud-Pie/SPDT/util"
)

/*
	Constructs different VM clusters (heterogeneous included) to add resources every time the workload
	increases in a factor of deltaLoad.
 */
type TreePolicy struct {
	algorithm  		string               //Algorithm's name
	limitNVMS  		int                  //Max number of vms of the same type in a cluster
	timeWindow 		TimeWindowDerivation //Algorithm used to process the forecasted time serie
	currentState	types.State			 //Current State
	sortedVMProfiles []types.VmProfile    			//List of VM profiles sorted by price
	mapVMProfiles map[string]types.VmProfile
	sysConfiguration	config.SystemConfiguration
}

/*
	Node that represents a candidate option to scale
*/
type Node struct {
	NReplicas	int
	vmType	string
	children []*Node
	vmScale types.VMScale
}

/**
	Tree structure used to create different combinations of VM types
 */
type Tree struct {
	Root *Node
}

/* Derive a list of policies using this approach
	in:
		@processedForecast
	out:
		[] Policy. List of type Policy
*/
func (p TreePolicy) CreatePolicies(processedForecast types.ProcessedForecast) []types.Policy {

	policies := []types.Policy {}
	configurations := []types.ScalingConfiguration{}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	containerResizeEnabled := p.sysConfiguration.PolicySettings.PodsResizeAllowed

	for _, it := range processedForecast.CriticalIntervals {

		var vmSet types.VMScale
		var newNumServiceReplicas int
		var resourceLimits types.Limit
		var totalServicesBootingTime int
		var stateLoadCapacity float64

		//Current configuration
		totalLoad := it.Requests
		serviceToScale := p.currentState.Services[p.sysConfiguration.ServiceName]
		currentContainerLimits := types.Limit{ MemoryGB:serviceToScale.Memory, NumberCores:serviceToScale.CPU }
		currentNumberReplicas := serviceToScale.Scale
		currentLoadCapacity := configurationLoadCapacity(currentNumberReplicas, currentContainerLimits)
		deltaLoad := totalLoad - currentLoadCapacity

		if deltaLoad == 0 {
			vmSet = p.currentState.VMs
			newNumServiceReplicas = currentNumberReplicas
			resourceLimits  = currentContainerLimits
			stateLoadCapacity = currentLoadCapacity
		} else {
			//Alternative configuration
			ProfileCurrentLimits := selectProfileWithLimits(totalLoad, currentContainerLimits, false)
			newNumServiceReplicas = ProfileCurrentLimits.PerformanceProfile.NumberReplicas
			resourceLimits  = ProfileCurrentLimits.Limits
			stateLoadCapacity = ProfileCurrentLimits.PerformanceProfile.TRN
			totalServicesBootingTime = ProfileCurrentLimits.PerformanceProfile.BootTimeSec

			if deltaLoad > 0 {
				computeCapacity(&p.sortedVMProfiles, ProfileCurrentLimits.Limits, &p.mapVMProfiles)
				currentReplicasCapacity := p.currentState.VMs.ReplicasCapacity(p.mapVMProfiles)
				if currentReplicasCapacity >= ProfileCurrentLimits.PerformanceProfile.NumberReplicas {
					//case 1: Increases number of replicas but VMS remain the same
					vmSet = p.currentState.VMs
				} else {
					if underProvisionAllowed {
						//case 2: search a new service profile with underprovisioning that possible fit into the
						//current VM set
						ProfileNewLimits := selectProfile(totalLoad, underProvisionAllowed)
						computeCapacity(&p.sortedVMProfiles, ProfileNewLimits.Limit, &p.mapVMProfiles)
						currentReplicasCapacity := p.currentState.VMs.ReplicasCapacity(p.mapVMProfiles)
						if currentReplicasCapacity >= ProfileNewLimits.TRNConfiguration[0].NumberReplicas {
							vmSet = p.currentState.VMs
							newNumServiceReplicas = ProfileNewLimits.TRNConfiguration[0].NumberReplicas
							resourceLimits  = ProfileNewLimits.Limit
							stateLoadCapacity = ProfileNewLimits.TRNConfiguration[0].TRN
							totalServicesBootingTime = ProfileNewLimits.TRNConfiguration[0].BootTimeSec
						}
					} else {
						//case 3: Increases number of VMS. Find new suitable Vm(s) to cover the number of replicas missing.
						deltaNumberReplicas := newNumServiceReplicas - currentNumberReplicas
						if deltaNumberReplicas > 0 {
							vmSet = p.FindSuitableVMs(deltaNumberReplicas, ProfileCurrentLimits.Limits)
							//Merge the current configuration with configuration for the new replicas
							vmSet.Merge(p.currentState.VMs)
						}else {
							vmSet = p.currentState.VMs
						}
					}
				}
			} else {
				deltaReplicas := currentNumberReplicas - ProfileCurrentLimits.PerformanceProfile.NumberReplicas
				vmSet = p.removeVMs(p.currentState.VMs, deltaReplicas, currentContainerLimits)
			}
		}

		services :=  make(map[string]types.ServiceInfo)
		services[ p.sysConfiguration.ServiceName] = types.ServiceInfo{
			Scale:  newNumServiceReplicas,
			CPU:    resourceLimits.NumberCores,
			Memory: resourceLimits.MemoryGB,
		}
		state := types.State{}
		state.Services = services
		cleanKeys(vmSet)
		state.VMs = vmSet
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		setConfiguration(&configurations,state,timeStart,timeEnd, p.sysConfiguration.ServiceName, totalServicesBootingTime, p.sysConfiguration, stateLoadCapacity)
		p.currentState = state
	}

	//Add new policy
	parameters := make(map[string]string)
	parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
	parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(true)
	parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)
	parameters[types.ISRESIZEPODS] = strconv.FormatBool(containerResizeEnabled)
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
	@limits = Resources (CPU, Memory) constraints to configure the containers.
 out:
	@VMScale with the suggested number of VMs for that type
*/
func (p TreePolicy) FindSuitableVMs(numberReplicas int, limits types.Limit) types.VMScale {
	tree := &Tree{}
	node := new(Node)
	node.NReplicas = numberReplicas
	node.vmScale = make(map[string]int)
	tree.Root = node
	mapVMScaleList := []types.VMScale {}
	computeVMsCapacity(limits,&p.mapVMProfiles)
	buildTree(tree.Root, numberReplicas,&mapVMScaleList, p.mapVMProfiles)

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


/*
	Remove VMs from the current set of VMs, the resources that hosts the defined number of container replicas
	underprovisioning is not allowed.
	in:
		@currentVMSet
		@numberReplicas
		@limits
	out:
		@VMScale
*/
func (p TreePolicy)removeVMs(currentVMSet types.VMScale, numberReplicas int, limits types.Limit) types.VMScale {
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
		cap := maxReplicasCapacityInVM(p.mapVMProfiles[kv.Key], limits)
		if  newVMSet.TotalVMs() > 1 {
			if numberReplicas == cap && kv.Value > 0{
				//Remove 1 VM of this type
				newVMSet[kv.Key]= newVMSet[kv.Key] - 1
				break
			} else if numberReplicas > cap && kv.Value * cap > numberReplicas {
				rmvVM := int(math.Floor(float64(numberReplicas /cap)))
				newVMSet[kv.Key]= newVMSet[kv.Key] - rmvVM
				break
			} else if numberReplicas > cap && kv.Value > 0{
				newVMSet[kv.Key]= newVMSet[kv.Key] - 1
				numberReplicas -= cap
			}
		}
	}
	return  newVMSet
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
func (p TreePolicy) selectContainersConfig(currentLimits types.Limit, profileCurrentLimits types.TRNConfiguration,
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