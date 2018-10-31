package execution

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/rest_clients/scheduler"
	"strconv"
	"strings"
)

func TriggerScheduler(policy types.Policy, endpoint string)([] scheduler.StateToSchedule,error) {
	var statesToSchedule  []scheduler.StateToSchedule
	for _, conf := range policy.ScalingActions {
		mapServicesToSchedule := make(map[string]scheduler.ServiceToSchedule)
		state := conf.DesiredState

		for k,v := range state.Services {
			cpu := CPUToString(v.CPU)
			memory := memGBToBytes(v.Memory)
			replicas := v.Scale
			mapServicesToSchedule[k] = scheduler.ServiceToSchedule{
				Scale:replicas,
				CPU:cpu,
				Memory:memory,
			}
		}
		vms := addRemovedKeys(conf.InitialState.VMs, conf.DesiredState.VMs)
		stateToSchedule := scheduler.StateToSchedule{
			LaunchTime:conf.TimeStartTransition,
			Services:mapServicesToSchedule,
			Name:state.Hash,
			VMs:vms,
			ExpectedStart:conf.TimeStart,
		}
		statesToSchedule = append(statesToSchedule, stateToSchedule)

		err := scheduler.CreateState(stateToSchedule, endpoint)
		if err != nil {
			return statesToSchedule,err
		}
	}
	return statesToSchedule,nil
}

func CPUToString(value float64) string {
	value = value * 1000
	cpu := strconv.FormatFloat(value, 'f', 1, 64)
	cpu = strings.Replace(cpu,".0","m",1)
	return cpu
}

func memGBToBytes(value float64) int64 {
	mem := 1000000000 * value
	memInt := int64(mem)
	return memInt
}

func memBytesToGB(value int64) float64 {
	memFloat := float64(value) / 1000000000
	return memFloat
}

func stringToCPUCores(value string) float64 {
	value = strings.Replace(value, "m", "", -1)
	cpu,err := strconv.ParseFloat(value, 64)

	if err == nil {
		cpu = cpu / 1000.0
	}

	return cpu
}

func RetrieveCurrentState(endpoint string ) (types.State, error) {
	var policyState types.State
	stateScheduled, _ := scheduler.InfraCurrentState(endpoint)
	mapServicesScheduled := stateScheduled.Services
	policyServices := make(map[string]types.ServiceInfo)

	for k,v := range mapServicesScheduled {
		mem := memBytesToGB(v.Memory)
		cpu := stringToCPUCores(v.CPU)
		replicas := v.Scale
		policyServices[k] = types.ServiceInfo{
			Memory:mem,
			CPU:cpu,
			Scale:replicas,
		}
	}

	policyState = types.State {
		VMs:stateScheduled.VMs,
		Services:policyServices,
	}
	return policyState,nil
}

//Note: Adjustment of keys required for PASSA
//It should include the keys of the VM that should be removed
func addRemovedKeys(initialVMSet types.VMScale, desiredVMSet types.VMScale) types.VMScale {
	for k,_ := range initialVMSet {
		if _,ok := desiredVMSet[k]; !ok {
			desiredVMSet[k] = 0
		}
	}
	return desiredVMSet
}