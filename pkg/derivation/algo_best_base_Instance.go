package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/config"
	"strconv"
	"github.com/Cloud-Pie/SPDT/util"
	"errors"
	"github.com/Cloud-Pie/SPDT/storage"
	"sort"
)

/*
After each change in the workload it calculates the number of VMs of a predefined size needed
Repeat the process for all the vm types available
*/
type BestBaseInstancePolicy struct {
	algorithm  string               //Algorithm's name
	timeWindow TimeWindowDerivation //Algorithm used to process the forecasted time serie
	currentState	types.State			 //Current State
	sortedVMProfiles []types.VmProfile    			//List of VM profiles sorted by price
	mapVMProfiles   map[string]types.VmProfile
	sysConfiguration	config.SystemConfiguration
}

/* Derive a list of policies using the Best Instance Approach approach
	in:
		@processedForecast
		@serviceProfile
	out:
		[] Policy. List of type Policy
*/
func (p BestBaseInstancePolicy) CreatePolicies(processedForecast types.ProcessedForecast) [] types.Policy {
	log.Info("Derive policies with %s algorithm", p.algorithm)
	policies := []types.Policy{}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	containerResizeEnabled := p.sysConfiguration.PolicySettings.PodsResizeAllowed
	percentageUnderProvision := p.sysConfiguration.PolicySettings.MaxUnderprovisionPercentage
	startAlgo := time.Now()
	//Loops all the VM types and derive a policy using a single VMType
	selectedResourceLimits := p.selectServiceProfilesLimits(processedForecast.CriticalIntervals, )

	for vmType, vm := range p.mapVMProfiles {
		vmLimits := types.Limit{ MemoryGB:vm.Memory, CPUCores:vm.CPUCores}
		//Container limits that fit into the VM type
		for _, li := range selectedResourceLimits {
			vmTypeSuitable, newPolicy := p.deriveCandidatePolicy(processedForecast.CriticalIntervals,containerResizeEnabled, li, vmLimits, vmType, underProvisionAllowed, percentageUnderProvision )
			if vmTypeSuitable {
				policies = append(policies, newPolicy)
			}
		}
	}

	for i := range policies {
		cost := ComputePolicyCost((policies)[i],systemConfiguration.PricingModel.BillingUnit, p.mapVMProfiles)
		(policies)[i].Metrics.Cost = math.Ceil(cost*100)/100
	}
	//Sort policies based on price
	sort.Slice(policies, func(i, j int) bool {
		order := (policies)[i].Metrics.Cost < (policies)[j].Metrics.Cost
		return order
	})
	//return policies
	timeEndAlgo := time.Now()
	selectedOption := policies[0]
	selectedOption.Metrics.StartTimeDerivation = startAlgo
	selectedOption.Metrics.FinishTimeDerivation = timeEndAlgo
	selectedOption.Metrics.DerivationDuration = selectedOption.Metrics.FinishTimeDerivation.Sub(selectedOption.Metrics.StartTimeDerivation).Seconds()
	return []types.Policy{selectedOption}
}

/*Calculate VM set able to host the required number of replicas
 in:
	@numberReplicas = Amount of replicas that should be hosted
	@resourcesLimit = Resources (CPU, Memory) constraints to configure the containers.
 out:
	@VMScale with the suggested number of VMs for that type
*/
func (p BestBaseInstancePolicy) FindSuitableVMs(numberReplicas int, resourcesLimit types.Limit, vmType string) (types.VMScale, error) {
	vmScale := make(map[string]int)
	var err error
	profile := p.mapVMProfiles[vmType]
	maxReplicas := maxReplicasCapacityInVM(profile, resourcesLimit)
	if maxReplicas > 0 {
		numVMs := math.Ceil(float64(numberReplicas) / float64(maxReplicas))
		vmScale[vmType] = int(numVMs)
	} else {
		return vmScale,errors.New("No suitable VM set found")
	}
	return vmScale,err
}


