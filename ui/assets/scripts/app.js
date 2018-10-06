var policiesEndpoint = ''
var forecastRequestsEndpoint = ''

var allPolicies = []
var requestDemand = []
var timeRequestDemand = []

function computeUnits(data){
    time = [];
    vms = [];
    replicas = [];
    MSC = [];
    utilizationCpuCores = [];
    utilizationMemGB = [];
    limitsCpuCores = [];
    limitsMemGB = [];
    labelsTypesVmSet = [];
    arrayScalingActions = data.scaling_actions
    vmTypesList = data.parameters["vm-types"].split(",");
    maxNumVMs = 0
    maxNumContainers = 0
    vmScalesInTime = {}
    accumulatedCost = []
    vmTypesList.forEach(function (vmType) {
        vmScalesInTime[vmType] = []
    })

    arrayScalingActions.forEach(function (conf) {
        time.push(conf.time_start)
        let vmLabels = ""
        let totalVMS = 0

        //Flags map to identify if a record for that vm type was already inserted
        vmScalesInTimeFlags = {}
        vmTypesList.forEach(function (vmType) {
            vmScalesInTimeFlags[vmType] = false
        })
        /*Virtual Machines*/
        vmSet = conf.desired_state.VMs
        for (var key in vmSet) {
            vmLabels = vmLabels + key + ":" + vmSet[key] + ", "
            totalVMS = totalVMS + vmSet[key]
            vmScalesInTime[key].push(vmSet[key])
            vmScalesInTimeFlags[key] = true
        }
        for (var type in vmScalesInTimeFlags) {
            if (vmScalesInTimeFlags[type] == false) {
                vmScalesInTime[type].push(0)
            }
        }
        vms.push(totalVMS)
        if (totalVMS > maxNumVMs){
            maxNumVMs = totalVMS
        }
        labelsTypesVmSet.push(vmLabels)

        /*Services*/
        services = conf.desired_state.Services
        for (var key in services) {
            replicas.push(services[key].Replicas)
            if (services[key].Replicas > maxNumContainers) {
                maxNumContainers = services[key].Replicas
            }
            limitsCpuCores.push(services[key].Cpu_cores)
            limitsMemGB.push(services[key].Mem_gb)
        }
        MSC.push(conf.metrics.requests_capacity)
        utilizationCpuCores.push(conf.metrics.cpu_utilization)
        utilizationMemGB.push(conf.metrics.mem_utilization)
        lenghtAccumulatedCost = accumulatedCost.length
        if (accumulatedCost.length == 0) {
            accumulatedCost.push(conf.metrics.cost)
        }else {
            cost = accumulatedCost[accumulatedCost.length-1] + conf.metrics.cost
            accumulatedCost.push(cost)
        }

    })

    //Needed to include the last time t into the plot
    lastConf = arrayScalingActions[arrayScalingActions.length - 1]
    time.push(lastConf.time_end)
    vmSet = lastConf.desired_state.VMs
    let vmLabels = ""
    let totalVMS = 0
    //Flags map to identify if a record for that type was already inserted
    vmScalesInTimeFlags = {}
    vmTypesList.forEach(function (vmType) {
        vmScalesInTimeFlags[vmType] = false
    })

    for (var key in vmSet) {
        vmLabels = vmLabels + key + ":" + vmSet[key] + ", "
        totalVMS = totalVMS + vmSet[key]
        vmScalesInTime[key].push(vmSet[key])
        vmScalesInTimeFlags[key] = true
    }
    for (var type in vmScalesInTimeFlags) {
        if (vmScalesInTimeFlags[type] == false) {
            vmScalesInTime[type].push(0)
        }
    }
    vms.push(totalVMS)
    if (totalVMS > maxNumVMs){
        maxNumVMs = totalVMS
    }
    labelsTypesVmSet.push(vmLabels)
    services = lastConf.desired_state.Services
    for (var key in services) {
        replicas.push(services[key].Replicas)
        if (services[key].Replicas > maxNumContainers) {
            maxNumContainers = services[key].Replicas
        }
        limitsCpuCores.push(services[key].Cpu_cores)
        limitsMemGB.push(services[key].Mem_gb)
    }
    MSC.push(lastConf.metrics.requests_capacity)
    utilizationCpuCores.push(lastConf.metrics.cpu_utilization)
    utilizationMemGB.push(lastConf.metrics.mem_utilization)
    //End last t

   return {
        time: time,
        vms: vms,
        replicas: replicas,
        msc: MSC,
        utilizationCpuCores: utilizationCpuCores,
        utilizationMemGB: utilizationMemGB,
        labelsTypesVmSet: labelsTypesVmSet,
        vmScalesInTime: vmScalesInTime,
        limitsCpuCores: limitsCpuCores,
        limitsMemGB:limitsMemGB,
        maxNumContainers: maxNumContainers,
        maxNumVMs: maxNumVMs,
        accumulatedCost: accumulatedCost
    }
}

