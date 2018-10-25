package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
)

type MSCProfile struct {
	ResourceLimits types.Limit
	NumberReplicas int
	MSC            float64
}

/*
	Constructs different VM clusters (heterogeneous included) to add resources every time the workload
	increases in a factor of deltaLoad.
 */
type DeltaLoadPolicy struct {
	algorithm  		string               //Algorithm's name
	timeWindow 		TimeWindowDerivation //Algorithm used to process the forecasted time serie
	currentState	types.State			 //Current State
	sortedVMProfiles []types.VmProfile    			//List of VM profiles sorted by price
	mapVMProfiles map[string]types.VmProfile
	sysConfiguration	util.SystemConfiguration
}

/*
	Node that represents a candidate option to scale
*/
type Node struct {
	NReplicas	int
	vmType	string
	children []*Node
	vmScale types.VMScale
}

/*
	Tree structure used to create different combinations of VM types
 */
type Tree struct {
	Root *Node
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
func DeltaVMSet(current types.VMScale, candidate types.VMScale) (types.VMScale, types.VMScale){
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

func MapKeysToString(keys map[string] bool)string {
	var vmTypes string
	for k,_ := range keys {
		vmTypes += k + ","
	}
	nkeys := len(vmTypes)
	if nkeys > 0 {
		vmTypes = vmTypes[:nkeys-1]
	}
	return  vmTypes
}

/*
	out:
		String with the name of the current VM type
*/
func biggestVMTypeInSet(vmSet types.VMScale, mapVMProfiles map[string]types.VmProfile) string {
	//It selects teh VM with more resources in case there is more than onw vm type
	var vmType string
	memGB := 0.0
	for k,_ := range vmSet {
		if mapVMProfiles[k].Memory > memGB {
			vmType = k
			memGB =  mapVMProfiles[k].Memory
		}
	}

	return vmType
}