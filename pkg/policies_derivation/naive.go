package policies_derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"time"
	"math"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/config"
)

type NaivePolicy struct {
	algorithm  		string               //Algorithm's name
	timeWindow 		TimeWindowDerivation //Algorithm used to process the forecasted time serie
	currentState	types.State			 //Current State
	mapVMProfiles   map[string]types.VmProfile
	sysConfiguration	config.SystemConfiguration
}

func (p NaivePolicy) CreatePolicies(processedForecast types.ProcessedForecast, serviceProfile types.ServiceProfile) []types.Policy {
	policies := []types.Policy {}
	newPolicy := types.Policy{}
	newPolicy.StartTimeDerivation = time.Now()
	configurations := []types.Configuration {}

	//Select the performance profile that fits better
	performanceProfile := selectProfile(serviceProfile.PerformanceProfiles)

	for _, it := range processedForecast.CriticalIntervals {
		requests := it.Requests
		services :=  make(map[string]types.ServiceInfo)
		//Compute number of replicas needed depending on requests
		newNumServiceReplicas := int(math.Ceil(requests / performanceProfile.TRN)) * performanceProfile.NumReplicas
		//Compute the max capacity in terms of number of  service replicas for each VM type
		computeVMsCapacity(performanceProfile,&p.mapVMProfiles)

		//Compute new  vmset.
		//This set might be more expensive and with overprovisioning
		vmSet := p.FindSuitableVMs(newNumServiceReplicas, performanceProfile.Limit)
		//vmSet = newVMSet
		if p.sysConfiguration.PolicySettings.UnderprovisioningAllowed {
			underProvReplicas,newVMset:= p.considerIfUnderprovision(vmSet,performanceProfile,requests)
			if newVMset != nil {
				newNumServiceReplicas = underProvReplicas
				vmSet = newVMset
			}
		}

		services[serviceProfile.Name] = types.ServiceInfo {
												Scale: newNumServiceReplicas,
												CPU: performanceProfile.Limit.NumCores,
												Memory: performanceProfile.Limit.Memory,
		}
		state := types.State{
					Services:services,
					VMs:vmSet,
		}

		//update state before next iteration
		timeStart := it.TimeStart
		timeEnd := it.TimeEnd
		totalServicesBootingTime := performanceProfile.BootTimeSec
		setConfiguration(&configurations,state,timeStart,timeEnd,serviceProfile.Name, totalServicesBootingTime, p.sysConfiguration)
	}

	totalConfigurations := len(configurations)
	//Add new policy
	newPolicy.Configurations = configurations
	newPolicy.FinishTimeDerivation = time.Now()
	newPolicy.Algorithm = p.algorithm
	newPolicy.ID = bson.NewObjectId()
	newPolicy.Status = types.DISCARTED	//State by default
	newPolicy.Metrics = types.Metrics {
		NumberConfigurations:totalConfigurations,
	}
	policies = append(policies, newPolicy)
	return policies
}

//Find suitable Vm(s) depending on resources limit, VM Type and Number of replicas that should be deployed
func (p NaivePolicy) FindSuitableVMs(targetNumReplicas int, resourceslimit types.Limit) types.VMScale {
	vmType := p.currentVMType()
	vmScale := make(map[string]int)
	profile := p.mapVMProfiles[vmType]
	maxReplicas := maxReplicasCapacityInVM(profile, resourceslimit)
	if maxReplicas > targetNumReplicas {
		vmScale[vmType] = 1
		return vmScale
	} else if maxReplicas > 0 {
		numVMs := targetNumReplicas / maxReplicas
		vmScale[vmType] = int(numVMs)
		return vmScale
	}
	return vmScale
}

//Return the VM type used by the current Homogeneous VM cluster
func (p NaivePolicy) currentVMType() string {
	//Assumption for p approach: There is only 1 vm Type in current state
	var vmType string
	for k,_ := range p.currentState.VMs {
		vmType = k
	}
	if len(p.currentState.VMs) > 1 {
		log.Warning("Current config has more than one VM type, type %s was selected to continue", vmType)
	}
	return vmType
}

//Compares if according to the minimum percentage of underprovisioning is possible to find a cheaper VM set
//by decreasing the number of replicas and comparing the capacity of a VM set for overprovisioning against the new one
//found for underprovision
func (p NaivePolicy) considerIfUnderprovision(overVmSet types.VMScale, performanceProfile types.PerformanceProfile, requests float64)(int, types.VMScale){
	var newNumServiceReplicas int

	//Compute number of replicas that leads to minimal underprovision
	underNumServiceReplicas := int(math.Floor(requests / performanceProfile.TRN)) * performanceProfile.NumReplicas
	underProvisionTRN := float64(underNumServiceReplicas / performanceProfile.NumReplicas)*performanceProfile.TRN
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
}