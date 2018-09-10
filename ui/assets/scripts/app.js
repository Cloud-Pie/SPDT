const policiesEndpoint = 'http://localhost:8083/api/policies/'
const forecastRequestsEndpoint = 'http://localhost:8083/api/forecast?'

var allPolicies
var requestDemand
var timeLine

function getVirtualUnits(data){

    time = [];
    vms = [];
    replicas = [];
    TRN = [];
    utilizationCpuCores = [];
    utilizationMemGB = [];
    cpuCores = [];
    memGB = [];
    labelsTypesVmSet = [];
    arrayConfig = data.scaling_actions

    usedTypesString = data.parameters["vm-types"]
    vmTypesList = usedTypesString.split(",");

    vmScalesInTime = {}
    vmTypesList.forEach(function (vmType) {
        vmScalesInTime[vmType] = []
    })

    arrayConfig.forEach(function (conf) {
        time.push(conf.time_start)
        let vmLabels = ""
        let totalVMS = 0

        //Flags map to identify if a record for that type was already inserted
        vmScalesInTimeFlags = {}
        vmTypesList.forEach(function (vmType) {
            vmScalesInTimeFlags[vmType] = false
        })
        vmSet = conf.State.VMs
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
        labelsTypesVmSet.push(vmLabels)
        services = conf.State.Services
        for (var key in services) {
            replicas.push(services[key].Scale)
            cpuCores.push(services[key].CPU)
            memGB.push(services[key].Memory)
        }
        TRN.push(conf.metrics.requests_capacity)
        utilizationCpuCores.push(conf.metrics.cpu_utilization)
        utilizationMemGB.push(conf.metrics.mem_utilization)
    })

    //Needed to include the last time t into the plot
    lastConf = arrayConfig[arrayConfig.length - 1]
    time.push(lastConf.time_end)
    vmSet = lastConf.State.VMs
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
    labelsTypesVmSet.push(vmLabels)
    services = lastConf.State.Services
    for (var key in services) {
        replicas.push(services[key].Scale)
        cpuCores.push(services[key].CPU)
        memGB.push(services[key].Memory)
    }
    TRN.push(lastConf.metrics.requests_capacity)
    utilizationCpuCores.push(lastConf.metrics.cpu_utilization)
    utilizationMemGB.push(lastConf.metrics.mem_utilization)
    //End last t

   return {
        time: time,
        vms: vms,
        replicas: replicas,
        trn: TRN,
        utilizationCpuCores: utilizationCpuCores,
        utilizationMemGB: utilizationMemGB,
        labelsTypesVmSet: labelsTypesVmSet,
        vmScalesInTime: vmScalesInTime,
        cpuCores: cpuCores,
        memGB:memGB
    }
}