function plotVMUnitsPerType(time, vms, timeRequests, textHover, maxNumberVMs) {
   let data = [];
   for (var key in vms) {
      data.push(
           {
               x: time,
               y: vms[key],
               name: key,
               type: 'scatter',
               line: {shape: 'hv'},
               fill: 'tonexty'
           }
       )

   }
   l = data.length
   data[l-1].text = textHover
   data.push(
        {
            x: timeRequests,
            y: requestDemand,
            name: 'Demand',
            type: 'scatter',
            line: {shape: 'spline', color:'#092e20'},
            yaxis: 'y2',
            visible: 'legendonly'
        }
    )

   var layout = {
        title: '<b>N° Virtual Machines</b>',
        titlefont: {
           size:18, color: '#092e20'
        },
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        yaxis: {title: 'N° VMs', range: [0, maxNumberVMs]},
        yaxis2: {
           title: 'Requests/Hour',
           titlefont: {color: '#092e20'},
           tickfont: {color: '#092e20'},
           overlaying: 'y',
           side: 'right'
        },
        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.088,
            x: 0.2
        },
    };

   Plotly.newPlot('vmUnits', stackedArea(data),layout);
}

function stackedArea(traces) {
    for(var i=1; i<traces.length; i++) {
        for(var j=0; j<(Math.min(traces[i]['y'].length, traces[i-1]['y'].length)); j++) {
            traces[i]['y'][j] += traces[i-1]['y'][j];
        }
    }
    return traces;
}

function plotVMUnits(time, vms, textHover) {
    var trace1 = {
        x: time,
        y: vms,
        type: 'scatter',
        name: 'N° VMs',
        text: textHover,
        line: {shape: 'hv'}
    };

    var layout = {
        title: 'N° Virtual Machines',
        titlefont: {
           size:18
        },
        autosize:true,
        //margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',

        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.09,
            x: 0.9
        },
    };
    var data = [trace1];
    Plotly.newPlot('vmUnits', data,layout);
}

function plotContainerUnits(time, replicas, cpuCores, memGB, maxNumContainers) {
    var trace1 = {
        x: time,
        y: replicas,
        type: 'scatter',
        name: 'N° Replicas',
        line: {shape: 'hv'},
        fill: 'tonexty'
    };

    var trace2 = {
        x: time,
        y: cpuCores,
        type: 'scatter',
        name: 'CPU cores',
        yaxis: 'y2',
        line: {shape: 'hv'}
    };

    var trace3 = {
        x: time,
        y: memGB,
        type: 'scatter',
        name: 'Mem GB',
        yaxis: 'y2',
        line: {shape: 'hv'}
    };

    var layout = {
        title: '<b>N° Containers</b>',
        titlefont: {
           size:18, color: '#092e20'
        },
        yaxis: {title: 'N° Containers', range: [0, maxNumContainers]},
        yaxis2: {
            title: 'Resources',
            titlefont: {color: 'rgb(148, 103, 189)'},
            tickfont: {color: 'rgb(148, 103, 189)'},
            overlaying: 'y',
            side: 'right'
        },
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',

        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.088,
            x: 0.2
        },
    };
    var data = [trace1, trace2, trace3];
    Plotly.newPlot('containerUnits', data,layout);
}

