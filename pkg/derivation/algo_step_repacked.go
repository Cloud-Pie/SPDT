package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/config"
	"strconv"
	"errors"
	"github.com/Cloud-Pie/SPDT/util"
)

/*
After each change in the workload it calculates the number of VMs in a heterogeneous cluster
*/
type StepRepackPolicy struct {
	algorithm 		string				 //Algorithm's name
	timeWindow 		TimeWindowDerivation //Algorithm used to process the forecasted time serie
	sortedVMProfiles []types.VmProfile    			//List of VM profiles sorted by price
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
	policies := []types.Policy{}
	//Compute results for cluster of each type
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}
	configurations := []types.ScalingAction{}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	containerResizeEnabled := p.sysConfiguration.PolicySettings.PodsResizeAllowed
	percentageUnderProvision := p.sysConfiguration.PolicySettings.MaxUnderprovisionPercentage
	biggestVM := p.sortedVMProfiles[len(p.sortedVMProfiles)-1]
	vmLimits := types.Limit{ MemoryGB:biggestVM.Memory, CPUCores:biggestVM.CPUCores}


	for _, it := range processedForecast.CriticalIntervals {
		serviceToScale := p.currentState.Services[p.sysConfiguration.ServiceName]
		currentContainerLimits := types.Limit{ MemoryGB:serviceToScale.Memory, CPUCores:serviceToScale.CPU }
		ProfileCurrentLimits := selectProfileWithLimits(it.Requests, currentContainerLimits, false)

		if containerResizeEnabled {
			ProfileNewLimits := selectProfile(it.Requests, vmLimits, false)
			resize := shouldResizeContainer(ProfileCurrentLimits, ProfileNewLimits)
			if resize {
				ProfileCurrentLimits = ProfileNewLimits
			}
		}

		vmSet,_ := p.FindSuitableVMs(ProfileCurrentLimits.MSCSetting.Replicas, ProfileCurrentLimits.Limits)

		if underProvisionAllowed {
			ProfileCurrentLimitsUnder := selectProfileWithLimits(it.Requests, currentContainerLimits, underProvisionAllowed)
			ProfileNewLimitsUnder := selectProfile(it.Requests, vmLimits, underProvisionAllowed)
			resize := shouldResizeContainer(ProfileCurrentLimitsUnder, ProfileNewLimitsUnder)
			if resize {
				ProfileCurrentLimitsUnder = ProfileNewLimitsUnder
			}
			vmSetUnder,_ := p.FindSuitableVMs(ProfileCurrentLimits.MSCSetting.Replicas, ProfileCurrentLimits.Limits)

			if isUnderProvisionInRange(it.Requests, ProfileCurrentLimitsUnder.MSCSetting.MSCPerSecond, percentageUnderProvision) &&
				vmSetUnder.Cost(p.mapVMProfiles) < vmSet.Cost(p.mapVMProfiles) {

				vmSet = vmSetUnder
				ProfileCurrentLimits = ProfileCurrentLimitsUnder
			}
		}

		newNumServiceReplicas := ProfileCurrentLimits.MSCSetting.Replicas
		stateLoadCapacity := ProfileCurrentLimits.MSCSetting.MSCPerSecond
		totalServicesBootingTime := ProfileCurrentLimits.MSCSetting.BootTimeSec
		limits := ProfileCurrentLimits.Limits

		services := make(map[string]types.ServiceInfo)
		services[ p.sysConfiguration.ServiceName] = types.ServiceInfo {
			Scale:  newNumServiceReplicas,
			CPU:    limits.CPUCores,
			Memory: limits.MemoryGB,
		}

		state := types.State{}
		state.Services = services
		state.VMs = vmSet

		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		setConfiguration(&configurations,state,timeStart,timeEnd, p.sysConfiguration.ServiceName, totalServicesBootingTime, p.sysConfiguration, stateLoadCapacity)
		p.currentState = state
	}

		//Add new policy
		parameters := make(map[string]string)
		parameters[types.METHOD] =  util.SCALE_METHOD_HORIZONTAL
		parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(false)
		parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)
		parameters[types.ISRESIZEPODS] = strconv.FormatBool(containerResizeEnabled)
		numConfigurations := len(configurations)
		newPolicy.ScalingActions = configurations
		newPolicy.Algorithm = p.algorithm
		newPolicy.ID = bson.NewObjectId()
		newPolicy.Status = types.DISCARTED	//State by default
		newPolicy.Parameters = parameters
		newPolicy.Metrics.NumberScalingActions = numConfigurations
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
func (p StepRepackPolicy) FindSuitableVMs(numberReplicas int, limits types.Limit) (types.VMScale,error) {
	heterogeneousAllowed := p.sysConfiguration.PolicySettings.HetereogeneousAllowed
	vmSet,err := buildHomogeneousVMSet(numberReplicas,limits, p.mapVMProfiles)

	if heterogeneousAllowed {
		hetVMSet,_ := buildHeterogeneousVMSet(numberReplicas, limits, p.mapVMProfiles)
		costi := hetVMSet.Cost(p.mapVMProfiles)
		costj := vmSet.Cost(p.mapVMProfiles)
		if costi < costj {
			vmSet = hetVMSet
		}
	}
	if err!= nil {
		return vmSet,errors.New("No suitable VM set found")
	}
	return vmSet,err
}
