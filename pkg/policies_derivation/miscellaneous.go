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

func VMListToMap(listVMProfiles []types.VmProfile) map[string]types.VmProfile{
	mapVMProfiles := make(map[string]types.VmProfile)
	for _,p := range listVMProfiles {
		mapVMProfiles[p.Type] = p
	}
	return mapVMProfiles
}

//Creates a map duplicate
func copyMap(m map[string]int) map[string]int {
	newM := make(map[string] int)
	for k,v := range m {
		newM[k]=v
	}
	return newM
}

//Compare 2 VM Sets
//Params:
// - currentVMSet: current configuration of VMs
// - candidateVMSet: set of VMs to which a reconfiguration could be possible
// Output:
// - Set of VMs that were added
// - Set of VMs that were removed
func deltaVMSet(current types.VMScale, candidate types.VMScale) (types.VMScale, types.VMScale){
	delta := types.VMScale{}
	startSet := types.VMScale{}
	shutdownSet := types.VMScale{}

	for k,_ :=  range current {
		if _,ok := candidate[k]; ok {
			delta[k] = -1 * (current[k] - candidate[k])
			if (delta[k]> 0) {
				startSet[k] = delta[k]
			} else if (delta[k] < 0) {
				shutdownSet[k] = -1 * delta[k]
			}
		} else {
			delta[k] = -1 * current[k]
			shutdownSet[k] =  current[k]
		}
	}

	for k,_ :=  range candidate {
		if _,ok := current[k]; !ok {
			delta[k] = candidate[k]
			startSet[k] = candidate[k]
		}
	}
	return startSet, shutdownSet
}

//Removes the keys of a map that have as value 0
func cleanKeys(m map[string]int){
	for k,v := range m {
		if v == 0 {
			delete(m, k)
		}
	}
}