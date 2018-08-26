package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/config"
	"strconv"
	"sort"
	"errors"
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
func (p StepRepackPolicy) CreatePolicies(processedForecast types.ProcessedForecast, serviceProfile types.ServiceProfile) [] types.Policy {
	policies := []types.Policy{}
	//Compute results for cluster of each type
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}

	configurations := []types.Configuration{}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	containerResizeEnabled := true
	currentContainerLimits := p.currentContainerLimits()

	for _, it := range processedForecast.CriticalIntervals {

		ProfileSameLimits := selectProfileWithLimits(serviceProfile.PerformanceProfiles, it.Requests, currentContainerLimits)
		ProfileNewLimits := selectProfile(serviceProfile.PerformanceProfiles, it.Requests, underProvisionAllowed)

		containersConfig,_ := p.selectContainersConfig(ProfileSameLimits.Limit, ProfileSameLimits.TRNConfiguration[0],
			ProfileNewLimits.Limit, ProfileNewLimits.TRNConfiguration[0], containerResizeEnabled)
		//TODO: check for case -> vm set dont fit and not underprovision
		newNumServiceReplicas := containersConfig.PerformanceProfile.NumberReplicas
		stateLoadCapacity := containersConfig.PerformanceProfile.TRN
		totalServicesBootingTime := containersConfig.PerformanceProfile.BootTimeSec
		vmSet := containersConfig.VMSet
		limits := containersConfig.ResourceLimits

		if underProvisionAllowed {
			underContainersConfig,_ := p.selectContainersConfig(ProfileSameLimits.Limit,
				ProfileSameLimits.TRNConfiguration[1], ProfileNewLimits.Limit,
				ProfileNewLimits.TRNConfiguration[1], containerResizeEnabled)

				if underContainersConfig.Cost > containersConfig.Cost {
					newNumServiceReplicas = underContainersConfig.PerformanceProfile.NumberReplicas
					stateLoadCapacity = underContainersConfig.PerformanceProfile.TRN
					totalServicesBootingTime = underContainersConfig.PerformanceProfile.BootTimeSec
					vmSet = underContainersConfig.VMSet
					limits = underContainersConfig.ResourceLimits
				}

		}

		services := make(map[string]types.ServiceInfo)
		services[serviceProfile.Name] = types.ServiceInfo{
			Scale:  newNumServiceReplicas,
			CPU:    limits.NumberCores,
			Memory: limits.MemoryGB,
		}

		state := types.State{}
			state.Services = services
			state.VMs = vmSet

			timeStart := it.TimeStart
			timeEnd := it.TimeEnd
			setConfiguration(&configurations,state,timeStart,timeEnd,serviceProfile.Name, totalServicesBootingTime, p.sysConfiguration, stateLoadCapacity)
	}

		//Add new policy
		parameters := make(map[string]string)
		parameters[types.METHOD] = "hybrid"
		parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(false)
		parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)

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
	@resourcesLimit = Resources (CPU, Memory) constraints to configure the containers.
 out:
	@VMScale with the suggested number of VMs for that type
*/
func (p StepRepackPolicy) FindSuitableVMs(numberReplicas int, resourcesLimit types.Limit) types.VMScale {
	vmScaleList := []types.VMScale{}
		for vmType,v := range p.mapVMProfiles {
			maxReplicas := maxReplicasCapacityInVM(v, resourcesLimit)
			vmScale :=  make(map[string]int)
			if maxReplicas > 0 {
				numVMs := math.Ceil(float64(numberReplicas) / float64(maxReplicas))
				vmScale :=  make(map[string]int)
				vmScale[vmType] = int(numVMs)
			}
			vmScaleList = append(vmScaleList, vmScale)
		}

	sort.Slice(vmScaleList, func(i, j int) bool {
		return vmScaleList[i].Cost(p.mapVMProfiles) < vmScaleList[j].Cost(p.mapVMProfiles)
	})

	return vmScaleList[0]
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
	newLimits types.Limit, profileNewLimits types.TRNConfiguration, containerResize bool) (ContainersConfig, error) {

	vmSet1 := p.FindSuitableVMs(profileCurrentLimits.NumberReplicas, currentLimits)
	costCurrent := vmSet1.Cost(p.mapVMProfiles)
	vmSet2 := p.FindSuitableVMs(profileNewLimits.NumberReplicas, newLimits)
	costNew := vmSet2.Cost(p.mapVMProfiles)

	if len(vmSet1) == 0 && len(vmSet2)== 0 {
		return ContainersConfig{}, errors.New("Containers ")
	}
	if costNew < costCurrent && containerResize {
		return ContainersConfig{ResourceLimits:newLimits,
			PerformanceProfile:	profileNewLimits,
			VMSet:vmSet2,
			Cost:costNew,
		}, nil
	} else {
		return ContainersConfig{ResourceLimits:currentLimits,
			PerformanceProfile:	profileCurrentLimits,
			VMSet:vmSet1,
			Cost:costCurrent,
		}, nil
	}
}

/*Return the Limit constraint of the current configuration
	out:
		Limit
*/
func (p StepRepackPolicy) currentContainerLimits() types.Limit {
	var limits types.Limit
	for _,s := range p.currentState.Services {
		limits.MemoryGB = s.Memory
		limits.NumberCores = s.CPU
	}
	return limits
}