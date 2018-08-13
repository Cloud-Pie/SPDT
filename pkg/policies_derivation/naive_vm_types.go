package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/config"
	"strconv"
)

type NaiveTypesPolicy struct {
	algorithm  string               //Algorithm's name
	timeWindow TimeWindowDerivation //Algorithm used to process the forecasted time serie
	mapVMProfiles   map[string]types.VmProfile
	sysConfiguration	config.SystemConfiguration
}

func (p NaiveTypesPolicy) CreatePolicies(processedForecast types.ProcessedForecast, serviceProfile types.ServiceProfile) [] types.Policy {
	policies := []types.Policy{}
	//Compute results for cluster of each type
	for vmType, _ := range p.mapVMProfiles {
		newPolicy := types.Policy{}
		newPolicy.Metrics = types.PolicyMetrics {
			StartTimeDerivation:time.Now(),
		}
		configurations := []types.Configuration{}
		underprovisionAllowed := p.sysConfiguration.PolicySettings.UnderprovisioningAllowed
		//Select the performance profile that fits better
		performanceProfile := selectProfile(serviceProfile.PerformanceProfiles)

		for _, it := range processedForecast.CriticalIntervals {
			requests := it.Requests
			services :=  make(map[string]types.ServiceInfo)

			//Compute number of replicas needed depending on requests
			newNumServiceReplicas := int(math.Ceil(requests / performanceProfile.TRN)) * performanceProfile.NumReplicas
			//Compute the max capacity in terms of number of  service replicas for each VM type
			computeVMsCapacity(performanceProfile,&p.mapVMProfiles)

			//Find suitable Vm(s) depending on resources limit and current state
			vmSet := p.FindSuitableVMs(newNumServiceReplicas,performanceProfile.Limit,vmType)
			if len(vmSet) == 0 { break } //The VM Type doesn't fit for the limits

			if underprovisionAllowed {
				underProvReplicas,newVMset:= p.considerIfUnderprovision(vmSet,performanceProfile,requests, vmType)
				if newVMset != nil {
					newNumServiceReplicas = underProvReplicas
					vmSet = newVMset
				}
			}

			services[serviceProfile.Name] = types.ServiceInfo{
				Scale:  newNumServiceReplicas,
				CPU:    performanceProfile.Limit.NumCores,
				Memory: performanceProfile.Limit.Memory,
			}

			state := types.State{}
			state.Services = services
			state.VMs = vmSet

			timeStart := it.TimeStart
			timeEnd := it.TimeEnd
			totalServicesBootingTime := performanceProfile.BootTimeSec
			setConfiguration(&configurations,state,timeStart,timeEnd,serviceProfile.Name, totalServicesBootingTime, p.sysConfiguration)
		}

		totalConfigurations := len(configurations)
		if len(configurations) > 0 {
			//Add new policy
			parameters := make(map[string]string)
			parameters[types.METHOD] = "horizontal"
			parameters[types.ISHETEREOGENEOUS] = strconv.FormatBool(false)
			parameters[types.ISUNDERPROVISION] = strconv.FormatBool(underprovisionAllowed)
			if underprovisionAllowed {
				parameters[types.MAXUNDERPROVISION] = strconv.FormatFloat(p.sysConfiguration.PolicySettings.MaxUnderprovision, 'f', -1, 64)
			}
			newPolicy.Configurations = configurations

			newPolicy.Algorithm = p.algorithm
			newPolicy.ID = bson.NewObjectId()
			newPolicy.Status = types.DISCARTED	//State by default
			newPolicy.Parameters = parameters
			newPolicy.Metrics.NumberConfigurations = totalConfigurations
			newPolicy.Metrics.FinishTimeDerivation = time.Now()
			policies = append(policies, newPolicy)
		}
	}
	return policies
}
//Find suitable Vm(s) depending on resources limit, VM Type and Number of replicas that should be deployed
func (p NaiveTypesPolicy) FindSuitableVMs(numReplicas int, resourceslimit types.Limit, vmType string) types.VMScale {
	vmScale := make(map[string]int)
	profile := p.mapVMProfiles[vmType]
	maxReplicas := maxReplicasCapacityInVM(profile, resourceslimit)
	if maxReplicas > numReplicas {
		vmScale[vmType] = 1
		return vmScale
	} else if maxReplicas > 0 {
		numVMs := numReplicas / maxReplicas
		vmScale[vmType] = int(numVMs)
		return vmScale
	}
	return vmScale
}

//Compares if according to the minimum percentage of underprovisioning is possible to find a cheaper VM set
//by decreasing the number of replicas and comparing the capacity of a VM set for overprovisioning against the new one
//found for underprovision
func (p NaiveTypesPolicy) considerIfUnderprovision(overVmSet types.VMScale, performanceProfile types.PerformanceProfile, requests float64, vmType string)(int, types.VMScale){
	var newNumServiceReplicas int

	//Compute number of replicas that leads to minimal underprovision
	underNumServiceReplicas := int(math.Floor(requests / performanceProfile.TRN)) * performanceProfile.NumReplicas
	underProvisionTRN := float64(underNumServiceReplicas / performanceProfile.NumReplicas)*performanceProfile.TRN
	percentageUnderProvisioned := underProvisionTRN * requests / 100.0
	//Compare if underprovision in terms of number of request is acceptable
	if percentageUnderProvisioned <= p.sysConfiguration.PolicySettings.MaxUnderprovision {
		vmSet := p.FindSuitableVMs(underNumServiceReplicas, performanceProfile.Limit, vmType)
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
}