function plotCapacity(time, demand, supply, timeSuply){
    var trace1 = {
        x: time,
        y: demand,
        name: 'Demand',
        type: 'scatter',
        line: {shape: 'spline', color:'#092e20'}
    };

    var trace2 = {
        x: timeSuply,
        y: supply,
        name: 'Supply',
        type: 'scatter',
        line: {shape: 'hv'}
    };

    var layout = {
        title: '<b>Workload</b>',
        titlefont: {
           size:18, color: '#092e20'
        },
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',

        yaxis: {title: 'Requests/Sec'},
        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.088,
            x: 0.2
        },
    };
    var data = [trace1, trace2];
    Plotly.newPlot('requestsUnits', data,layout);
}
function plotAccumulatedCost(time, cost) {
    var trace = {
        x: time,
        y: cost,
        type: 'scatter',
        name: '$'
    };

    var layout = {
        title: '<b>Accumulated Cost</b>',
        titlefont: {
            size:18, color: '#092e20'
        },
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',

        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.088,
            x: 0.2
        },
    };
    var data = [trace];
    Plotly.newPlot('accumulatedCost', data,layout);
}

function plotProvidedResources(time, cpuCores, memGB) {

    var trace = {
        x: time,
        y: cpuCores,
        type: 'scatter',
        name: '% CPU Cores'
    };

    var trace2 = {
        x: time,
        y: memGB,
        type: 'scatter',
        name: '% Mem GB'
    };

    var layout = {
        title: '<b>% Resource Allocation</b>',
        titlefont: {
           size:18, color: '#092e20'
        },
        yaxis: {range: [0, 100]},
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',

        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.088,
            x: 0.2
        },
    };
    var data = [trace, trace2];
    Plotly.newPlot('resources', data,layout);
}

function searchByID(policyId) {
    appName = document.getElementById("appNameid").value;
    policiesEndpoint = '/api/'+appName+'/policies'
    requestURL = policiesEndpoint + "/" + policyId

    if (policyId == null) {
        policyId = document.getElementById("searchpolicyid").value;
    }

    fetch(requestURL)
        .then((response) => response.json())
        .then(function (policy){
            var timeStart = new Date(policy.window_time_start).toISOString();
            var timeEnd = new Date( policy.window_time_end).toISOString();
            //fetchLoadPredicted(timeStart,timeEnd)
            displayPolicyInformation(policy)
        })
        .catch(function(err) {
            console.log('Fetch Error :-S', err);
            showNoResultsPanel()
        });
}

function displayPolicyInformation(policy) {
    showSinglePolicyPanels()
    var timeStart = new Date(policy.window_time_start).toISOString();
    var timeEnd = new Date( policy.window_time_end).toISOString();
    var units = computeUnits(policy)
    document.getElementById("jsonId").innerText = JSON.stringify(policy,undefined, 5);

    //if (requestDemand.length == 0) {
        fetchLoadPredicted(timeStart,timeEnd)
   // }
    plotCapacity(timeRequestDemand, requestDemand, units.msc, units.time)
    plotVMUnitsPerType(units.time, units.vmScalesInTime,timeRequestDemand, units.labelsTypesVmSet, units.maxNumVMs)
    plotContainerUnits(units.time, units.replicas, units.limitsCpuCores, units.limitsMemGB, units.maxNumContainers)
    plotProvidedResources(units.time, units.utilizationCpuCores, units.utilizationMemGB)
    plotAccumulatedCost(units.time, units.accumulatedCost)
    fillMetrics(policy)
    fillDetailsTable(policy)
}

function fetchLoadPredicted(timeStart, timeEnd){
    let params = {
        "start": timeStart,
        "end": timeEnd
    }
    let esc = encodeURIComponent
    let query = Object.keys(params)
        .map(k => esc(k) + '=' + esc(params[k]))
        .join('&')
    appName = document.getElementById("appNameid").value;
    forecastRequestsEndpoint = '/api/'+appName +'/forecast?' + query
    requestURL = forecastRequestsEndpoint
    fetch(requestURL)
        .then((response) => response.json())
        .then(function (data){
            requestDemand = data.Requests
            timeRequestDemand = data.Timestamp
        }).catch(function(err) {
        console.log('Fetch Error :-S', err);
        showNoResultsPanel()
    });
}

