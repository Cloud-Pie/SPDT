package derivation

import (
	"github.com/Cloud-Pie/SPDT/types"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
	"github.com/Cloud-Pie/SPDT/planner/execution"
	"math"
	"sort"
	"github.com/Cloud-Pie/SPDT/rest_clients/performance_profiles"
	"github.com/op/go-logging"
	"github.com/Cloud-Pie/SPDT/storage"
	"errors"
	"github.com/cnf/structhash"
	"strings"
	"fmt"
	"github.com/Cloud-Pie/SPDT/planner/forecast_processing"
)

var log = logging.MustGetLogger("spdt")
var systemConfiguration util.SystemConfiguration
var initialState types.State

//Interface for strategies of how to scale
type PolicyDerivation interface {
	CreatePolicies (processedForecast types.ProcessedForecast) []types.Policy
	FindSuitableVMs (numberReplicas int, limits types.Limit) types.VMScale
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
func Policies(sortedVMProfiles []types.VmProfile, sysConfiguration util.SystemConfiguration, forecast types.Forecast) ([]types.Policy, error) {
	var policies []types.Policy
	systemConfiguration = sysConfiguration
	mapVMProfiles := VMListToMap(sortedVMProfiles)

	log.Info("Request current state" )
	currentState,err := execution.RetrieveCurrentState(sysConfiguration.SchedulerComponent.Endpoint + util.ENDPOINT_CURRENT_STATE)

	if err != nil {
		log.Error("Error to get current state %s", err.Error() )
	} else {
		log.Info("Finish request for current state" )
	}
	if currentState.Services[systemConfiguration.MainServiceName].Scale == 0 {
		return policies, errors.New("Service "+ systemConfiguration.MainServiceName +" is not deployed")
	}
	if available, vmType := validateVMProfilesAvailable(currentState.VMs, mapVMProfiles); !available{
		return policies, errors.New("Information not available for VM Type "+vmType )
	}

	granularity := systemConfiguration.ForecastComponent.Granularity
	processedForecast := forecast_processing.ScalingIntervals(forecast, granularity)
	initialState = currentState


	switch sysConfiguration.PreferredAlgorithm {
	case util.NAIVE_ALGORITHM:
		naive := NaivePolicy {algorithm:util.NAIVE_ALGORITHM,
							 currentState:currentState, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies = naive.CreatePolicies(processedForecast)

	case util.BEST_RESOURCE_PAIR_ALGORITHM:
		base := BestResourcePairPolicy{algorithm:util.BEST_RESOURCE_PAIR_ALGORITHM,
			sortedVMProfiles:sortedVMProfiles,currentState:currentState,mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies = base.CreatePolicies(processedForecast)

	case util.ALWAYS_RESIZE_ALGORITHM:
		alwaysResize := AlwaysResizePolicy{algorithm:util.ALWAYS_RESIZE_ALGORITHM,
									mapVMProfiles:mapVMProfiles, sortedVMProfiles:sortedVMProfiles, sysConfiguration: sysConfiguration, currentState:currentState}
		policies = alwaysResize.CreatePolicies(processedForecast)

	case util.ONLY_DELTA_ALGORITHM:
		tree := DeltaLoadPolicy{algorithm:util.ONLY_DELTA_ALGORITHM, currentState:currentState,
			mapVMProfiles:mapVMProfiles,sysConfiguration: sysConfiguration}
		policies = tree.CreatePolicies(processedForecast)

	case util.RESIZE_WHEN_BENEFICIAL:
		algorithm := ResizeWhenBeneficialPolicy{algorithm:util.RESIZE_WHEN_BENEFICIAL, currentState:currentState,
		sortedVMProfiles:sortedVMProfiles, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies = algorithm.CreatePolicies(processedForecast)
	default:
		//naive
		naive := NaivePolicy {algorithm:util.NAIVE_ALGORITHM,
			currentState:currentState, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies1 := naive.CreatePolicies(processedForecast)
		policies = append(policies, policies1...)
		//types
		base := BestResourcePairPolicy{algorithm:util.BEST_RESOURCE_PAIR_ALGORITHM,
			currentState:currentState, sortedVMProfiles:sortedVMProfiles, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies2 := base.CreatePolicies(processedForecast)
		policies = append(policies, policies2...)
		//always resize
		alwaysResize := AlwaysResizePolicy{algorithm:util.ALWAYS_RESIZE_ALGORITHM,
			mapVMProfiles:mapVMProfiles, sortedVMProfiles:sortedVMProfiles, sysConfiguration: sysConfiguration, currentState:currentState}
		policies3 := alwaysResize.CreatePolicies(processedForecast)
		policies = append(policies, policies3...)
		//resize when beneficial
		algorithm := ResizeWhenBeneficialPolicy{algorithm:util.RESIZE_WHEN_BENEFICIAL, currentState:currentState,
			sortedVMProfiles:sortedVMProfiles, mapVMProfiles:mapVMProfiles, sysConfiguration: sysConfiguration}
		policies4 := algorithm.CreatePolicies(processedForecast)
		policies = append(policies, policies4...)

		//delta load
		tree := DeltaLoadPolicy{algorithm:util.ONLY_DELTA_ALGORITHM, currentState:currentState,
			mapVMProfiles:mapVMProfiles,sysConfiguration: sysConfiguration}
		policies6 := tree.CreatePolicies(processedForecast)
		policies = append(policies, policies6...)
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
func computeVMBootingTime(vmsScale types.VMScale, sysConfiguration util.SystemConfiguration) float64 {
	bootTime := 0.0
	//Check in db if already data is stored
	vmBootingProfileDAO := storage.GetVMBootingProfileDAO()

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
func computeVMTerminationTime(vmsScale types.VMScale, sysConfiguration util.SystemConfiguration) float64 {
	terminationTime := 0.0
	//Check in db if already data is stored
	vmBootingProfileDAO := storage.GetVMBootingProfileDAO()

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

/* Compute the max number of pod replicas (Replicas capacity) that a VM can host
	in:
		@vmProfile types.VmProfile
		@resourceLimit types.Limits
	out:
		@int	Number of replicas
*/
func maxPodsCapacityInVM(vmProfile types.VmProfile, resourceLimit types.Limit) int {
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
	out:
		@ContainersConfig	- configuration with number of replicas and limits that best fit for the number of requests
*/
func estimatePodsConfiguration(requests float64, limits types.Limit) (types.ContainersConfig, error){
	var containerConfig types.ContainersConfig
	var err error
	serviceProfileDAO := storage.GetPerformanceProfileDAO(systemConfiguration.MainServiceName)

	performanceProfileBase,_ := serviceProfileDAO.FindByLimitsAndReplicas(limits.CPUCores, limits.MemoryGB, 1)
	estimatedReplicas := int(math.Ceil(requests / performanceProfileBase.MSCSettings[0].MSCPerSecond))
	performanceProfileCandidate,err1 := serviceProfileDAO.FindByLimitsAndReplicas(limits.CPUCores, limits.MemoryGB, estimatedReplicas)

	if err1 == nil && performanceProfileCandidate.MSCSettings[0].MSCPerSecond >= requests {
		containerConfig.MSCSetting.Replicas = performanceProfileCandidate.MSCSettings[0].Replicas
		containerConfig.MSCSetting.MSCPerSecond = performanceProfileCandidate.MSCSettings[0].MSCPerSecond
		containerConfig.Limits = limits
	} else {
		url := systemConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_SERVICE_PROFILE_BY_MSC
		appName := systemConfiguration.AppName
		appType := systemConfiguration.AppType
		mainServiceName := systemConfiguration.MainServiceName
		mscSetting,err := performance_profiles.GetPredictedReplicas(url,appName,appType,mainServiceName,requests,limits.CPUCores, limits.MemoryGB)

		newMSCSetting := types.MSCSimpleSetting{}
		if err == nil {
			containerConfig.MSCSetting.Replicas = mscSetting.Replicas
			containerConfig.MSCSetting.MSCPerSecond = mscSetting.MSCPerSecond.RegBruteForce
			containerConfig.Limits = limits

			newMSCSetting.Replicas = mscSetting.Replicas
			newMSCSetting.MSCPerSecond = mscSetting.MSCPerSecond.RegBruteForce
			if mscSetting.BootTimeMs > 0 {
				newMSCSetting.BootTimeSec = util.MillisecondsToSeconds(mscSetting.BootTimeMs)
			} else {
				newMSCSetting.BootTimeSec = util.DEFAULT_POD_BOOT_TIME
			}
			profile,err3 := serviceProfileDAO.FindByLimitsAndReplicas(limits.CPUCores, limits.MemoryGB, mscSetting.Replicas)
			if err3 != nil {
				profile,_= serviceProfileDAO.FindProfileByLimits(limits)
				profile.MSCSettings = append(profile.MSCSettings,newMSCSetting)
				err3 = serviceProfileDAO.UpdateById(profile.ID, profile)
				if err3 != nil{
					log.Error("Performance profile not updated")
				}
			}
		} else {
			return containerConfig, err
		}
	}

	return containerConfig, err
}

/* Select the service profile for any limit resources that satisfies the number of requests
	in:
		@requests	float64 - number of requests that the service should serve
	out:
		@ContainersConfig	- configuration with number of replicas and limits that best fit for the number of requests
*/
func selectProfileUnderVMLimits(requests float64,  limits types.Limit) (types.ContainersConfig, error) {
	var profiles []types.ContainersConfig
	var profile  types.ContainersConfig
	serviceProfileDAO := storage.GetPerformanceProfileDAO(systemConfiguration.MainServiceName)
	profiles,err2 := serviceProfileDAO.MatchProfileFitLimitsOver(limits.CPUCores, limits.MemoryGB, requests)

    if err2 == nil{
		sort.Slice(profiles, func(i, j int) bool {
			utilizationFactori := float64(profiles[i].MSCSetting.Replicas) * profiles[i].Limits.CPUCores +  float64(profiles[i].MSCSetting.Replicas) * profiles[i].Limits.MemoryGB
			utilizationFactorj := float64(profiles[j].MSCSetting.Replicas) * profiles[j].Limits.CPUCores + float64(profiles[j].MSCSetting.Replicas) * profiles[j].Limits.MemoryGB
			msci := profiles[i].MSCSetting.MSCPerSecond
			mscj := profiles[j].MSCSetting.MSCPerSecond
			if msci == mscj {
				if utilizationFactori != utilizationFactorj {
					return utilizationFactori < utilizationFactorj
				} else {
					return 	profiles[i].MSCSetting.Replicas < profiles[j].MSCSetting.Replicas
				}
			}
			return   math.Abs(requests - msci) <  math.Abs(requests - mscj)
		})
	}
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
func getStateLoadCapacity(numberReplicas int, limits types.Limit) types.MSCSimpleSetting {
	serviceProfileDAO := storage.GetPerformanceProfileDAO(systemConfiguration.MainServiceName)
	profile,_ := serviceProfileDAO.FindByLimitsAndReplicas(limits.CPUCores, limits.MemoryGB, numberReplicas)
	newMSCSetting := types.MSCSimpleSetting{}
	if len(profile.MSCSettings) > 0 {
		return profile.MSCSettings[0]
	}else {
		url := systemConfiguration.PerformanceProfilesComponent.Endpoint + util.ENDPOINT_SERVICE_PROFILE_BY_REPLICAS
		appName := systemConfiguration.AppName
		appType := systemConfiguration.AppType
		mainServiceName := systemConfiguration.MainServiceName
		mscCompleteSetting,_ := performance_profiles.GetPredictedMSCByReplicas(url,appName,appType,mainServiceName,numberReplicas,limits.CPUCores, limits.MemoryGB)
		newMSCSetting = types.MSCSimpleSetting{
			MSCPerSecond:mscCompleteSetting.MSCPerSecond.RegBruteForce,
			BootTimeSec:mscCompleteSetting.BootTimeMs,
			Replicas:mscCompleteSetting.Replicas,
			StandDevBootTimeSec:mscCompleteSetting.StandDevBootTimeMS/1000,
		}
		if newMSCSetting.BootTimeSec == 0 {
			newMSCSetting.BootTimeSec = util.DEFAULT_POD_BOOT_TIME
		}
		//update in db

		profile,_:= serviceProfileDAO.FindByLimitsAndReplicas(limits.CPUCores, limits.MemoryGB, numberReplicas)
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
	return newMSCSetting
}

/* Utility method to set up each scaling configuration
*/
func setScalingSteps(scalingSteps *[]types.ScalingAction, currentState types.State,newState types.State, timeStart time.Time, timeEnd time.Time, totalServicesBootingTime float64, stateLoadCapacity float64) {
	nScalingSteps := len(*scalingSteps)
	if nScalingSteps >= 1 && newState.Equal((*scalingSteps)[nScalingSteps-1].DesiredState) {
		(*scalingSteps)[nScalingSteps-1].TimeEnd = timeEnd
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

		//newState.LaunchTime = startTransitionTime
		name,_ := structhash.Hash(newState, 1)
		newState.Hash = strings.Replace(name, "v1_", "", -1)
		*scalingSteps = append(*scalingSteps,
			types.ScalingAction{
				InitialState:currentState,
				DesiredState:        newState,
				TimeStart:           timeStart,
				TimeEnd:             timeEnd,
				Metrics:             types.ConfigMetrics{RequestsCapacity:stateLoadCapacity,},
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
	fmt.Println("lenght tree")
	fmt.Println(len(candidateVMSets))
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
		replicasCapacity :=  maxPodsCapacityInVM(v, limits)
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
				return candidateVMSets[i].TotalVMs() < candidateVMSets[j].TotalVMs()
			}
			return false
		})
		return candidateVMSets[0],err
	}else {
		return types.VMScale{},errors.New("No VM Candidate")
	}
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
		//Case 1: New VMs
		bootTimeVMAdded := computeVMBootingTime(vmAdded, systemConfiguration)
		transitionTime = timeStart.Add(-1 * time.Duration(bootTimeVMAdded) * time.Second)
		//Time for add new VMS into k8s cluster
		transitionTime = transitionTime.Add(-1 * time.Duration(util.TIME_ADD_NODE_TO_K8S) * time.Second)
		//Time to boot pods assuming worst scenario, when image has to be pulled
		transitionTime = transitionTime.Add(-1 * time.Duration(podsBootingTime) * time.Second)
	} else {
		//Case: Only replication of pods
		transitionTime = transitionTime.Add(-1 * time.Duration(util.TIME_CONTAINER_START) * time.Second)
	}
	return transitionTime
}

func validateVMProfilesAvailable(vmSet types.VMScale, mapVMProfiles map[string]types.VmProfile ) (bool, string) {
	for k,_ := range vmSet {
		if _,ok := mapVMProfiles[k]; !ok {
			return false, k
		}
	}
	return true,""
}

func adjustGranularity(granularity string, capacityInSeconds float64) float64{
	var factor float64

	switch granularity {
	case util.HOUR:
		factor = 3600
	case util.MINUTE:
		factor = 60
	case util.SECOND:
		factor = 1
	default:
		factor = 3600
	}
	return factor * capacityInSeconds
}