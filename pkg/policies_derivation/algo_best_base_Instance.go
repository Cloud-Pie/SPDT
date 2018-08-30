package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/config"
	"strconv"
	"errors"
	"github.com/Cloud-Pie/SPDT/util"
)

/*
After each change in the workload it calculates the number of VMs of a predefined size needed
Repeat the process for all the vm types available
*/
type BestBaseInstancePolicy struct {
	algorithm  string               //Algorithm's name
	timeWindow TimeWindowDerivation //Algorithm used to process the forecasted time serie
	currentState	types.State			 //Current State
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
	policies := []types.Policy{}

	//Loops all the VM types and derive a policy using a single VMType
	for vmType, _ := range p.mapVMProfiles {
		vmTypeSuitable := true

		newPolicy := types.Policy{}
		newPolicy.Metrics = types.PolicyMetrics {
			StartTimeDerivation:time.Now(),
		}
		configurations := []types.Configuration{}
		underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
		containerResizeEnabled := p.sysConfiguration.PolicySettings.PodsResizeAllowed
		serviceToScale := p.currentState.Services[p.sysConfiguration.ServiceName]
		currentContainerLimits := types.Limit{ MemoryGB:serviceToScale.Memory, NumberCores:serviceToScale.CPU }

		for _, it := range processedForecast.CriticalIntervals {
			ProfileCurrentLimits := selectProfileWithLimits(it.Requests, currentContainerLimits, false)
			ProfileNewLimits := selectProfile(it.Requests, false)

			containersConfig,err := p.selectContainersConfig(ProfileCurrentLimits.Limit, ProfileCurrentLimits.TRNConfiguration[0],
																	ProfileNewLimits.Limit, ProfileNewLimits.TRNConfiguration[0], containerResizeEnabled, vmType)
			if err !=  nil {
				vmTypeSuitable = false
			}
			newNumServiceReplicas := containersConfig.PerformanceProfile.NumberReplicas
			stateLoadCapacity := containersConfig.PerformanceProfile.TRN
			totalServicesBootingTime := containersConfig.PerformanceProfile.BootTimeSec
			vmSet := containersConfig.VMSet
			limits := containersConfig.ResourceLimits

			if underProvisionAllowed {
				ProfileSameLimits := selectProfileWithLimits(it.Requests, currentContainerLimits, underProvisionAllowed)
				ProfileNewLimits := selectProfile(it.Requests, underProvisionAllowed)
				underContainersConfig,err := p.selectContainersConfig(ProfileSameLimits.Limit,ProfileSameLimits.TRNConfiguration[0],
					ProfileNewLimits.Limit, ProfileNewLimits.TRNConfiguration[0], containerResizeEnabled, vmType)
				if err !=  nil {
					vmTypeSuitable = false
					break // No VMset fits for the containers set
				} else {
					if underContainersConfig.Cost > containersConfig.Cost {
						newNumServiceReplicas = underContainersConfig.PerformanceProfile.NumberReplicas
						stateLoadCapacity = underContainersConfig.PerformanceProfile.TRN
						totalServicesBootingTime = underContainersConfig.PerformanceProfile.BootTimeSec
						vmSet = underContainersConfig.VMSet
						limits = underContainersConfig.ResourceLimits
						vmTypeSuitable = true
					}
				}
			}

			services :=  make(map[string]types.ServiceInfo)
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

		if !vmTypeSuitable {
			break		//Try with other VM type
		}

		numConfigurations := len(configurations)
		if numConfigurations > 0 {
			//Add new policy
			parameters := make(map[string]string)
			parameters[types.METHOD] = util.SCALE_METHOD_HORIZONTAL
			parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(false)
			parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)
			parameters[types.ISRESIZEPODS] = strconv.FormatBool(containerResizeEnabled)
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
		}
	}
	return policies
}

/*Calculate VM set able to host the required number of replicas
 in:
	@numberReplicas = Amount of replicas that should be hosted
	@resourcesLimit = Resources (CPU, Memory) constraints to configure the containers.
 out:
	@VMScale with the suggested number of VMs for that type
*/
func (p BestBaseInstancePolicy) FindSuitableVMs(numberReplicas int, resourcesLimit types.Limit, vmType string) types.VMScale {
	vmScale := make(map[string]int)
	profile := p.mapVMProfiles[vmType]
	maxReplicas := maxReplicasCapacityInVM(profile, resourcesLimit)
	if maxReplicas > 0 {
		numVMs := math.Ceil(float64(numberReplicas) / float64(maxReplicas))
		vmScale[vmType] = int(numVMs)
	}
	return vmScale
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
func (p BestBaseInstancePolicy) selectContainersConfig(currentLimits types.Limit, profileCurrentLimits types.TRNConfiguration,
	newLimits types.Limit, profileNewLimits types.TRNConfiguration, containerResize bool, vmType string) (ContainersConfig, error) {

	vmSet1 := p.FindSuitableVMs(profileCurrentLimits.NumberReplicas, currentLimits, vmType)
	costCurrent := vmSet1.Cost(p.mapVMProfiles)
	vmSet2 := p.FindSuitableVMs(profileNewLimits.NumberReplicas, newLimits, vmType)
	costNew := vmSet2.Cost(p.mapVMProfiles)
	//TODO:Review logic.
	if len(vmSet1) != 0 && len(vmSet2) != 0 {
		if  containerResize {
			return ContainersConfig{ResourceLimits: newLimits,
				PerformanceProfile: profileNewLimits,
				VMSet: vmSet2,
				Cost: costNew,
			}, nil
		} else {
			return ContainersConfig{ResourceLimits: currentLimits,
				PerformanceProfile: profileCurrentLimits,
				VMSet: vmSet1,
				Cost: costCurrent,
			}, nil
		}
	} else if len(vmSet1) != 0 {
		return ContainersConfig{ResourceLimits: currentLimits,
			PerformanceProfile: profileCurrentLimits,
			VMSet: vmSet1,
			Cost: costCurrent,
		}, nil
	} else if len(vmSet2) != 0 {
		return ContainersConfig{ResourceLimits: newLimits,
			PerformanceProfile: profileNewLimits,
			VMSet: vmSet2,
			Cost: costNew,
		}, nil
	} else {
		return ContainersConfig{}, errors.New("Container limits don't fit in this type of VM")
	}
}