function searchByTimestamp() {
    var timeStart = new Date(document.getElementById("datetimestart").value).toISOString();
    var timeEnd = new Date(document.getElementById("datetimeend").value).toISOString();
    let params = {
        "start": timeStart,
        "end": timeEnd
    }
    let esc = encodeURIComponent
    let query = Object.keys(params)
        .map(k => esc(k) + '=' + esc(params[k]))
        .join('&')

    appName = document.getElementById("appNameid").value;
    policiesEndpoint = '/api/'+appName+'/policies'
    policiesRequest = policiesEndpoint +"?" + query
    fetch(policiesRequest)
        .then((response) => response.json())
        .then(function (data){
            allPolicies = data
            if(allPolicies.length > 0) {
                showSinglePolicyPanels()
                fillCandidateTable(data)
                displayPolicyInformation(allPolicies[0])
            }else{
                showNoResultsPanel()
            }
        })
        .catch(function(err) {
            console.log('Fetch Error :-S', err);
            showNoResultsPanel()
        });
}

function fillCandidateTable(policyCandidates) {
    $("#tBodyCandidates").children().remove()
    for(var i = 0; i < policyCandidates.length; i++) {

        label = "label-warning"
        if (policyCandidates[i].status == "selected") {
            label = "label-success"
        }

        $("#tCandidates > tbody").append("<tr>" +
            "<td width=\"50%\">"+policyCandidates[i].id+"</td>" +
            "<td>"+policyCandidates[i].algorithm+"</td>" +
            "<td>"+policyCandidates[i].metrics.cost+"</td>" +
            "<td> <span class='label "+ label+" '>" +policyCandidates[i].status+ "</span> </td>" +
            "</tr>");
    }

    $('#tCandidates > tbody > tr').click(function () {
      $('#tCandidates > tbody > tr').removeClass("selected");
      $(this).addClass("selected");
       var id = $(this).find('td:first').text()
       searchByID(id)
    });
}

function fillMetrics(policy){
    document.getElementById("costid").innerText = policy.metrics.cost;
    document.getElementById("overid").innerText = policy.metrics.over_provision;
    document.getElementById("underid").innerText = policy.metrics.under_provision;
    document.getElementById("durationId").innerText = policy.metrics.derivation_duration;
    document.getElementById("reconfVMsid").innerText = policy.metrics.num_scale_vms;
    document.getElementById("reconfContid").innerText = policy.metrics.num_scale_containers;
    document.getElementById("shadowTimeId").innerText = policy.metrics.avg_shadow_time_sec;
    document.getElementById("transitionTimeId").innerText = policy.metrics.avg_transition_time_sec;
    document.getElementById("timeBetweenStatesId").innerText = policy.metrics.avg_time_between_scaling_sec;
}

function fillDetailsTable(policy) {
    $("#tBodyDetails").children().remove()
    parameters = policy.parameters

    $("#tDetails > tbody").append("<tr>" +
        "<td><b>"+"Policy ID"+"</b></td>" +
        "<td><b>"+policy.id+"</b></td>" +
        "</tr>");

    $("#tDetails > tbody").append("<tr>" +
        "<td><b>"+"Algorithm"+"</b></td>" +
        "<td>"+policy.algorithm+"</td>" +
        "</tr>");

    $("#tDetails > tbody").append("<tr>" +
        "<td><b>"+"Duration of derivation"+"</b></td>" +
        "<td>"+policy.metrics.derivation_duration+"</td>" +
        "</tr>");


    let timewindow = new Date( policy.window_time_start).toLocaleString() + " - " + new Date( policy.window_time_end).toLocaleString() ;

    $("#tDetails > tbody").append("<tr>" +
        "<td><b>"+"Time Window"+"</b></td>" +
        "<td>"+timewindow+"</td>" +
        "</tr>");

    for(var key in parameters) {
        $("#tDetails > tbody").append("<tr>" +
            "<td><b>"+key+"<b></td>" +
            "<td>"+parameters[key]+"</td>" +
            "</tr>");
    }
}