func (p BestBaseInstancePolicy) deriveCandidatePolicy(criticalIntervals []types.CriticalInterval, containerResizeEnabled bool,
	containerLimits types.Limit, vmLimits types.Limit, vmType string, underProvisionAllowed bool,percentageUnderProvision float64 ) (bool, types.Policy) {

	vmTypeSuitable := true
	scalingSteps := []types.ScalingStep{}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}

	for _, it := range criticalIntervals {
		servicePerformanceProfile := selectProfileByLimits(it.Requests, containerLimits, false)
		vmSet,err := p.FindSuitableVMs(servicePerformanceProfile.MSCSetting.Replicas, servicePerformanceProfile.Limits, vmType)
		if err !=  nil {
			vmTypeSuitable = false
		}

		costVMSetOverProvision := vmSet.Cost(p.mapVMProfiles)
		//Update values if the configuration that leads to under provisioning is cheaper
		if underProvisionAllowed {
			containerConfigUnder := p.optionWithUnderProvision(it.Requests, containerLimits, percentageUnderProvision, vmType)
			if containerConfigUnder.Cost >0 && containerConfigUnder.Cost < costVMSetOverProvision &&
				isUnderProvisionInRange(it.Requests, containerConfigUnder.MSCSetting.MSCPerSecond, percentageUnderProvision) {
				vmSet = containerConfigUnder.VMSet
				servicePerformanceProfile = containerConfigUnder
			}
		}

		newNumServiceReplicas := servicePerformanceProfile.MSCSetting.Replicas
		stateLoadCapacity := servicePerformanceProfile.MSCSetting.MSCPerSecond
		totalServicesBootingTime := servicePerformanceProfile.MSCSetting.BootTimeSec
		limits := servicePerformanceProfile.Limits

		services :=  make(map[string]types.ServiceInfo)
		services[ p.sysConfiguration.MainServiceName] = types.ServiceInfo {
			Scale:  newNumServiceReplicas,
			CPU:    limits.CPUCores,
			Memory: limits.MemoryGB,
		}

		state := types.State{}
		state.Services = services
		state.VMs = vmSet
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		setScalingSteps(&scalingSteps,p.currentState, state,timeStart,timeEnd, totalServicesBootingTime, stateLoadCapacity)
		p.currentState = state
	}

	numScalingSteps := len(scalingSteps)
	if vmTypeSuitable && numScalingSteps > 0 {
		//Add new policy
		parameters := make(map[string]string)
		parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
		parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(false)
		parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)
		parameters[types.ISRESIZEPODS] = strconv.FormatBool(containerResizeEnabled)
		newPolicy.ScalingActions = scalingSteps
		newPolicy.Algorithm = p.algorithm
		newPolicy.ID = bson.NewObjectId()
		newPolicy.Status = types.DISCARTED	//State by default
		newPolicy.Parameters = parameters
		newPolicy.Metrics.NumberScalingActions = numScalingSteps
		newPolicy.Metrics.FinishTimeDerivation = time.Now()
		newPolicy.Metrics.DerivationDuration = newPolicy.Metrics.FinishTimeDerivation.Sub(newPolicy.Metrics.StartTimeDerivation).Seconds()
		newPolicy.TimeWindowStart = scalingSteps[0].TimeStart
		newPolicy.TimeWindowEnd = scalingSteps[numScalingSteps-1].TimeEnd
	}
	return vmTypeSuitable, newPolicy
}

/*Search a container configuration (limits & n replicas) which has under provision but leads to a cheaper VM set
	in:
		@totalLoad - float64 = Number of requests
		@containerLimits -Limit = Resource limits for the container
		@percentageUnderProvision -float64 = percentage of the max under provision allowed
	out:
		ContainersConfig -
*/
func (p BestBaseInstancePolicy) optionWithUnderProvision(totalLoad float64, containerLimits types.Limit, percentageUnderProvision float64, vmType string) types.ContainersConfig {
	containerConfigUnder := selectProfileByLimits(totalLoad, containerLimits, true)
	vmSetUnder,_ := p.FindSuitableVMs(containerConfigUnder.MSCSetting.Replicas, containerConfigUnder.Limits, vmType)
	costVMSetUnderProvision := vmSetUnder.Cost(p.mapVMProfiles)
	containerConfigUnder.VMSet = vmSetUnder
	containerConfigUnder.Cost = costVMSetUnderProvision
	return  containerConfigUnder
}

func (p BestBaseInstancePolicy) selectServiceProfilesLimits(forecastedValues []types.CriticalInterval, ) [] types.Limit{
	max := 0.0
	for _,v := range forecastedValues {
		if v.Requests > max {
			max = v.Requests
		}
	}

	selectedLimits := []types.Limit{}
	vmProfiles := p.sortedVMProfiles
	sort.Slice(vmProfiles, func(i, j int) bool {
		return vmProfiles[i].Pricing.Price >  vmProfiles[j].Pricing.Price
	})
	biggestVMType := p.mapVMProfiles[vmProfiles[0].Type]
	allLimits,_ := storage.GetPerformanceProfileDAO(p.sysConfiguration.MainServiceName).FindAllUnderLimits(biggestVMType.CPUCores, biggestVMType.Memory)

	for _,v := range allLimits {
		servicePerformanceProfile := selectProfileByLimits(max, v.Limit, false)
		if servicePerformanceProfile.MSCSetting.MSCPerSecond >= max {
			selectedLimits = append(selectedLimits,v.Limit)
		}
	}
	return selectedLimits
}