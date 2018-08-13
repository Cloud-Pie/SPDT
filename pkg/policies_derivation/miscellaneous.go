package policies_derivation

import "github.com/Cloud-Pie/SPDT/types"

type VMSet struct {
	VMSet                 types.VMScale
	Cost                  float64
	TotalNVMs             int
	TotalReplicasCapacity int
}

// Set missing values for the VMSet structure
func (set *VMSet) setValues(mapVMProfiles map[string]types.VmProfile) {
	cost := float64(0.0)
	totalNVMs := 0
	totalCapacity :=0
	for k,v := range set.VMSet {
		cost += mapVMProfiles[k].Pricing.Price * float64(v)
		totalNVMs += v
		totalCapacity += mapVMProfiles[k].ReplicasCapacity * v
	}
	set.Cost = cost
	set.TotalNVMs = totalNVMs
	set.TotalReplicasCapacity = totalCapacity
}

func computeCapacity(listVMProfiles *[]types.VmProfile, performanceProfile types.PerformanceProfile,  mapVMProfiles *map[string]types.VmProfile) {
	//calculate the capacity of services replicas to each VM type
	for i,v := range *listVMProfiles {
		cap := maxReplicasCapacityInVM(v,performanceProfile.Limit)
		(*listVMProfiles)[i].ReplicasCapacity = cap
		profile := (*mapVMProfiles)[v.Type]
		profile.ReplicasCapacity = cap
		(*mapVMProfiles)[v.Type] = profile
	}
}

//calculate the capacity of services replicas to each VM type
func computeVMsCapacity(performanceProfile types.PerformanceProfile,  mapVMProfiles *map[string]types.VmProfile) {
	for _,v := range *mapVMProfiles {
		cap := maxReplicasCapacityInVM(v,performanceProfile.Limit)
		profile := (*mapVMProfiles)[v.Type]
		profile.ReplicasCapacity = cap
		(*mapVMProfiles)[v.Type] = profile
	}
}