function clickedCompareAll(){
    showMultiplePolicyPanels()

    units = getVirtualUnitsAll(allPolicies)
    plotCapacityAll(units.time,requestDemand, units.trnAll, units.tracesAll)
   // plotVMsAll(units.time, units.vmsAll, units.tracesAll)
    plotReplicasAll(units.time, units.replicasAll,units.tracesAll)
   // plotCPUAll(units.time, units.cpuCoresAll, units.tracesAll)
   // plotMemAll(units.time, units.memGBAll, units.tracesAll)
    plotNScalingVMAll(units.nScalingVMsAll, units.nScalingContainersAll,units.tracesAll)
    plotOverUnderProvisionAll(units.overprovisionAll,units.underprovisionAll, units.tracesAll)
    plotCostAll(units.costAll,units.tracesAll)
    plotAccumulatedCostAll(units.time, units.accumulatedCostAll, units.tracesAll)
    //plotDerivationTimeAll(units.derivationDurationTimeAll, units.tracesAll)

    //plotAvgShadowTimeAll(units.avgShadowTimeAll, units.tracesAll)
    //plotTransitionTimeAll(units.avgTransitionTimeAll, units.tracesAll)
    plotBarComparisonAll(units.derivationDurationTimeAll, units.tracesAll, 'Derivation Time (s)', 'Derivation Time', 'derivationTimeAll')
    plotBarComparisonAll(units.avgShadowTimeAll, units.tracesAll, 'Avg Shadow Time (s)', 'Avg Shadow Time', 'avgShadowTimeAll')
    plotBarComparisonAll(units.avgTransitionTimeAll, units.tracesAll, 'Avg Transition Time (s)', 'Avg Transition Time', 'avgTransitionTimeAll')
}

function getVirtualUnitsAll(policies) {
    vmsAll = [];
    replicasAll = [];
    TRNAll = [];
    cpuCoresAll = [];
    memGBAll = [];
    tracesAll = [];
    overprovisionAll = [];
    underprovisionAll = [];
    nScalingVMsAll = [];
    nScalingContainersAll = [];
    costAll = [];
    accumulatedCostAll = [];
    avgShadowTimeAll = [];
    avgTransitionTimeAll = [];
    derivationDurationTimeAll = [];
    policies.forEach(function (policy) {
        time = [];
        vms = [];
        replicas = [];
        MSC = [];
        utilizationCpuCores = [];
        utilizationMemGB = [];
        accumulatedCost = [];
        tracesAll.push(policy.algorithm)
        arrayScalingActions = policy.scaling_actions
        arrayScalingActions.forEach(function (conf) {
            time.push(conf.time_start)

            vmSet = conf.desired_state.VMs
            for (var key in vmSet) {
                vms.push(vmSet[key])
            }
            services = conf.desired_state.Services
            for (var key in services) {
                replicas.push(services[key].Replicas)
                utilizationCpuCores.push(services[key].Cpu_cores * services[key].Replicas)
                utilizationMemGB.push(services[key].Mem_gb * services[key].Replicas)
            }
            MSC.push(conf.metrics.requests_capacity)

            lenghtAccumulatedCost = accumulatedCost.length
            if (accumulatedCost.length == 0) {
                accumulatedCost.push(conf.metrics.cost)
            }else {
                cost = accumulatedCost[accumulatedCost.length-1] + conf.metrics.cost
                accumulatedCost.push(cost)
            }
        })
        vmsAll.push(vms)
        replicasAll.push(replicas)
        TRNAll.push(MSC)
        cpuCoresAll.push(utilizationCpuCores)
        memGBAll.push(utilizationMemGB)
        overprovisionAll.push(policy.metrics.over_provision)
        underprovisionAll.push(policy.metrics.under_provision)
        nScalingVMsAll.push(policy.metrics.num_scale_vms)
        nScalingContainersAll.push(policy.metrics.num_scale_containers)
        costAll.push(policy.metrics.cost)
        accumulatedCostAll.push(accumulatedCost)
        avgShadowTimeAll.push(policy.metrics.avg_shadow_time_sec)
        avgTransitionTimeAll.push(policy.metrics.avg_transition_time_sec)
        derivationDurationTimeAll.push(policy.metrics.derivation_duration)
    })

    return {
        time: time,
        tracesAll: tracesAll,
        vmsAll: vmsAll,
        replicasAll: replicasAll,
        trnAll: TRNAll,
        cpuCoresAll: cpuCoresAll,
        memGBAll: memGBAll,
        overprovisionAll: overprovisionAll,
        underprovisionAll: underprovisionAll,
        nScalingVMsAll: nScalingVMsAll,
        nScalingContainersAll:nScalingContainersAll,
        costAll: costAll,
        accumulatedCostAll:accumulatedCostAll,
        avgShadowTimeAll:avgShadowTimeAll,
        avgTransitionTimeAll:avgTransitionTimeAll,
        derivationDurationTimeAll:derivationDurationTimeAll
    }
}

