package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"fmt"
	"sort"
	"github.com/Cloud-Pie/SPDT/util"
)

/*
	Constructs different VM clusters to add resources every time the workload
	increases in a factor of deltaLoad.
 */
type DeltaLoadPolicy struct {
	algorithm  		string               //Algorithm's name
	currentState	types.State			 //Current State
	mapVMProfiles map[string]types.VmProfile
	sysConfiguration	util.SystemConfiguration
}

/* Derive a list of policies using this approach
	in:
		@processedForecast
	out:
		[] Policy. List of type Policy
*/
func (p DeltaLoadPolicy) CreatePolicies(processedForecast types.ProcessedForecast) []types.Policy {
	log.Info("Derive policies with %s algorithm", p.algorithm)
	policies := []types.Policy {}
	scalingActions := []types.ScalingAction{}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}

	for _, it := range processedForecast.CriticalIntervals {
		var vmSet types.VMScale
		var newNumPods int
		var podLimits types.Limit
		var totalServicesBootingTime float64
		var stateLoadCapacity float64

		//Current configuration
		totalLoad := it.Requests
		serviceToScale := p.currentState.Services[p.sysConfiguration.MainServiceName]
		currentPodLimits := types.Limit{ MemoryGB:serviceToScale.Memory, CPUCores:serviceToScale.CPU }
		currentNumPods := serviceToScale.Scale
		currentLoadCapacity := getStateLoadCapacity(currentNumPods, currentPodLimits).MSCPerSecond
		deltaLoad := totalLoad - currentLoadCapacity

		if deltaLoad == 0 {
			//case 0: Resource configuration do not change
			vmSet = p.currentState.VMs
			newNumPods = currentNumPods
			podLimits = currentPodLimits
			stateLoadCapacity = currentLoadCapacity
		} else {
			//Alternative configuration to handle the total load
			profileCurrentLimits,_ := estimatePodsConfiguration(totalLoad, currentPodLimits)
			newNumPods = profileCurrentLimits.MSCSetting.Replicas
			podLimits = profileCurrentLimits.Limits
			stateLoadCapacity = profileCurrentLimits.MSCSetting.MSCPerSecond
			totalServicesBootingTime = profileCurrentLimits.MSCSetting.BootTimeSec

			if deltaLoad > 0 {
				//case 1: Increase resources
				computeVMsCapacity(profileCurrentLimits.Limits,&p.mapVMProfiles )
				currentPodsCapacity := p.currentState.VMs.ReplicasCapacity(p.mapVMProfiles)
				if currentPodsCapacity >= newNumPods {
					//case 1.1: Increases number of replicas with the current limit resources but VMS remain the same
					vmSet = p.currentState.VMs
				}else{
					//case 1.2: Increases number of VMS. Find new suitable Vm(s) to cover the number of replicas missing.
					deltaNumPods := newNumPods - currentPodsCapacity
					vmSet = p.FindSuitableVMs(deltaNumPods, profileCurrentLimits.Limits)
					vmSet.Merge(p.currentState.VMs)
				}
			} else {
				//case 2: delta load is negative, some resources should be terminated
				//deltaNumPods := currentNumPods - newNumPods
				vmSet = p.releaseVMs(p.currentState.VMs, newNumPods, currentPodLimits)
			}
		}

		services :=  make(map[string]types.ServiceInfo)
		services[ p.sysConfiguration.MainServiceName] = types.ServiceInfo {
			Scale:  newNumPods,
			CPU:    podLimits.CPUCores,
			Memory: podLimits.MemoryGB,
		}
		state := types.State{}
		state.Services = services
		cleanKeys(vmSet)
		state.VMs = vmSet
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		stateLoadCapacity = adjustGranularity(systemConfiguration.ForecastComponent.Granularity, stateLoadCapacity)
		setScalingSteps(&scalingActions,p.currentState,state,timeStart,timeEnd, totalServicesBootingTime, stateLoadCapacity)
		p.currentState = state
	}

	//Add new policy
	parameters := make(map[string]string)
	parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
	parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(true)
	parameters[types.ISRESIZEPODS] = strconv.FormatBool(false)
	numConfigurations := len(scalingActions)
	newPolicy.ScalingActions = scalingActions
	newPolicy.Algorithm = p.algorithm
	newPolicy.ID = bson.NewObjectId()
	newPolicy.Status = types.DISCARTED	//State by default
	newPolicy.Parameters = parameters
	newPolicy.Metrics.NumberScalingActions = numConfigurations
	newPolicy.Metrics.FinishTimeDerivation = time.Now()
	newPolicy.Metrics.DerivationDuration = newPolicy.Metrics.FinishTimeDerivation.Sub(newPolicy.Metrics.StartTimeDerivation).Seconds()
	newPolicy.TimeWindowStart = scalingActions[0].TimeStart
	newPolicy.TimeWindowEnd = scalingActions[numConfigurations -1].TimeEnd
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
func (p DeltaLoadPolicy) FindSuitableVMs(numberReplicas int, limits types.Limit) types.VMScale {
	vmSet, _ := buildHomogeneousVMSet(numberReplicas,limits, p.mapVMProfiles)
	/*hetVMSet,_ := buildHeterogeneousVMSet(numberReplicas, limits, p.mapVMProfiles)
	costi := hetVMSet.Cost(p.mapVMProfiles)
	costj := vmSet.Cost(p.mapVMProfiles)
	if costi < costj {
		vmSet = hetVMSet
	}
	*/
	return vmSet
}


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
	under provisioning is not allowed.
	in:
		@currentVMSet
		@numberReplicas
		@limits
	out:
		@VMScale
*/
func (p DeltaLoadPolicy) releaseVMs(vmSet types.VMScale, numberPods int, limits types.Limit) types.VMScale {
	computeVMsCapacity(limits, &p.mapVMProfiles)

	var currentVMSet types.VMScale
	currentVMSet = copyMap(vmSet)
	newVMSet :=  make(map[string] int)

	type mapTypeCapacity struct {
		Key   string
		Value int
	}
	//Creates a list sorted by the number of machines per type
	var listMaps []mapTypeCapacity
	for k, v := range currentVMSet {
		listMaps = append(listMaps, mapTypeCapacity{k, v})
	}
	sort.Slice(listMaps, func(i, j int) bool { return listMaps[i].Value < listMaps[j].Value })

	for _,v := range listMaps {
		i:=0
		cap := p.mapVMProfiles[v.Key].ReplicasCapacity
		for i < v.Value && numberPods > 0{
			numberPods = numberPods - cap
			newVMSet[v.Key] = newVMSet[v.Key] + 1
			i+=1
		}
		if numberPods <= 0 {
			break
		}
	}

	return newVMSet
}
