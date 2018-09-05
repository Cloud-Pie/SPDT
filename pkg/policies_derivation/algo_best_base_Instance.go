package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/config"
	"strconv"
	"github.com/Cloud-Pie/SPDT/util"
	"errors"
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
	percentageUnderProvision := p.sysConfiguration.PolicySettings.MaxUnderprovisionPercentage
	serviceToScale := p.currentState.Services[p.sysConfiguration.ServiceName]

	//Loops all the VM types and derive a policy using a single VMType
	for vmType, vm := range p.mapVMProfiles {
		vmTypeSuitable := true
		vmLimits := types.Limit{ MemoryGB:vm.Memory, CPUCores:vm.CPUCores}
		newPolicy := types.Policy{}
		newPolicy.Metrics = types.PolicyMetrics {
			StartTimeDerivation:time.Now(),
		}
		configurations := []types.ScalingConfiguration{}
		for _, it := range processedForecast.CriticalIntervals {

			currentContainerLimits := types.Limit{ MemoryGB:serviceToScale.Memory, CPUCores:serviceToScale.CPU }
			ProfileCurrentLimits := selectProfileWithLimits(it.Requests, currentContainerLimits, false)

			if containerResizeEnabled {
				ProfileNewLimits := selectProfile(it.Requests, vmLimits, false)
				resize := shouldResizeContainer(ProfileCurrentLimits, ProfileNewLimits)
				if resize {
					ProfileCurrentLimits = ProfileNewLimits
				}
			}

			vmSet,err := p.FindSuitableVMs(ProfileCurrentLimits.TRNConfiguration.NumberReplicas, ProfileCurrentLimits.Limits, vmType)
			if err !=  nil {
				vmTypeSuitable = false
			}

			if underProvisionAllowed {
				ProfileCurrentLimitsUnder := selectProfileWithLimits(it.Requests, currentContainerLimits, underProvisionAllowed)
				ProfileNewLimitsUnder := selectProfile(it.Requests, vmLimits, underProvisionAllowed)
				resize := shouldResizeContainer(ProfileCurrentLimitsUnder, ProfileNewLimitsUnder)
				if resize {
					ProfileCurrentLimitsUnder = ProfileNewLimitsUnder
				}
				vmSetUnder,err2 := p.FindSuitableVMs(ProfileCurrentLimits.TRNConfiguration.NumberReplicas, ProfileCurrentLimits.Limits, vmType)

				if err2 !=  nil {
					vmTypeSuitable = false
					// No VMset fits for the containers set
					break
				}
				vmTypeSuitable = true
				if isUnderProvisionInRange(it.Requests, ProfileCurrentLimitsUnder.TRNConfiguration.TRN, percentageUnderProvision) &&
					vmSetUnder.Cost(p.mapVMProfiles) < vmSet.Cost(p.mapVMProfiles) {

					vmSet = vmSetUnder
					ProfileCurrentLimits = ProfileCurrentLimitsUnder
				}
			}

			newNumServiceReplicas := ProfileCurrentLimits.TRNConfiguration.NumberReplicas
			stateLoadCapacity := ProfileCurrentLimits.TRNConfiguration.TRN
			totalServicesBootingTime := ProfileCurrentLimits.TRNConfiguration.BootTimeSec
			limits := ProfileCurrentLimits.Limits

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
			//Try with other VM type
			break
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

/*
	in:
		@currentConfiguration types.ContainersConfig
							- Current container configuration
		@newCandidateConfiguration types.ContainersConfig
							- Candidate container configuration with different limits and number of replicas
	out:
		@bool	- Flag to indicate if it is convenient to resize the containers
*/
func shouldResizeContainer(currentConfiguration types.ContainersConfig, newCandidateConfiguration types.ContainersConfig) bool{

	utilizationFactorCurrent :=  currentConfiguration.Limits.MemoryGB * float64(currentConfiguration.TRNConfiguration.NumberReplicas) +
								currentConfiguration.Limits.CPUCores* float64(currentConfiguration.TRNConfiguration.NumberReplicas)

	utilizationFactorNew := newCandidateConfiguration.Limits.MemoryGB * float64(newCandidateConfiguration.TRNConfiguration.NumberReplicas) +
								newCandidateConfiguration.Limits.CPUCores* float64(newCandidateConfiguration.TRNConfiguration.NumberReplicas)

	if utilizationFactorNew < utilizationFactorCurrent {
		return true
	}
	return false
}