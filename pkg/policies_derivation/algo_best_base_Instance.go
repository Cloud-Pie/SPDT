package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/config"
	"strconv"
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
	policies := []types.Policy{}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	containerResizeEnabled := p.sysConfiguration.PolicySettings.PodsResizeAllowed

	//Loops all the VM types and derive a policy using a single VMType
	for vmType, vm := range p.mapVMProfiles {
		vmTypeSuitable := true

		newPolicy := types.Policy{}
		newPolicy.Metrics = types.PolicyMetrics {
			StartTimeDerivation:time.Now(),
		}
		configurations := []types.ScalingConfiguration{}
		for _, it := range processedForecast.CriticalIntervals {
			serviceToScale := p.currentState.Services[p.sysConfiguration.ServiceName]
			currentContainerLimits := types.Limit{ MemoryGB:serviceToScale.Memory, CPUCores:serviceToScale.CPU }
			ProfileCurrentLimits := selectProfileWithLimits(it.Requests, currentContainerLimits, false)
			vmLimits := types.Limit{ MemoryGB:vm.Memory, CPUCores:vm.CPUCores}
			ProfileNewLimits := selectProfile(it.Requests, vmLimits, false)

			containersConfig,err := p.selectContainersConfig(ProfileCurrentLimits.Limits, ProfileCurrentLimits.TRNConfiguration,
																	ProfileNewLimits.Limits, ProfileNewLimits.TRNConfiguration, containerResizeEnabled, vmType)
			if err !=  nil {
				vmTypeSuitable = false
			}
			newNumServiceReplicas := containersConfig.TRNConfiguration.NumberReplicas
			stateLoadCapacity := containersConfig.TRNConfiguration.TRN
			totalServicesBootingTime := containersConfig.TRNConfiguration.BootTimeSec
			vmSet := containersConfig.VMSet
			limits := containersConfig.Limits

			if underProvisionAllowed {
				ProfileSameLimits := selectProfileWithLimits(it.Requests, currentContainerLimits, underProvisionAllowed)
				ProfileNewLimits := selectProfile(it.Requests, vmLimits, underProvisionAllowed)
				underContainersConfig,err := p.selectContainersConfig(ProfileSameLimits.Limits,ProfileSameLimits.TRNConfiguration,
					ProfileNewLimits.Limits, ProfileNewLimits.TRNConfiguration, containerResizeEnabled, vmType)
				if err !=  nil {
					vmTypeSuitable = false
					break // No VMset fits for the containers set
				} else {
					if underContainersConfig.Cost < containersConfig.Cost {
						newNumServiceReplicas = underContainersConfig.TRNConfiguration.NumberReplicas
						stateLoadCapacity = underContainersConfig.TRNConfiguration.TRN
						totalServicesBootingTime = underContainersConfig.TRNConfiguration.BootTimeSec
						vmSet = underContainersConfig.VMSet
						limits = underContainersConfig.Limits
						vmTypeSuitable = true
					}
				}
			}

			services :=  make(map[string]types.ServiceInfo)
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

		if !vmTypeSuitable {
			break		//Try with other VM type
		}

		numConfigurations := len(configurations)
		if numConfigurations > 0 {
			//Add new policy
			parameters := make(map[string]string)
			parameters[types.VMTYPES] = vmType
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
			newPolicy.Metrics.DerivationDuration = newPolicy.Metrics.FinishTimeDerivation.Sub(newPolicy.Metrics.StartTimeDerivation).Seconds()
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
	newLimits types.Limit, profileNewLimits types.TRNConfiguration, containerResize bool, vmType string) (types.ContainersConfig, error) {

	vmSet1 := p.FindSuitableVMs(profileCurrentLimits.NumberReplicas, currentLimits, vmType)
	costCurrent := vmSet1.Cost(p.mapVMProfiles)
	vmSet2 := p.FindSuitableVMs(profileNewLimits.NumberReplicas, newLimits, vmType)
	costNew := vmSet2.Cost(p.mapVMProfiles)

	performanceFactorCurrent :=  (currentLimits.MemoryGB * float64(profileCurrentLimits.NumberReplicas) + currentLimits.CPUCores* float64(profileCurrentLimits.NumberReplicas))
	performanceFactorNew := (newLimits.MemoryGB * float64(profileNewLimits.NumberReplicas) + newLimits.CPUCores* float64(profileNewLimits.NumberReplicas))

	if  performanceFactorNew < performanceFactorCurrent && containerResize && len(vmSet2) != 0 {
		return types.ContainersConfig{Limits: newLimits,
			TRNConfiguration: profileNewLimits,
			VMSet: vmSet2,
			Cost: costNew,
		}, nil

	} else {
		return types.ContainersConfig{Limits: currentLimits,
			TRNConfiguration: profileCurrentLimits,
			VMSet: vmSet1,
			Cost: costCurrent,
		}, nil
	}
}