package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/config"
	"time"
	"strconv"
	"gopkg.in/mgo.v2/bson"
	"math"
)

/*
It assumes that the current VM set where the microservice is deployed is a homogeneous set
Based on the unique VM type and its capacity to host a number of replicas it increases or decreases the number of VMs
*/
type NaivePolicy struct {
	algorithm  		string               //Algorithm's name
	timeWindow 		TimeWindowDerivation //Algorithm used to process the forecasted time serie
	currentState	types.State			 //Current State
	mapVMProfiles   map[string]types.VmProfile
	sysConfiguration	config.SystemConfiguration

}

/* Derive a list of policies using the Naive approach
	in:
		@processedForecast
		@serviceProfile
	out:
		[] Policy. List of type Policy
*/
func (p NaivePolicy) CreatePolicies(processedForecast types.ProcessedForecast, serviceProfile types.ServiceProfile) []types.Policy {
	policies := []types.Policy {}
	newPolicy := types.Policy{}
	newPolicy.Metrics = types.PolicyMetrics {
		StartTimeDerivation:time.Now(),
	}

	configurations := []types.Configuration {}
	underProvisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
	currentContainerLimits := p.currentContainerLimits()

	for _, it := range processedForecast.CriticalIntervals {
			//Select the performance profile that fits better
			performanceProfile := selectProfileWithLimits(serviceProfile.PerformanceProfiles, it.Requests, currentContainerLimits)

			//Compute the max capacity in terms of number of  service replicas for each VM type
			//computeVMsCapacity(performanceProfile,&p.mapVMProfiles)

			overProvision := performanceProfile.TRNConfiguration[0]
			newNumServiceReplicas := overProvision.NumberReplicas
			vmSet := p.FindSuitableVMs(newNumServiceReplicas, performanceProfile.Limit)
			costOver := vmSet.Cost(p.mapVMProfiles)
			stateLoadCapacity := overProvision.TRN
			totalServicesBootingTime := overProvision.BootTimeSec

			if underProvisionAllowed {
				underProvision := performanceProfile.TRNConfiguration[1]
				vmSetUnder := p.FindSuitableVMs(underProvision.NumberReplicas, performanceProfile.Limit)
				costUnder := vmSetUnder.Cost(p.mapVMProfiles)
				if costUnder < costOver {
					vmSet = vmSetUnder
					newNumServiceReplicas = underProvision.NumberReplicas
					stateLoadCapacity = underProvision.TRN
					totalServicesBootingTime = underProvision.BootTimeSec
				}
			}

			services := make(map[string]types.ServiceInfo)
			services[serviceProfile.Name] = types.ServiceInfo {
				Scale:  newNumServiceReplicas,
				CPU:    performanceProfile.Limit.NumberCores,
				Memory: performanceProfile.Limit.MemoryGB,
			}
			state := types.State{
				Services: services,
				VMs:      vmSet,
			}

			//update state before next iteration
			timeStart := it.TimeStart
			timeEnd := it.TimeEnd
			setConfiguration(&configurations, state, timeStart, timeEnd, serviceProfile.Name, totalServicesBootingTime, p.sysConfiguration, stateLoadCapacity)

			parameters := make(map[string]string)
			parameters[types.METHOD] = "horizontal"
			parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(false)
			parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underProvisionAllowed)

			//Add new policy
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

	}
	return policies
}

/*Calculate VM set able to host the required number of replicas
 in:
	@numberReplicas = Amount of replicas that should be hosted
	@limits = Resources (CPU, Memory) constraints to configure the containers.
 out:
	@VMScale with the suggested number of VMs for that type
*/
func (p NaivePolicy) FindSuitableVMs(numberReplicas int, limits types.Limit) types.VMScale {
	vmScale := make(types.VMScale)
	vmType := p.currentVMType()
	profile := p.mapVMProfiles[vmType]
	maxReplicas := maxReplicasCapacityInVM(profile, limits)
	if maxReplicas > 0 {
		numVMs := math.Ceil(float64(numberReplicas) / float64(maxReplicas))
		vmScale[vmType] = int(numVMs)
	}
	return vmScale
}

/*Return the VM type used by the current Homogeneous VM cluster
	out:
		String with the name of the current VM type
*/
func (p NaivePolicy) currentVMType() string {
	//Assumption for p approach: There is only 1 vm Type in current state
	var vmType string
	for k := range p.currentState.VMs {
		vmType = k
	}
	if len(p.currentState.VMs) > 1 {
		log.Warning("Current config has more than one VM type, type %s was selected to continue", vmType)
	}
	return vmType
}

/*Return the Limit constraint of the current configuration
	out:
		Limit
*/
func (p NaivePolicy) currentContainerLimits() types.Limit {
	var limits types.Limit
	for _,s := range p.currentState.Services {
		limits.MemoryGB = s.Memory
		limits.NumberCores = s.CPU
	}
	return limits
}

//Compares if according to the minimum percentage of underprovisioning is possible to find a cheaper VM set
//by decreasing the number of replicas and comparing the capacity of a VM set for overprovisioning against the new one
//found for underprovision
/*func (p NaivePolicy) considerIfUnderprovision(overVmSet types.VMScale, performanceProfile types.PerformanceProfile, requests float64)(int, types.VMScale){
	var newNumServiceReplicas int

	//Compute number of replicas that leads to minimal underprovision
	underNumServiceReplicas := p.TRNConfiguration.NumberReplicas
	underProvisionTRN := p.TRNConfiguration.TRN
	percentageUnderProvisioned := underProvisionTRN * requests / 100.0
	//Compare if underprovision in terms of number of request is acceptable
	if percentageUnderProvisioned <= p.sysConfiguration.PolicySettings.MaxUnderprovision {
		vmSet := p.FindSuitableVMs(underNumServiceReplicas, performanceProfile.Limit)
		//Compare vm sets for underprovisioning and overprovisioning of service replicas
		overVMSet := VMSet{VMSet:overVmSet}
		overVMSet.setValues(p.mapVMProfiles)
		underVMSet := VMSet{VMSet:vmSet}
		underVMSet.setValues(p.mapVMProfiles)
		//Compare if the change allowing underprovisioning really affect the selected vm set
		if underVMSet.TotalReplicasCapacity < overVMSet.TotalReplicasCapacity {
			newNumServiceReplicas = underNumServiceReplicas
			return newNumServiceReplicas,vmSet
		}
	}
	return newNumServiceReplicas,nil
}*/
/*
func (p NaivePolicy)s (configurations *[]types.Configuration, performanceProfile types.PerformanceProfile, containerSet types.TRNConfiguration, timeStart time.Time, timeEnd time.Time){
	services := make(map[string]types.ServiceInfo)
	newNumServiceReplicas := containerSet.NumberReplicas

	//Compute new  vmset.
	vmSet := p.FindSuitableVMs(newNumServiceReplicas, performanceProfile.Limit)

	stateLoadCapacity := containerSet.TRN
	services[serviceProfile.Name] = types.ServiceInfo{
		Scale:  newNumServiceReplicas,
		CPU:    performanceProfile.Limit.NumberCores,
		Memory: performanceProfile.Limit.MemoryGB,
	}
	state := types.State{
		Services: services,
		VMs:      vmSet,
	}

	totalServicesBootingTime := containerSet.BootTimeSec
	setConfiguration(configurations, state, timeStart, timeEnd, serviceProfile.Name, totalServicesBootingTime, p.sysConfiguration, stateLoadCapacity)
}*/