function plotVMUnitsPerType(time, vms, textHover) {
   let data = [];
   for (var key in vms) {
      data.push(
           {
               x: time,
               y: vms[key],
               name: key,
               type: 'scatter',
               line: {shape: 'hv'}

           }
       )

   }
   l = data.length
   data[l-1].text = textHover
   data.push(
        {
            x: time,
            y: requestDemand,
            name: 'Demand',
            type: 'scatter',
            line: {shape: 'spline', color:'#092e20'},
            yaxis: 'y2'
        }
    )

   var layout = {
        title: 'N° Virtual Machines',
        titlefont: {
           size:18
        },
        autosize:true,
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 300,
        yaxis: {title: 'N° VMs', range: [0, 8]},
        yaxis2: {
           title: 'Requests/Sec',
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

   // Plotly.newPlot('vmUnits', data, layout);
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
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 300,
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

function plotContainerUnits(time, replicas, cpuCores, memGB) {
    var trace1 = {
        x: time,
        y: replicas,
        type: 'scatter',
        name: 'N° Replicas',
        line: {shape: 'hv'}
    };

    var trace2 = {
        x: time,
        y: cpuCores,
        type: 'scatter',
        name: 'CPU cores',
        yaxis: 'y2',
    };

    var trace3 = {
        x: time,
        y: memGB,
        type: 'scatter',
        name: 'Mem GB',
        yaxis: 'y2',
    };

    var layout = {
        title: 'N° Containers',
        titlefont: {
           size:18
        },
        yaxis: {title: 'N° Containers', range: [0, 15]},
        yaxis2: {
            title: 'Resources',
            titlefont: {color: 'rgb(148, 103, 189)'},
            tickfont: {color: 'rgb(148, 103, 189)'},
            overlaying: 'y',
            side: 'right'
        },
        autosize:true,
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 300,
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
        title: 'Workload',
        titlefont: {
           size:18
        },
        autosize:true,
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 300,
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

function plotMem(time, memGB) {
    var trace = {
        x: time,
        y: memGB,
        type: 'scatter',
        name: 'Mem GB'
    };

    var layout = {
        title: 'Memory GB',
        titlefont: {
           size:18
        },
        yaxis: {range: [0, 100]},
        autosize:true,
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 300,
        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.088,
            x: 0.2
        },
    };
    var data = [trace];
    Plotly.newPlot('resourceMem', data,layout);
}

function plotCPU(time, cpuCores) {

    var trace = {
        x: time,
        y: cpuCores,
        type: 'scatter',
        name: 'CPU Cores'
    };

    var layout = {
        title: 'CPU Cores',
        titlefont: {
           size:18
        },
        yaxis: {range: [0, 100]},
        autosize:true,
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 300,
        legend: {
            "orientation": "h",
            xanchor: "center",
            y: 1.088,
            x: 0.2
        },
    };
    var data = [trace];
    Plotly.newPlot('resourceCPU', data,layout);
}

function searchByID(policyId) {
    if (policyId == null) {
        policyId = document.getElementById("searchpolicyid").value;
    }

    requestURL=policiesEndpoint+policyId
    var units
    fetch(requestURL)
        .then((response) => response.json())
        .then(function (policy){
           showPolicyInfo(policy)
           //hideCandidatesDiv()
        })
        .catch(function(err) {
            console.log('Fetch Error :-S', err);
            showNoResultsPanel()
        });
}


function showPolicyInfo(policy) {
    var units
    document.getElementById("jsonId").innerText = JSON.stringify(policy,undefined, 5);
    showSinglePolicyPanels()
    var timeStart = new Date(policy.window_time_start).toISOString();
    var timeEnd = new Date( policy.window_time_end).toISOString();
    units = getVirtualUnits(policy)

    let params = {
       "start": timeStart,
       "end": timeEnd
    }
    let esc = encodeURIComponent
    let query = Object.keys(params)
        .map(k => esc(k) + '=' + esc(params[k]))
        .join('&')

    url_forecast = forecastRequestsEndpoint + query
    fetch(url_forecast)
         .then((response) => response.json())
         .then(function (forecast){
                requestDemand = forecast.Requests
                plotCapacity(forecast.Timestamp, forecast.Requests, units.trn, units.time)
                plotVMUnitsPerType(units.time, units.vmScalesInTime, units.labelsTypesVmSet)
                plotContainerUnits(units.time, units.replicas, units.cpuCores, units.memGB)
                plotMem(units.time, units.utilizationMemGB)
                plotCPU(units.time, units.utilizationCpuCores)
                fillMetrics(policy)
                fillDetailsTable(policy)
         })
        .catch(function(err) {
              console.log('Fetch Error :-S', err);
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

    requestURL = forecastRequestsEndpoint + query
    fetch(requestURL)
        .then((response) => response.json())
        .then(function (data){
            requestDemand = data.Requests
            fetch(policiesEndpoint)
                .then((response) => response.json())
                .then(function (data){
                    allPolicies = data
                    if(allPolicies.length > 0) {
                        showSinglePolicyPanels()
                        fillCandidateTable(data)
                        showPolicyInfo(allPolicies[0])
                    }else{
                        showNoResultsPanel()
                    }
                })
                .catch(function(err) {
                    console.log('Fetch Error :-S', err);
                    showNoResultsPanel()
                });

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
            "<td>"+policyCandidates[i].id+"</td>" +
            "<td>"+policyCandidates[i].algorithm+"</td>" +
            "<td>"+policyCandidates[i].metrics.cost+"</td>" +
            "<td> <span class='label "+ label+" '>" +policyCandidates[i].status+ "</span> </td>" +
            "</tr>");
    }

   $("tr").click(function() {
        var id = $(this).find('td:first').text()
        searchByID(id)
    });
}

function fillMetrics(policy){
    document.getElementById("costid").innerText = policy.metrics.cost;
    document.getElementById("overid").innerText = policy.metrics.over_provision;
    document.getElementById("underid").innerText = policy.metrics.under_provision;
    document.getElementById("reconfid").innerText = policy.metrics.n_scaling_actions;
    document.getElementById("reconfVMsid").innerText = policy.metrics.num_scale_vms;
    document.getElementById("reconfContid").innerText = policy.metrics.num_scale_containers;
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

function fillParameters(policy){
    $("#lParameters").children().remove()
    $("#lParameters").append(
        "<li><label>"+"Algorithm:"+"</label><span>"+policy.algorithm+"</span></li>");
    parameters = policy.parameters
    for (var key in parameters) {
        $("#lParameters").append(
            "<li><label>"+key+":"+"</label><span>"+parameters[key]+"</span></li>");
    }
}

function clickedCompareAll(){
    showMultiplePolicyPanels()

    units = getVirtualUnitsAll(allPolicies)
    plotCapacityAll(units.time,requestDemand, units.trnAll, units.tracesAll)
    plotVMsAll(units.time, units.vmsAll, units.tracesAll)
    plotReplicasAll(units.time, units.replicasAll,units.tracesAll)
    plotCPUAll(units.time, units.cpuCoresAll, units.tracesAll)
    plotMemAll(units.time, units.memGBAll, units.tracesAll)
    plotNScalingVMAll(units.nScalingVMsAll, units.tracesAll)
    plotOverprovisionAll(units.overprovisionAll, units.tracesAll)
    plotUnderprovisionAll(units.underprovisionAll, units.tracesAll)
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
    policies.forEach(function (policy) {
        time = [];
        vms = [];
        replicas = [];
        TRN = [];
        utilizationCpuCores = [];
        utilizationMemGB = [];
        tracesAll.push(policy.id)
        arrayConfig = policy.scaling_actions
        arrayConfig.forEach(function (conf) {
            time.push(conf.time_start)

            vmSet = conf.State.VMs
            for (var key in vmSet) {
                vms.push(vmSet[key])
            }
            services = conf.State.Services
            for (var key in services) {
                replicas.push(services[key].Scale)
                utilizationCpuCores.push(services[key].CPU * services[key].Scale)
                utilizationMemGB.push(services[key].Memory * services[key].Scale)
            }
            TRN.push(conf.metrics.requests_capacity)
        })
        vmsAll.push(vms)
        replicasAll.push(replicas)
        TRNAll.push(TRN)
        cpuCoresAll.push(utilizationCpuCores)
        memGBAll.push(utilizationMemGB)
        overprovisionAll.push(policy.metrics.over_provision)
        underprovisionAll.push(policy.metrics.under_provision)
        nScalingVMsAll.push(policy.metrics.num_scale_vms)
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
        nScalingVMsAll: nScalingVMsAll
    }
}

function plotCapacityAll(time, demand, supplyAll,tracesAll){

    var data = [];
    data.push(
        {
            x: time,
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
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 200
    };

    Plotly.newPlot('requestsUnitsAll', data,layout);
}

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
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 200
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
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 200
    };

    Plotly.newPlot('cpuUtilizationAll', data,layout);
}

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
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 200
    };

    Plotly.newPlot('vmUnitsAll', data,layout);
}

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
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 200
    };

    Plotly.newPlot('replicaUnitsAll', data,layout);
}

function plotNScalingVMAll(nScalingVMs, tracesAll) {
    var trace = {
        x: tracesAll,
        y: nScalingVMs,
        type: 'bar'
    };

    var data = [trace];
    var layout = {
        title: 'N° times VM scaling',
        autosize:true,
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 200
    };

    Plotly.newPlot('nScalingVmsAll', data,layout);
}

function plotOverprovisionAll(overAll, tracesAll) {
    var trace = {
        x: tracesAll,
        y: overAll,
        type: 'bar'
    };

    var data = [trace];
    var layout = {
        title: '% Over provision',
        autosize:true,
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 200
    };

    Plotly.newPlot('overprovisionAll', data,layout);
}

function plotUnderprovisionAll(underAll, tracesAll) {
    var trace = {
        x: tracesAll,
        y: underAll,
        type: 'bar'
    };

    var data = [trace];
    var layout = {
        title: '% Under provision',
        autosize:true,
        margin: {l: 50,r: 50,b: 45,t: 45, pad: 4},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 200
    };

    Plotly.newPlot('underprovisionAll', data,layout);
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