function plotCapacityAll(time, demand, supplyAll,tracesAll){

    var data = [];
    data.push(
        {
            x: timeRequestDemand,
            y: demand,
            name: 'Demand',
            type: 'scatter',
            line: {shape: 'spline'}
        }
    )
    var i = 0
    supplyAll.forEach(function (item) {
        {
            data.push(
                {
                    x: time,
                    y: item,
                    name: tracesAll[i],
                    type: 'scatter',
                    line: {shape: 'hv'}
                }
            )
            i=i+1
        }
    })

    var layout = {
        title: 'Workload',
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',

    };

    Plotly.newPlot('requestsUnitsAll', data,layout);
}
/*
function plotMemAll(time, memGBAll, tracesAll) {

    var data = [];
    var i = 0
    memGBAll.forEach(function (item) {
        {
            data.push(
                {
                    x: time,
                    y: item,
                    type: 'scatter',
                    name: tracesAll[i],
                    line: {shape: 'hv'}

                }
            )
            i=i+1
        }
    })


    var layout = {
        title: 'Memory provisioned',
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',

    };

    Plotly.newPlot('memoryUtilizationAll', data,layout);
}

function plotCPUAll(time, cpuCoresAll, tracesAll) {

    var data = [];
    var i = 0;
    cpuCoresAll.forEach(function (item) {
        {
            data.push(
                {
                    x: time,
                    y: item,
                    type: 'scatter',
                    name: tracesAll[i],
                    line: {shape: 'hv'}

                }
            )
            i=i+1
        }
    })


    var layout = {
        title: 'CPU cores provisioned',
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        //height: 200
    };

    Plotly.newPlot('cpuUtilizationAll', data,layout);
}
/*
function plotVMsAll(time, vmsAll, tracesAll) {

    var data = [];
    var i = 0;
    vmsAll.forEach(function (item) {
        {
            data.push(
                {
                    x: time,
                    y: item,
                    type: 'scatter',
                    name: tracesAll[i],
                    line: {shape: 'hv'}

                }
            )
            i=i+1
        }
    })


    var layout = {
        title: 'N° VMs',
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        //height: 200
    };

    Plotly.newPlot('vmUnitsAll', data,layout);
}*/

function plotReplicasAll(time, replicasAll, tracesAll) {
    var data = [];
    var i = 0;
    replicasAll.forEach(function (item) {
        {
            data.push(
                {
                    x: time,
                    y: item,
                    type: 'scatter',
                    name: tracesAll[i],
                    line: {shape: 'hv'}

                }
            )
            i=i+1
        }
    })

    var layout = {
        title: 'N° Replicas',
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        //height: 200
    };

    Plotly.newPlot('replicaUnitsAll', data,layout);
}

