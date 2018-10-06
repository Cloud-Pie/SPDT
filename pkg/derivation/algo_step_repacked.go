package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/config"
	"strconv"
	"github.com/Cloud-Pie/SPDT/util"
)

type StepRepackPolicy struct {
	algorithm 		string
	timeWindow 		TimeWindowDerivation
	sortedVMProfiles []types.VmProfile
	mapVMProfiles map[string]types.VmProfile
	sysConfiguration	config.SystemConfiguration
	currentState	types.State			 //Current State
}


/* Derive a list of policies using the best homogeneous cluster, change of type is possible
	in:
		@processedForecast
		@serviceProfile
	out:
		[] Policy. List of type Policy
*/
func (p StepRepackPolicy) CreatePolicies(processedForecast types.ProcessedForecast) [] types.Policy {
	log.Info("Derive policies with %s algorithm", p.algorithm)
	policies := []types.Policy{}
	//Compute results for cluster of each type

	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	containerResizeEnabled := p.sysConfiguration.PolicySettings.PodsResizeAllowed
	percentageUnderProvision := p.sysConfiguration.PolicySettings.MaxUnderprovisionPercentage
	biggestVM := p.sortedVMProfiles[len(p.sortedVMProfiles)-1]
	vmLimits := types.Limit{ MemoryGB:biggestVM.Memory, CPUCores:biggestVM.CPUCores}

	serviceToScale := p.currentState.Services[p.sysConfiguration.MainServiceName]
	currentContainerLimits := types.Limit{ MemoryGB:serviceToScale.Memory, CPUCores:serviceToScale.CPU }
	newPolicy := p.deriveCandidatePolicy(processedForecast.CriticalIntervals,currentContainerLimits, vmLimits, containerResizeEnabled, underProvisionAllowed, percentageUnderProvision )
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
func (p StepRepackPolicy) FindSuitableVMs(numberReplicas int, limits types.Limit) (types.VMScale,error) {
	vmSet,err := buildHomogeneousVMSet(numberReplicas,limits, p.mapVMProfiles)
	/*hetVMSet,_ := buildHeterogeneousVMSet(numberReplicas, limits, p.mapVMProfiles)
	costi := hetVMSet.Cost(p.mapVMProfiles)
	costj := vmSet.Cost(p.mapVMProfiles)
	if costi < costj {
		vmSet = hetVMSet
	}

	if err!= nil {
		return vmSet,errors.New("No suitable VM set found")
	}*/
	return vmSet,err
}


func (p StepRepackPolicy)deriveCandidatePolicy(criticalIntervals []types.CriticalInterval, containerLimits types.Limit,
	vmLimits types.Limit, containerResizeEnabled bool,underProvisionAllowed bool,percentageUnderProvision float64) types.Policy {

	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics{
		StartTimeDerivation: time.Now(),
	}
	scalingSteps := []types.ScalingStep{}

	for _, it := range criticalIntervals {
		profileCurrentLimits := selectProfileByLimits(it.Requests, containerLimits, false)
		var vmSet types.VMScale
		if containerResizeEnabled {
				ProfileNewLimits, _ := selectProfileUnderVMLimits(it.Requests, vmLimits, false)
				resize := shouldResizeContainer(profileCurrentLimits, ProfileNewLimits)
				if resize {
					profileCurrentLimits = ProfileNewLimits
				}
				vmSet, _ = p.FindSuitableVMs(profileCurrentLimits.MSCSetting.Replicas, profileCurrentLimits.Limits)
		} else {
			vmSet, _ = p.FindSuitableVMs(profileCurrentLimits.MSCSetting.Replicas, profileCurrentLimits.Limits)
		}

		newNumServiceReplicas := profileCurrentLimits.MSCSetting.Replicas
		stateLoadCapacity := profileCurrentLimits.MSCSetting.MSCPerSecond
		totalServicesBootingTime := profileCurrentLimits.MSCSetting.BootTimeSec
		limits := profileCurrentLimits.Limits

		services := make(map[string]types.ServiceInfo)
		services[ p.sysConfiguration.MainServiceName] = types.ServiceInfo{
			Scale:  newNumServiceReplicas,
			CPU:    limits.CPUCores,
			Memory: limits.MemoryGB,
		}

		state := types.State{}
		state.Services = services
		state.VMs = vmSet

		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		setScalingSteps(&scalingSteps, p.currentState, state, timeStart, timeEnd, totalServicesBootingTime, stateLoadCapacity)
		p.currentState = state
	}

	//Add new policy
	parameters := make(map[string]string)
	parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
	parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(true)
	parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)
	parameters[types.ISRESIZEPODS] = strconv.FormatBool(containerResizeEnabled)
	numConfigurations := len(scalingSteps)
	newPolicy.ScalingActions = scalingSteps
	newPolicy.Algorithm = p.algorithm
	newPolicy.ID = bson.NewObjectId()
	newPolicy.Status = types.DISCARTED //State by default
	newPolicy.Parameters = parameters
	newPolicy.Metrics.NumberScalingActions = numConfigurations
	newPolicy.Metrics.FinishTimeDerivation = time.Now()
	newPolicy.TimeWindowStart = scalingSteps[0].TimeStart
	newPolicy.TimeWindowEnd = scalingSteps[numConfigurations-1].TimeEnd
	newPolicy.Metrics.DerivationDuration = newPolicy.Metrics.FinishTimeDerivation.Sub(newPolicy.Metrics.StartTimeDerivation).Seconds()

	return newPolicy
}