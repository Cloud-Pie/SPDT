package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"fmt"
	"sort"
	"github.com/Cloud-Pie/SPDT/util"
)


/* Derive a list of policies using this approach
	in:
		@processedForecast
	out:
		[] Policy. List of type Policy
*/
func (p DeltaLoadPolicy) CreatePolicies(processedForecast types.ProcessedForecast) []types.Policy {
	log.Info("Derive policies with %s algorithm", p.algorithm)
	policies := []types.Policy {}
	configurations := []types.ScalingStep{}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	percentageUnderProvision := p.sysConfiguration.PolicySettings.MaxUnderprovisionPercentage

	for _, it := range processedForecast.CriticalIntervals {
		var vmSet types.VMScale
		var newNumServiceReplicas int
		var resourceLimits types.Limit
		var totalServicesBootingTime float64
		var stateLoadCapacity float64

		//Current configuration
		totalLoad := it.Requests
		serviceToScale := p.currentState.Services[p.sysConfiguration.MainServiceName]
		currentContainerLimits := types.Limit{ MemoryGB:serviceToScale.Memory, CPUCores:serviceToScale.CPU }
		currentNumberReplicas := serviceToScale.Scale
		currentLoadCapacity := getStateLoadCapacity(currentNumberReplicas, currentContainerLimits).MSCPerSecond
		deltaLoad := totalLoad - currentLoadCapacity

		if deltaLoad == 0 {
			//case 0: Resource configuration do not change
			vmSet = p.currentState.VMs
			newNumServiceReplicas = currentNumberReplicas
			resourceLimits  = currentContainerLimits
			stateLoadCapacity = currentLoadCapacity
		} else {
			//Alternative configuration to handle the total load
			profileCurrentLimits := selectProfileByLimits(totalLoad, currentContainerLimits, false)
			newNumServiceReplicas = profileCurrentLimits.MSCSetting.Replicas
			resourceLimits  = profileCurrentLimits.Limits
			stateLoadCapacity = profileCurrentLimits.MSCSetting.MSCPerSecond
			totalServicesBootingTime = profileCurrentLimits.MSCSetting.BootTimeSec

			if deltaLoad > 0 {
				//case 1: Increase resources
				computeCapacity(&p.sortedVMProfiles, profileCurrentLimits.Limits, &p.mapVMProfiles)
				currentReplicasCapacity := p.currentState.VMs.ReplicasCapacity(p.mapVMProfiles)
				if currentReplicasCapacity >= profileCurrentLimits.MSCSetting.Replicas {
					//case 1.1: Increases number of replicas with the current limit resources but VMS remain the same
					vmSet = p.currentState.VMs
				} else {
					//search for a different profile that fit into the current VM set and handle total load
					/*profileNewLimits, candidateFound := findConfigOptionByContainerResize(p.currentState.VMs, totalLoad, p.mapVMProfiles)
					if candidateFound {
						//case 1.2: Change the configuration of limit resources for the containers but VMS remain the same
						vmSet = p.currentState.VMs
						profileCurrentLimits = profileNewLimits
					}else{*/
						//case 1.3: Increases number of VMS. Find new suitable Vm(s) to cover the number of replicas missing.
						//Keep the current resource limits for the containers
						deltaNumberReplicas := newNumServiceReplicas - currentNumberReplicas
						vmSet = p.FindSuitableVMs(deltaNumberReplicas, profileCurrentLimits.Limits)

						if underProvisionAllowed {
							//ProfileCurrentLimitsUnder := selectProfileByLimits(deltaLoad, currentContainerLimits, underProvisionAllowed)
							replicasUnder := deltaNumberReplicas-1
							if replicasUnder > 1 {
								vmSetUnder := p.FindSuitableVMs(replicasUnder,currentContainerLimits)
								costVMSetUnderProvision := vmSetUnder.Cost(p.mapVMProfiles)
								replicasUnder = newNumServiceReplicas - 1
								mscConfig := getStateLoadCapacity(replicasUnder,currentContainerLimits)
								if isUnderProvisionInRange(totalLoad, mscConfig.MSCPerSecond, percentageUnderProvision) &&
									costVMSetUnderProvision > 0 && costVMSetUnderProvision < vmSet.Cost(p.mapVMProfiles) {
									vmSet = vmSetUnder
									profileCurrentLimits.MSCSetting = mscConfig
								}
							}
						}
						//Merge the current configuration with configuration for the new replicas
						vmSet.Merge(p.currentState.VMs)
					//}
				}
			} else {
				//case 2: delta load is negative, some resources should be terminated
				deltaReplicas := currentNumberReplicas - profileCurrentLimits.MSCSetting.Replicas
				vmSet = p.removeVMs(p.currentState.VMs, deltaReplicas, currentContainerLimits)
			}
			newNumServiceReplicas = profileCurrentLimits.MSCSetting.Replicas
			resourceLimits  = profileCurrentLimits.Limits
			stateLoadCapacity = profileCurrentLimits.MSCSetting.MSCPerSecond
			totalServicesBootingTime = profileCurrentLimits.MSCSetting.BootTimeSec
		}

		services :=  make(map[string]types.ServiceInfo)
		services[ p.sysConfiguration.MainServiceName] = types.ServiceInfo{
			Scale:  newNumServiceReplicas,
			CPU:    resourceLimits.CPUCores,
			Memory: resourceLimits.MemoryGB,
		}
		state := types.State{}
		state.Services = services
		cleanKeys(vmSet)
		state.VMs = vmSet
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		setScalingSteps(&configurations,p.currentState,state,timeStart,timeEnd, totalServicesBootingTime, stateLoadCapacity)
		p.currentState = state
	}

	//Add new policy
	parameters := make(map[string]string)
	parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
	parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(true)
	parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)
	parameters[types.ISRESIZEPODS] = strconv.FormatBool(false)
	numConfigurations := len(configurations)
	newPolicy.ScalingActions = configurations
	newPolicy.Algorithm = p.algorithm
	newPolicy.ID = bson.NewObjectId()
	newPolicy.Status = types.DISCARTED	//State by default
	newPolicy.Parameters = parameters
	newPolicy.Metrics.NumberScalingActions = numConfigurations
	newPolicy.Metrics.FinishTimeDerivation = time.Now()
	newPolicy.Metrics.DerivationDuration = newPolicy.Metrics.FinishTimeDerivation.Sub(newPolicy.Metrics.StartTimeDerivation).Seconds()
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
	under provisioning is not allowed.
	in:
		@currentVMSet
		@numberReplicas
		@limits
	out:
		@VMScale
