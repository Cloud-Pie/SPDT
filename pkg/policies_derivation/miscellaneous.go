package policies_derivation

import "github.com/Cloud-Pie/SPDT/types"
type TRNProfile struct {
	ResourceLimits	types.Limit
	NumberReplicas	int
	TRN				float64
}

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


/* Compute the maximum capacity regarding the number of replicas hosted in each VM type
	in:
		@listVMProfiles
		@limits
		@mapVMProfiles
*/
func computeCapacity(listVMProfiles *[]types.VmProfile, limits types.Limit,  mapVMProfiles *map[string]types.VmProfile) {
	//calculate the capacity of services replicas to each VM type
	for i,v := range *listVMProfiles {
		cap := maxReplicasCapacityInVM(v,limits)
		(*listVMProfiles)[i].ReplicasCapacity = cap
		profile := (*mapVMProfiles)[v.Type]
		profile.ReplicasCapacity = cap
		(*mapVMProfiles)[v.Type] = profile
	}
}

/* Compute the maximum capacity regarding the number of replicas hosted in each VM type
	in:
		@limits
		@mapVMProfiles
*/
func computeVMsCapacity(limits types.Limit,  mapVMProfiles *map[string]types.VmProfile) {
	for _,v := range *mapVMProfiles {
		cap := maxReplicasCapacityInVM(v,limits)
		profile := (*mapVMProfiles)[v.Type]
		profile.ReplicasCapacity = cap
		(*mapVMProfiles)[v.Type] = profile
	}
}

/* Build a map taking as input a list
	in:
		@listVMProfiles
	out:
		@map[string]types.VmProfile
*/
func VMListToMap(listVMProfiles []types.VmProfile) map[string]types.VmProfile{
	mapVMProfiles := make(map[string]types.VmProfile)
	for _,p := range listVMProfiles {
		mapVMProfiles[p.Type] = p
	}
	return mapVMProfiles
}

/* Build a list of maps
	in:
		@map[string]int - Map with VM type as key and number of VMs of that VM type as value
	out:
		@[]types.VmProfile - List of key values
*/
func mapToList(vmSet map[string]int)[]types.StructMap {
	var ss [] types.StructMap
	for k, v := range vmSet {
		ss = append(ss, types.StructMap{k, v})
	}
	return ss
}

/* Duplicate a map
	in:
		@map[string]int
	out:
		@map[string]int
*/
func copyMap(m map[string]int) map[string]int {
	newM := make(map[string] int)
	for k,v := range m {
		newM[k]=v
	}
	return newM
}

/* compare the changes (vms added, vms removed) from one VM set to a candidate VM set
	in:
		@current	- Map with current VM cluster
		@candidate	- Map with candidate VM cluster
	out:
		@VMScale	- Map with VM cluster of the VMs that were added into the candidate VM set
		@VMScale	- Map with VM cluster of the VMs that were removed from the candidate VM set
*/
func deltaVMSet(current types.VMScale, candidate types.VMScale) (types.VMScale, types.VMScale){
	delta := types.VMScale{}
	startSet := types.VMScale{}
	shutdownSet := types.VMScale{}
	sameSet := types.VMScale{}

	for k,_ :=  range current {
		if _,ok := candidate[k]; ok {
			delta[k] = current[k] - candidate[k]
			if (delta[k] < 0) {
				startSet[k] = -1 * delta[k]
			} else if (delta[k] > 0) {
				shutdownSet[k] = delta[k]
			} else {
				sameSet[k] = current[k]
			}
		} else {
			shutdownSet[k] =  current[k]
		}
	}

	for k,_ :=  range candidate {
		if _,ok := current[k]; !ok {
			startSet[k] = candidate[k]
		}
	}
	return startSet, shutdownSet
}

/* Remove the keys from a map where the value is zero
	in:
		@m	- Map
*/
func cleanKeys(m map[string]int){
	for k,v := range m {
		if v == 0 {
			delete(m, k)
		}
	}
}

func vmTypesList(mapVMProfiles map[string]types.VmProfile) string{
	var vmTypes string
	for k,_ := range mapVMProfiles {
		vmTypes += k + ", "
	}

	return  vmTypes
}