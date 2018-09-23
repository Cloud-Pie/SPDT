package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
	"github.com/Cloud-Pie/SPDT/pkg/schedule"
	"math"
	"sort"
	"github.com/Cloud-Pie/SPDT/rest_clients/performance_profiles"
	"github.com/op/go-logging"
	"github.com/Cloud-Pie/SPDT/storage"
	"github.com/Cloud-Pie/SPDT/config"
	"errors"
	"github.com/cnf/structhash"
	"strings"
)

var log = logging.MustGetLogger("spdt")
var systemConfiguration config.SystemConfiguration
var initialState types.State

//Interface for strategies of how to scale
type PolicyDerivation interface {
	CreatePolicies (processedForecast types.ProcessedForecast) []types.Policy
	FindSuitableVMs (numberReplicas int, limits types.Limit) types.VMScale
}

//Interface for strategies of when to scale
type TimeWindowDerivation interface {
	NumberIntervals()	int
	WindowDerivation(values []float64, times [] time.Time)	types.ProcessedForecast
}

/* Derive scaling policies
	in:
		@poiList []types.PoI
		@values []float64
		@times [] time.Time
		@sortedVMProfiles []VmProfile
		@sysConfiguration SystemConfiguration
	out:
		@[]types.Policy
*/
func Policies(poiList []types.PoI, values []float64, times [] time.Time, sortedVMProfiles []types.VmProfile, sysConfiguration config.SystemConfiguration) ([]types.Policy, error) {
	var policies []types.Policy
	systemConfiguration = sysConfiguration

	log.Info("Request current state" )
	currentState,err := schedule.RetrieveCurrentState(sysConfiguration.SchedulerComponent.Endpoint + util.ENDPOINT_CURRENT_STATE)

	if err != nil {
		log.Error("Error to get current state %s", err.Error() )
	} else {
		log.Info("Finish request for current state" )
	}
	if currentState.Services[systemConfiguration.MainServiceName].Scale == 0 {
		return policies, errors.New("Service "+ systemConfiguration.MainServiceName +" is not deployed")
	}

	timeWindows := SmallStepOverProvision{PoIList:poiList}
	processedForecast := timeWindows.WindowDerivation(values,times)
	initialState = currentState
	mapVMProfiles := VMListToMap(sortedVMProfiles)

	switch sysConfiguration.PreferredAlgorithm {
	case util.NAIVE_ALGORITHM:
		naive := NaivePolicy {algorithm:util.NAIVE_ALGORITHM, timeWindow:timeWindows,
							 currentState:currentState, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies = naive.CreatePolicies(processedForecast)

	case util.BASE_INSTANCE_ALGORITHM:
		base := BestBaseInstancePolicy{algorithm:util.BASE_INSTANCE_ALGORITHM, timeWindow:timeWindows,
										currentState:currentState,mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies = base.CreatePolicies(processedForecast)

	case util.SMALL_STEP_ALGORITHM:
		sstep := StepRepackPolicy{algorithm:util.SMALL_STEP_ALGORITHM, timeWindow:timeWindows,
									mapVMProfiles:mapVMProfiles, sortedVMProfiles:sortedVMProfiles, sysConfiguration: sysConfiguration, currentState:currentState}
		policies = sstep.CreatePolicies(processedForecast)

	case util.SEARCH_TREE_ALGORITHM:
		tree := TreePolicy {algorithm:util.SEARCH_TREE_ALGORITHM, timeWindow:timeWindows, currentState:currentState,
			sortedVMProfiles:sortedVMProfiles,mapVMProfiles:mapVMProfiles,sysConfiguration: sysConfiguration}
		policies = tree.CreatePolicies(processedForecast)

	case util.DELTA_REPACKED:
		algorithm := DeltaRepackedPolicy {algorithm:util.DELTA_REPACKED, timeWindow:timeWindows, currentState:currentState,
		sortedVMProfiles:sortedVMProfiles, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies = algorithm.CreatePolicies(processedForecast)
	default:
		//naive
		naive := NaivePolicy {algorithm:util.NAIVE_ALGORITHM, timeWindow:timeWindows,
			currentState:currentState, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies1 := naive.CreatePolicies(processedForecast)
		policies = append(policies, policies1...)
		//types
		base := BestBaseInstancePolicy{algorithm:util.BASE_INSTANCE_ALGORITHM, timeWindow:timeWindows,
			currentState:currentState, sortedVMProfiles:sortedVMProfiles, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies2 := base.CreatePolicies(processedForecast)
		policies = append(policies, policies2...)
		//sstep
		sstep := StepRepackPolicy{algorithm:util.SMALL_STEP_ALGORITHM, timeWindow:timeWindows,
			sortedVMProfiles:sortedVMProfiles, mapVMProfiles:mapVMProfiles,sysConfiguration: sysConfiguration, currentState:currentState}
		policies3 := sstep.CreatePolicies(processedForecast)
		policies = append(policies, policies3...)
		//delta repack
		algorithm := DeltaRepackedPolicy {algorithm:util.DELTA_REPACKED, timeWindow:timeWindows, currentState:currentState,
			sortedVMProfiles:sortedVMProfiles, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies4 := algorithm.CreatePolicies(processedForecast)
		policies = append(policies, policies4...)

		//tree
		tree := TreePolicy {algorithm:util.SEARCH_TREE_ALGORITHM, timeWindow:timeWindows, currentState:currentState,
			sortedVMProfiles:sortedVMProfiles,mapVMProfiles:mapVMProfiles,sysConfiguration: sysConfiguration}
		policies5 := tree.CreatePolicies(processedForecast)
		policies = append(policies, policies5...)
	}
	return policies, err
}

/* Compute the booting time that will take a set of VMS
	in:
		@vmsScale types.VMScale
		@sysConfiguration SystemConfiguration
	out:
		@int	Time in seconds that the booting wil take
*/
func computeVMBootingTime(vmsScale types.VMScale, sysConfiguration config.SystemConfiguration) float64 {
	bootTime := 0.0
	//Check in db if already data is stored
	vmBootingProfileDAO := storage.GetVMBootingProfileDAO(sysConfiguration.MainServiceName)

	//Call API
	for vmType, n := range vmsScale {
		times, err := vmBootingProfileDAO.BootingShutdownTime(vmType, n)
		if err != nil {
			url := sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_VM_TIMES
			csp := sysConfiguration.CSP
			region := sysConfiguration.Region
			times, err = performance_profiles.GetBootShutDownProfileByType(url,vmType, n, csp, region)
			if err != nil {
				log.Error("Error in bootingTime query  type %s %d VMS. Details: %s", vmType, n, err.Error())
				log.Warning("Takes the biggest time available")
				times.BootTime = util.DEFAULT_VM_BOOT_TIME
			}else {
				vmBootingProfile,_ := vmBootingProfileDAO.FindByType(vmType)
				vmBootingProfile.InstancesValues = append(vmBootingProfile.InstancesValues, times)
				vmBootingProfileDAO.UpdateByType(vmType, vmBootingProfile)
			}
		}
		bootTime += times.BootTime
	}
	return bootTime
}


/* Compute the termination time of a set of VMs
	in:
		@vmsScale types.VMScale
		@sysConfiguration SystemConfiguration
	out:
		@int	Time in seconds that the termination wil take
*/
func computeVMTerminationTime(vmsScale types.VMScale, sysConfiguration config.SystemConfiguration) float64 {
	terminationTime := 0.0
	//Check in db if already data is stored
	vmBootingProfileDAO := storage.GetVMBootingProfileDAO(sysConfiguration.MainServiceName)

	//Call API
	for vmType, n := range vmsScale {
		times, err := vmBootingProfileDAO.BootingShutdownTime(vmType, n)
		if err != nil {
			url := sysConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_VM_TIMES
			csp := sysConfiguration.CSP
			region := sysConfiguration.Region
			times, err = performance_profiles.GetBootShutDownProfileByType(url,vmType, n, csp, region)
			if err != nil {
				log.Error("Error in terminationTime query for type %s %d VMS. Details: %s", vmType, n, err.Error())
				log.Warning("Takes default shutdown")
				times.ShutDownTime = util.DEFAULT_VM_SHUTDOWN_TIME
			} else {
				vmBootingProfile,_ := vmBootingProfileDAO.FindByType(vmType)
				vmBootingProfile.InstancesValues = append(vmBootingProfile.InstancesValues, times)
				vmBootingProfileDAO.UpdateByType(vmType, vmBootingProfile)
			}
		}
		terminationTime += times.ShutDownTime
	}
	return terminationTime
}

/* Compute the max number of service replicas (Replicas capacity) that a VM can host
	in:
		@vmProfile types.VmProfile
		@resourceLimit types.Limits
	out:
		@int	Number of replicas
*/
func maxReplicasCapacityInVM(vmProfile types.VmProfile, resourceLimit types.Limit) int {
	//For memory resources, Kubernetes Engine reserves aprox 6% of cores and 25% of Mem
		cpuCoresAvailable := vmProfile.CPUCores  *(1-util.PERCENTAGE_REQUIRED_k8S_INSTALLATION_CPU)
		memGBAvailable := vmProfile.Memory * (1-util.PERCENTAGE_REQUIRED_k8S_INSTALLATION_MEM)

		m := float64(cpuCoresAvailable) / float64(resourceLimit.CPUCores)
		n := float64(memGBAvailable) / float64(resourceLimit.MemoryGB)
		numReplicas := math.Min(n,m)
		return int(numReplicas)
}

/* Select the service profile for a given container limit resources
	in:
		@requests	float64 - number of requests that the service should serve
		@limits types.Limits	- resource limits (cpu cores and memory gb) configured in the container
		@underProvision bool	- flag that indicate if when searching for a service profile, the underprovision is allowed
	out:
		@ContainersConfig	- configuration with number of replicas and limits that best fit for the number of requests
*/
func selectProfileByLimits(requests float64, limits types.Limit, underProvision bool) types.ContainersConfig {
	var containerConfig types.ContainersConfig
	serviceProfileDAO := storage.GetPerformanceProfileDAO(systemConfiguration.MainServiceName)
	overProvisionConfig, err1 := serviceProfileDAO.MatchByLimitsOver(limits.CPUCores, limits.MemoryGB, requests)
	underProvisionConfig, err2 := serviceProfileDAO.MatchByLimitsUnder(limits.CPUCores, limits.MemoryGB, requests)

	if underProvision && err2 == nil {
		containerConfig = underProvisionConfig
	} else if err1 == nil{
		containerConfig = overProvisionConfig
	} else if err2 == nil {
		containerConfig = underProvisionConfig

		url := systemConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_SERVICE_UPDATE_PROFILE
		appName := systemConfiguration.AppName
		appType := systemConfiguration.AppType
		mainServiceName := systemConfiguration.MainServiceName
		mscSetting,err := performance_profiles.GetPredictedReplicas(url,appName,appType,mainServiceName,requests,limits.CPUCores, limits.MemoryGB)

		newMSCSetting := types.MSCSimpleSetting{}
		if err == nil{
			containerConfig.MSCSetting.Replicas = mscSetting.Replicas
			containerConfig.MSCSetting.MSCPerSecond = mscSetting.MSCPerSecond.RegBruteForce
			newMSCSetting.Replicas = mscSetting.Replicas
			newMSCSetting.MSCPerSecond = mscSetting.MSCPerSecond.RegBruteForce
			if mscSetting.BootTimeMs > 0 {
				newMSCSetting.BootTimeSec = mscSetting.BootTimeMs / 1000
			} else {
				newMSCSetting.BootTimeSec = util.DEFAULT_POD_BOOT_TIME
			}
			newMSCSetting.BootTimeSec = mscSetting.BootTimeMs / 1000

		} else {
			/*numberReplicas := float64(containerConfig.MSCSetting.Replicas) * requests / containerConfig.MSCSetting.MSCPerSecond
			containerConfig.MSCSetting.Replicas = int(numberReplicas)
			containerConfig.MSCSetting.MSCPerSecond = requests
			newMSCSetting = types.MSCSimpleSetting{Replicas:int(numberReplicas), MSCPerSecond:requests, BootTimeSec:100}*/
		}

		profile,_:= serviceProfileDAO.FindProfileTRN(limits.CPUCores, limits.MemoryGB, newMSCSetting.Replicas)
		if profile.ID == "" {
			profile,_= serviceProfileDAO.FindProfileByLimits(limits)
			profile.MSCSettings = append(profile.MSCSettings,newMSCSetting)
			err3 := serviceProfileDAO.UpdateById(profile.ID, profile)
			if err3 != nil{
				log.Error("Performance profile not updated")
			}
		}
	}
	//defer serviceProfileDAO.Session.Close()
	return containerConfig
}

/* Select the service profile for any limit resources that satisfies the number of requests
	in:
		@requests	float64 - number of requests that the service should serve
		@underProvision bool	- flag that indicate if when searching for a service profile, the underprovision is allowed
	out:
		@ContainersConfig	- configuration with number of replicas and limits that best fit for the number of requests
*/
func selectProfileUnderVMLimits(requests float64,  limits types.Limit, underProvision bool) (types.ContainersConfig, error){
	var profiles []types.ContainersConfig
	var profile  types.ContainersConfig
	serviceProfileDAO := storage.GetPerformanceProfileDAO(systemConfiguration.MainServiceName)
	profilesUnder,err1:= serviceProfileDAO.MatchProfileFitLimitsUnder(limits.CPUCores, limits.MemoryGB, requests)
	profilesOver,err2 := serviceProfileDAO.MatchProfileFitLimitsOver(limits.CPUCores, limits.MemoryGB, requests)

	if underProvision && err1 == nil && len(profilesUnder) > 0 {
		profiles = profilesUnder
		sort.Slice(profiles, func(i, j int) bool {
			utilizationFactori := float64(profiles[i].MSCSetting.Replicas) * profiles[i].Limits.CPUCores +  float64(profiles[i].MSCSetting.Replicas) * profiles[i].Limits.MemoryGB
			utilizationFactorj := float64(profiles[j].MSCSetting.Replicas) * profiles[j].Limits.CPUCores + float64(profiles[j].MSCSetting.Replicas) * profiles[i].Limits.MemoryGB
			return utilizationFactori < utilizationFactorj
		})

	} else if err2 == nil{
		profiles = profilesOver
		sort.Slice(profiles, func(i, j int) bool {
			return profiles[i].MSCSetting.MSCPerSecond < profiles[j].MSCSetting.MSCPerSecond
		})
	} else if err1 == nil {
		profiles = profilesUnder
	}
	//defer serviceProfileDAO.Session.Close()

	if len(profiles) > 0{
		profile = profiles[0]
		return profile, nil
	} else {
		return profile, errors.New("No profile found")
	}
}

/* Select the service profile for any limit resources that satisfies the number of requests
	in:
		@numberReplicas	int - number of replicas
		@limits bool types.Limits - limits constraints(cpu cores and memory gb) per replica
	out:
		@float64	- Max number of request for this containers configuration
*/
func getStateLoadCapacity(numberReplicas int, limits types.Limit) float64 {
	currentLoadCapacity := 0.0
	serviceProfileDAO := storage.GetPerformanceProfileDAO(systemConfiguration.MainServiceName)
	profile,_ := serviceProfileDAO.FindProfileTRN(limits.CPUCores, limits.MemoryGB, numberReplicas)
	if len(profile.MSCSettings) > 0 {
		currentLoadCapacity = profile.MSCSettings[0].MSCPerSecond
	}
	//defer serviceProfileDAO.Session.Close()
	return currentLoadCapacity
}

/* Utility method to set up each scaling configuration
*/
func setScalingSteps(scalingSteps *[]types.ScalingStep, currentState types.State,newState types.State, timeStart time.Time, timeEnd time.Time, totalServicesBootingTime float64, stateLoadCapacity float64) {
	nScalingSteps := len(*scalingSteps)
	timeStartBill := timeStart
	timeEndBill := timeEnd
	if nScalingSteps >= 1 && newState.Equal((*scalingSteps)[nScalingSteps-1].DesiredState) {
		(*scalingSteps)[nScalingSteps-1].TimeEnd = timeEnd
		timeBillingStarted := (*scalingSteps)[nScalingSteps-1].TimeStartBilling
		timeEndBill = updateEndBillingTime(timeBillingStarted,timeEnd)
		(*scalingSteps)[nScalingSteps-1].TimeEndBilling = timeEndBill
	} else {
		//var deltaTime int //time in seconds
		var shutdownVMDuration float64
		var startTransitionTime time.Time
		var currentVMSet types.VMScale
		currentVMSet = currentState.VMs
		vmAdded, vmRemoved := DeltaVMSet(currentVMSet, newState.VMs)
		nVMRemoved := len(vmRemoved)
		nVMAdded := len(vmAdded)

		if nVMRemoved > 0 && nVMAdded > 0 {
			//case 1: There is an overlaping of configurations
			if  nScalingSteps >= 1 {
				shutdownVMDuration = computeVMTerminationTime(vmRemoved, systemConfiguration)
				previousTimeEnd := (*scalingSteps)[nScalingSteps-1].TimeEnd
				(*scalingSteps)[nScalingSteps-1].TimeEnd = previousTimeEnd.Add(time.Duration(shutdownVMDuration) * time.Second)
			}
			startTransitionTime = computeScaleOutTransitionTime(vmAdded, true, timeStart, totalServicesBootingTime)
		} else if nVMRemoved > 0 && nVMAdded == 0 {
			//case 2:  Scale in,
			shutdownVMDuration = computeVMTerminationTime(vmRemoved, systemConfiguration)
			startTransitionTime = timeStart.Add(-1 * time.Duration(shutdownVMDuration) * time.Second)

		} else if (nVMRemoved == 0 && nVMAdded > 0) || ( nVMRemoved == 0 && nVMAdded == 0 ) {
			//case 3: Scale out
			startTransitionTime = computeScaleOutTransitionTime(vmAdded, true, timeStart, totalServicesBootingTime)
		}
		/*removeBilledVMs := checkBillingPeriod(systemConfiguration.PricingModel.BillingUnit, nVMRemoved, lastBilledStartedTime,timeStart)
		if !removeBilledVMs {
			newVMSet := currentVMSet
			newVMSet.Merge(vmAdded)
			newState.VMs = newVMSet
		}*/
		//newState.LaunchTime = startTransitionTime
		name,_ := structhash.Hash(newState, 1)
		newState.Hash = strings.Replace(name, "v1_", "", -1)
		*scalingSteps = append(*scalingSteps,
			types.ScalingStep{
				InitialState:currentState,
				DesiredState:        newState,
				TimeStart:           timeStart,
				TimeEnd:             timeEnd,
				Metrics:             types.ConfigMetrics{RequestsCapacity:stateLoadCapacity,},
				TimeStartBilling:    timeStartBill,
				TimeEndBilling:      timeEndBill,
				TimeStartTransition: startTransitionTime,
			})
	}
}

/* Build Heterogeneous cluster to deploy a number of replicas, each one with the defined constraint limits
	in:
		@numberReplicas	int - number of replicas
		@limits bool types.Limits - limits constraints(cpu cores and memory gb) per replica
		@mapVMProfiles - map with the profiles of VMs available
	out:
		@VMScale	- Map with the type of VM as key and the number of vms as value
*/
func buildHeterogeneousVMSet(numberReplicas int, limits types.Limit, mapVMProfiles map[string]types.VmProfile) (types.VMScale,error) {
	var err error
	tree := &Tree{}
	node := new(Node)
	node.NReplicas = numberReplicas
	node.vmScale = make(map[string]int)
	tree.Root = node
	candidateVMSets := []types.VMScale {}
	computeVMsCapacity(limits, &mapVMProfiles)

	buildTree(tree.Root, numberReplicas,&candidateVMSets,mapVMProfiles)

	if len(candidateVMSets)> 0{
		sort.Slice(candidateVMSets, func(i, j int) bool {
			costi := candidateVMSets[i].Cost(mapVMProfiles)
			costj := candidateVMSets[j].Cost(mapVMProfiles)
			if costi < costj {
				return true
			} else if costi ==  costj {
				return candidateVMSets[i].TotalVMs() >= candidateVMSets[j].TotalVMs()
			}
			return false
		})
		return candidateVMSets[0],err
	}else {
		return types.VMScale{},errors.New("No VM Candidate")
	}
}

/*
	Form scaling options using clusters of heterogeneous VMs
	Builds a tree to form the different combinations
	in:
		@node			- Node of the tree
		@numberReplicas	- Number of replicas that the VM set should host
		@vmScaleList	- List filled with the candidates VM sets
*/
func buildTree(node *Node, numberReplicas int, vmScaleList *[]types.VMScale, mapVMProfiles map[string]types.VmProfile) *Node {
	if node.NReplicas == 0 {
		return node
	}
	for k,v := range mapVMProfiles {
		maxReplicas := v.ReplicasCapacity
		if maxReplicas >= numberReplicas {
			newNode := new(Node)
			newNode.vmType = k
			newNode.NReplicas = 0
			newNode.vmScale = copyMap(node.vmScale)
			if _, ok := newNode.vmScale[newNode.vmType]; ok {
				newNode.vmScale[newNode.vmType] = newNode.vmScale[newNode.vmType]+1
			} else {
				newNode.vmScale[newNode.vmType] = 1
			}
			node.children = append(node.children, newNode)
			*vmScaleList = append(*vmScaleList, newNode.vmScale)
			//return node
		} else if maxReplicas > 0 {
			newNode := new(Node)
			newNode.vmType = k
			newNode.NReplicas = numberReplicas -maxReplicas
			newNode.vmScale = copyMap(node.vmScale)
			if _, ok := newNode.vmScale[newNode.vmType]; ok {
				newNode.vmScale[newNode.vmType] = newNode.vmScale[newNode.vmType] + 1
			} else {
				newNode.vmScale[newNode.vmType] = 1
			}
			newNode = buildTree(newNode, numberReplicas-maxReplicas, vmScaleList, mapVMProfiles)
			node.children = append(node.children, newNode)
		}
	}
	return node
}

/* Build Homogeneous cluster to deploy a number of replicas, each one with the defined constraint limits
	in:
		@numberReplicas	int - number of replicas
		@limits bool types.Limits - limits constraints(cpu cores and memory gb) per replica
		@mapVMProfiles - map with the profiles of VMs available
	out:
		@VMScale	- Map with the type of VM as key and the number of vms as value
*/
func buildHomogeneousVMSet(numberReplicas int, limits types.Limit, mapVMProfiles map[string]types.VmProfile) (types.VMScale,error) {
	var err error
	candidateVMSets := []types.VMScale{}
	for _,v := range mapVMProfiles {
		vmScale :=  make(map[string]int)
		replicasCapacity :=  maxReplicasCapacityInVM(v, limits)
		if replicasCapacity > 0 {
			numVMs := math.Ceil(float64(numberReplicas) / float64(replicasCapacity))
			vmScale[v.Type] = int(numVMs)
			candidateVMSets = append(candidateVMSets, vmScale)
		}
	}
	if len(candidateVMSets) > 0 {
		sort.Slice(candidateVMSets, func(i, j int) bool {
			costi := candidateVMSets[i].Cost(mapVMProfiles)
			costj := candidateVMSets[j].Cost(mapVMProfiles)
			if costi < costj {
				return true
			} else if costi ==  costj {
				return candidateVMSets[i].TotalVMs() >= candidateVMSets[j].TotalVMs()
			}
			return false
		})
		return candidateVMSets[0],err
	}else {
		return types.VMScale{},errors.New("No VM Candidate")
	}
}

/* Validate if the supplied load with under provision does not exceed the maximum percentage of under provision allowed
	in:
		@demandedRequests	float64 - Demanded load in terms of requests
		@suppliedRequests float64 - Supplied under provisioned load
		@percentageAllowed float64 - Max percentage under provision allowed
	out:
		@bool	- boolean flag that indicates that the supplied load even though under provision
					is still in the allowed under provision range
*/
func isUnderProvisionInRange(demandedRequests float64, suppliedRequests float64, percentageAllowed float64) bool{
	underProvisionedPercentage := (demandedRequests - suppliedRequests) * demandedRequests / suppliedRequests
	if underProvisionedPercentage <= percentageAllowed {
		return true
	} else {
		return false
	}
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

	utilizationFactorCurrent :=  currentConfiguration.Limits.MemoryGB * float64(currentConfiguration.MSCSetting.Replicas) +
		currentConfiguration.Limits.CPUCores* float64(currentConfiguration.MSCSetting.Replicas)

	utilizationFactorNew := newCandidateConfiguration.Limits.MemoryGB * float64(newCandidateConfiguration.MSCSetting.Replicas) +
		newCandidateConfiguration.Limits.CPUCores* float64(newCandidateConfiguration.MSCSetting.Replicas)

	if utilizationFactorNew < utilizationFactorCurrent {
		return true
	}
	return false
}

func checkBillingPeriod(billingUnit string, nRemovedVMs int, startBillingTime time.Time, startScalingAction time.Time) (bool) {
	switch billingUnit {
		case util.SECOND :
			return true
		case util.HOUR:
			deltaHours := startScalingAction.Sub(startBillingTime).Hours()
			delta := deltaHours - math.Floor(deltaHours)
			if delta == 0 || delta > 0.5 && nRemovedVMs > 0 {
				return true
			}
	}
	//TODO: change to false and modify how the billing is computed
	return true
}

func updateStartBillingTime(lastStartBillingTime time.Time, startScalingAction time.Time) (time.Time) {
	startBillingTime := lastStartBillingTime
	billedPeriod := startScalingAction.Sub(lastStartBillingTime).Hours()

	if billedPeriod >= 1{
		bp := int(math.Floor(billedPeriod))
		startBillingTime = startBillingTime.Add(time.Duration(bp)* time.Hour)
	}
	return startBillingTime
}


func updateEndBillingTime(startBillingTime time.Time, endScalingAction time.Time) (time.Time) {
	deltaScalingAction := endScalingAction.Sub(startBillingTime).Hours()
	ds := int(math.Ceil(deltaScalingAction))
	endBillingTime := startBillingTime.Add(time.Duration(ds)*time.Hour)
	return endBillingTime
}


/*
	Calculate base on the expected start time for the new state, when the launch should start
	in:
		@vmAdded types.VMScale
							- map of VM that were added
		@timeStart time.Time
							- time when the desired state should start
	out:
		@time.Time	- Time when the launch should start
*/
func computeScaleOutTransitionTime(vmAdded types.VMScale, podResize bool, timeStart time.Time, podsBootingTime float64) time.Time {
	transitionTime := timeStart
	//Time to boot new VMS
	nVMAdded := len(vmAdded)
	if nVMAdded > 0 {
		bootTimeVMAdded := computeVMBootingTime(vmAdded, systemConfiguration)
		transitionTime = timeStart.Add(-1 * time.Duration(bootTimeVMAdded) * time.Second)
		//Time for add new VMS into k8s cluster
		//60 seconds
		transitionTime = transitionTime.Add(-1 * time.Duration(60) * time.Second)
	}
	//Time to boot pods
	transitionTime = transitionTime.Add(-1 * time.Duration(podsBootingTime) * time.Second)
	return transitionTime
}


/*
	Try to find a containers configuration (num replicas, limit constraints) that fits in the current VM set and meet the load requirement
	in:
		@currentVMSet types.VMScale
							- map of VM of the current state
		@totalLoad float
							- load in terms of requests
		@mapVMProfiles map[string]types.VmProfile
							- map with the profiles of vms available
	out:
		@time.Time	- Time when the launch should start
*/
func findConfigOptionByContainerResize(currentVMSet types.VMScale, totalLoad float64, mapVMProfiles map[string]types.VmProfile) (types.ContainersConfig, bool){
	biggestType := biggestVMTypeInSet(currentVMSet, mapVMProfiles)
	biggestVM := mapVMProfiles[biggestType]
	allLimits,_ := storage.GetPerformanceProfileDAO(systemConfiguration.MainServiceName).FindAllUnderLimits(biggestVM.CPUCores, biggestVM.Memory)
	optionFound := false
	configurationOptionFound := types.ContainersConfig{}
	for _, li := range allLimits {
		configurationOption := selectProfileByLimits(totalLoad, li.Limit, false)
		replicas := configurationOption.MSCSetting.Replicas
		//Update the vm Profiles with the capacity for replicas with given limits
		computeVMsCapacity(li.Limit, &mapVMProfiles)
		currentConfigurationReplicasCapacity := currentVMSet.ReplicasCapacity(mapVMProfiles)
		if currentConfigurationReplicasCapacity >= replicas  && configurationOption.MSCSetting.MSCPerSecond >= totalLoad {
			configurationOptionFound = configurationOption
			optionFound = true
			break
		}
	}
	return configurationOptionFound, optionFound
}