function plotNScalingVMAll(nScalingVMs, nScalingContainers, tracesAll) {
    var trace1 = {
        x: tracesAll,
        y: nScalingVMs,
        type: 'bar',
        name: 'N° VM Scaling actions ',
    };

    var trace2 = {
        x: tracesAll,
        y: nScalingContainers,
        type: 'bar',
        name: 'N° Container Scaling actions ',
    };

    var data = [trace1, trace2];
    var layout = {
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.088,
            x: 0.2
        },
    };

    Plotly.newPlot('nScalingVmsAll', data,layout);
}

function plotOverUnderProvisionAll(overAll, underAll, tracesAll) {

    var trace1 = {
        x: tracesAll,
        y: overAll,
        type: 'bar',
        name: '% Over Provision',
    };

    var trace2 = {
        x: tracesAll,
        y: underAll,
        type: 'bar',
        name: '% Under Provision',
    };

    var data = [trace1, trace2];
    var layout = {
        autosize:true,
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.088,
            x: 0.2
        },
        //height: 200
    };

    Plotly.newPlot('overUnderProvisionAll', data,layout);
}


function plotCostAll(costAll, tracesAll) {

    var trace1 = {
        x: tracesAll,
        y: costAll,
        type: 'bar',
        name: 'Cost $',
    };


    var data = [trace1];
    var layout = {
        title: "Cost $",
        autosize:true,
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.088,
            x: 0.2
        },
        //height: 200
    };

    Plotly.newPlot('costAll', data,layout);
}


function plotDerivationTimeAll(derivationTimeAll, tracesAll) {

    var trace1 = {
        x: derivationTimeAll,
        y: tracesAll,
        type: 'bar',
        name: 'Cost $',
    };


    var data = [trace1];
    var layout = {
        title: "Derivation time",
        autosize:true,
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.088,
            x: 0.2
        },
    };

    Plotly.newPlot('derivationTimeAll', data,layout);
}


function plotBarComparisonAll(values, tracesAll, title, varName, nameDiv) {

    var trace1 = {
        x: tracesAll,
        y: values,
        type: 'bar',
        name: varName,
    };


    var data = [trace1];
    var layout = {
        title: title,
        autosize:true,
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',

    };

    Plotly.newPlot(nameDiv, data,layout);
}

function plotAccumulatedCostAll(time, accumulatedCostAll, tracesAll) {
    var data = [];
    var i = 0;
    accumulatedCostAll.forEach(function (item) {
        {
            data.push(
                {
                    x: time,
                    y: item,
                    type: 'scatter',
                    name: tracesAll[i],
                    line: {shape: 'hv'}

                }
            )
            i=i+1
        }
    })

    var layout = {
        title: 'Accumulated cost',
        autosize:true,
        //margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        //height: 200
    };

    Plotly.newPlot('accumulatedCostAll', data,layout);
}
function showNoResultsPanel(){
    var x = document.getElementById("singlePolicyDiv");
    x.style.display = "none";
    var y = document.getElementById("multiplePolicyDiv");
    y.style.display = "none";
    var m = document.getElementById("noResultsDiv");
    m.style.display = "block";
    var x = document.getElementById("candidatesDiv");
    x.style.display = "none";
}

function showCandidatesDiv() {
    var x = document.getElementById("candidatesDiv");
    x.style.display = "block";
}

function hideCandidatesDiv() {
    var x = document.getElementById("candidatesDiv");
    x.style.display = "none";
}

function showSinglePolicyPanels() {
    var m = document.getElementById("noResultsDiv");
    m.style.display = "none";

    var x = document.getElementById("multiplePolicyDiv");
    x.style.display = "none";

    var y = document.getElementById("singlePolicyDiv");
    if (y.style.display === "none") {
        y.style.display = "block";
    }
    var x = document.getElementById("candidatesDiv");
    x.style.display = "block";

    var m = document.getElementById("metricsDiv");
    m.style.display = "block";
}

function showMultiplePolicyPanels() {
    var x = document.getElementById("singlePolicyDiv");
    x.style.display = "none";

    var m = document.getElementById("metricsDiv");
    m.style.display = "none";

    var x = document.getElementById("candidatesDiv");
    x.style.display = "block";

    var y = document.getElementById("multiplePolicyDiv");
    if (y.style.display === "none") {
        y.style.display = "block";
    }
}