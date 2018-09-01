package policies_derivation

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
	configurations := []types.ScalingConfiguration{}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	containerResizeEnabled := p.sysConfiguration.PolicySettings.PodsResizeAllowed

	for _, it := range processedForecast.CriticalIntervals {
		serviceToScale := p.currentState.Services[p.sysConfiguration.ServiceName]
		currentContainerLimits := types.Limit{ MemoryGB:serviceToScale.Memory, NumberCores:serviceToScale.CPU }
		ProfileCurrentLimits := selectProfileWithLimits(it.Requests, currentContainerLimits, false)
		ProfileNewLimits := selectProfile(it.Requests, false)

		containersConfig,_ := p.selectContainersConfig(ProfileCurrentLimits.Limits, ProfileCurrentLimits.PerformanceProfile,
			ProfileNewLimits.Limit, ProfileNewLimits.TRNConfiguration[0], containerResizeEnabled)

		newNumServiceReplicas := containersConfig.PerformanceProfile.NumberReplicas
		stateLoadCapacity := containersConfig.PerformanceProfile.TRN
		totalServicesBootingTime := containersConfig.PerformanceProfile.BootTimeSec
		vmSet := containersConfig.VMSet
		limits := containersConfig.Limits

		if underProvisionAllowed {
			ProfileCurrentLimits := selectProfileWithLimits(it.Requests, currentContainerLimits, underProvisionAllowed)
			ProfileNewLimits := selectProfile(it.Requests, underProvisionAllowed)
			underContainersConfig,_ := p.selectContainersConfig(ProfileCurrentLimits.Limits,
				ProfileCurrentLimits.PerformanceProfile, ProfileNewLimits.Limit,
				ProfileNewLimits.TRNConfiguration[0], containerResizeEnabled)

				if underContainersConfig.Cost > containersConfig.Cost {
					newNumServiceReplicas = underContainersConfig.PerformanceProfile.NumberReplicas
					stateLoadCapacity = underContainersConfig.PerformanceProfile.TRN
					totalServicesBootingTime = underContainersConfig.PerformanceProfile.BootTimeSec
					vmSet = underContainersConfig.VMSet
					limits = underContainersConfig.Limits
				}
		}

		services := make(map[string]types.ServiceInfo)
		services[ p.sysConfiguration.ServiceName] = types.ServiceInfo {
			Scale:  newNumServiceReplicas,
			CPU:    limits.NumberCores,
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
func (p StepRepackPolicy) FindSuitableVMs(numberReplicas int, limits types.Limit) types.VMScale {
	heterogeneousAllowed := p.sysConfiguration.PolicySettings.HetereogeneousAllowed
	vmSet, _ := buildHomogeneousVMSet(numberReplicas,limits, p.mapVMProfiles)

	if heterogeneousAllowed {
		hetVMSet,_ := buildHeterogeneousVMSet(numberReplicas, limits, p.mapVMProfiles)
		costi := hetVMSet.Cost(p.mapVMProfiles)
		costj := vmSet.Cost(p.mapVMProfiles)
		if costi < costj {
			vmSet = hetVMSet
		}
	}
	return vmSet
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
func (p StepRepackPolicy) selectContainersConfig(currentLimits types.Limit, profileCurrentLimits types.TRNConfiguration,
	newLimits types.Limit, profileNewLimits types.TRNConfiguration, containerResize bool) (types.ContainersConfig, error) {

	vmSet1 := p.FindSuitableVMs(profileCurrentLimits.NumberReplicas, currentLimits)
	costCurrent := vmSet1.Cost(p.mapVMProfiles)
	vmSet2 := p.FindSuitableVMs(profileNewLimits.NumberReplicas, newLimits)
	costNew := vmSet2.Cost(p.mapVMProfiles)

	if len(vmSet1) == 0 && len(vmSet2)== 0 {
		return types.ContainersConfig{}, errors.New("Containers ")
	}
	//TODO:Review logic
	if containerResize {
		return types.ContainersConfig{Limits:newLimits,
			PerformanceProfile:	profileNewLimits,
			VMSet:vmSet2,
			Cost:costNew,
		}, nil
	} else {
		return types.ContainersConfig{Limits:currentLimits,
			PerformanceProfile:	profileCurrentLimits,
			VMSet:vmSet1,
			Cost:costCurrent,
		}, nil
	}
}