*/
func (p DeltaLoadPolicy)removeVMs(currentVMSet types.VMScale, numberReplicas int, limits types.Limit) types.VMScale {
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
func (p DeltaLoadPolicy) selectContainersConfig(currentLimits types.Limit, profileCurrentLimits types.MSCSimpleSetting,
	newLimits types.Limit, profileNewLimits types.MSCSimpleSetting, containerResize bool) (MSCProfile, error) {

	currentNumberReplicas := float64(profileCurrentLimits.Replicas)
	utilizationCurrent := (currentNumberReplicas * currentLimits.CPUCores)+(currentNumberReplicas * currentLimits.MemoryGB)

	newNumberReplicas := float64(profileNewLimits.Replicas)
	utilizationNew := (newNumberReplicas * newLimits.CPUCores)+(newNumberReplicas * newLimits.MemoryGB)

	if utilizationNew < utilizationCurrent && containerResize {
		return MSCProfile{ResourceLimits:newLimits,
			NumberReplicas:int(newNumberReplicas),
			MSC:profileNewLimits.MSCPerSecond,}, nil
	} else {
		return MSCProfile{ResourceLimits:currentLimits,
			NumberReplicas:int(currentNumberReplicas),
			MSC:profileCurrentLimits.MSCPerSecond,}, nil
	}
}