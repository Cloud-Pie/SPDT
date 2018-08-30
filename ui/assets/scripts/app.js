const policiesEndpoint = 'http://localhost:8083/api/policies/'
const forecastRequestsEndpoint = 'http://localhost:8083/api/forecast?'

var allPolicies
var requestDemand
var timeLine

function getVirtualUnits(data){
    console.log(data)
    time = [];
    vms = [];
    replicas = [];
    TRN = [];
    cpuCores = [];
    memGB = [];
    arrayConfig = data.configuration
    arrayConfig.forEach(function (conf) {
        time.push(conf.TimeStart)

        vmSet = conf.State.VMs
        for (var key in vmSet) {
            vms.push(vmSet[key])
        }
        services = conf.State.Services
        for (var key in services) {
            replicas.push(services[key].Scale)
            cpuCores.push(services[key].CPU * services[key].Scale)
            memGB.push(services[key].Memory * services[key].Scale)
        }
        TRN.push(conf.Metrics.CapacityTRN)
    })
    //Needed to include the last time t into the plot
    lastConf = arrayConfig[arrayConfig.length - 1]
    time.push(lastConf.TimeEnd)
    vmSet = lastConf.State.VMs
    for (var key in vmSet) {
        vms.push(vmSet[key])
    }
    services = lastConf.State.Services
    for (var key in services) {
        replicas.push(services[key].Scale)
        cpuCores.push(services[key].CPU * services[key].Scale)
        memGB.push(services[key].Memory * services[key].Scale)
    }
    TRN.push(lastConf.Metrics.CapacityTRN)

   return {
        time: time,
        vms: vms,
        replicas: replicas,
        trn: TRN,
        cpuCores: cpuCores,
        memGB: memGB
    }
}

function plotVirtualUnits(time, vms, replicas) {
    var trace1 = {
        x: time,
        y: vms,
        type: 'scatter',
        name: 'N° VMs',
        line: {shape: 'hv'}
    };

    var trace2 = {
        x: time,
        y: replicas,
        type: 'scatter',
        name: 'N° Replicas',
        line: {shape: 'hv'}
    };

    var layout = {
        title: 'Virtual Units',
        autosize:true,
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',

        height: 200
    };
    var data = [trace1, trace2];
    Plotly.newPlot('virtualUnits', data,layout);
}

function plotCapacity(time, demand, supply, timeSuply){
    var trace1 = {
        x: time,
        y: demand,
        name: 'Demand',
        type: 'scatter',
        line: {shape: 'spline'}
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
        autosize:true,
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        width: 640,
        height: 200
    };
    var data = [trace1, trace2];
    Plotly.newPlot('requestsUnits', data,layout);
}

function plotMemCPU(time, memGB, cpuCores) {
    var trace1 = {
        x: time,
        y: memGB,
        type: 'scatter',
        name: 'Mem GB',

    };

    var trace2 = {
        x: time,
        y: cpuCores,
        type: 'scatter',
        name: 'CPU Cores',

    };

    var layout = {
        title: 'Utilization',
        autosize:true,
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        width: 640,
        height: 200
    };
    var data = [trace1, trace2];
    Plotly.newPlot('resourceUtilization', data,layout);
}

function searchByID(policyId) {

    showSinglePolicyPannels()

    if (policyId == null) {
        policyId = document.getElementById("searchpolicyid").value;
    }

    requestURL=policiesEndpoint+policyId
    var units
    fetch(requestURL)
        .then((response) => response.json())
        .then(function (data){
            var timeStart = new Date(data.window_time_start).toISOString();
            var timeEnd = new Date( data.window_time_end).toISOString();
            units = getVirtualUnits(data)
            plotVirtualUnits(units.time, units.vms, units.replicas)
            plotMemCPU(units.time, units.memGB, units.cpuCores)
            fillData(data)
            fillParameters(data)
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
                .then(function (data){
                    requestDemand = data.Requests
                    plotCapacity(data.Timestamp, data.Requests, units.trn, units.time)
                })
                .catch(function(err) {
                    console.log('Fetch Error :-S', err);
                });
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

    fetch(policiesEndpoint)
        .then((response) => response.json())
        .then(function (data){
            fillCandidateTable(data)
            allPolicies = data
            if(allPolicies.length > 0) {
                searchByID(allPolicies[0].id)
            }
        })
        .catch(function(err) {
            console.log('Fetch Error :-S', err);
        });
}

function fillCandidateTable(policyCandidates) {
    $("#tBodyCandidates").children().remove()
    for(var i = 0; i < policyCandidates.length; i++) {

        label = "label-warning"
        if (policyCandidates[i].Status == "selected") {
            label = "label-success"
        }

        $("#tCandidates > tbody").append("<tr>" +
            "<td>"+policyCandidates[i].id+"</td>" +
            "<td>"+policyCandidates[i].algorithm+"</td>" +
            "<td>"+policyCandidates[i].metrics.Cost+"</td>" +
            "<td> <span class='label "+ label+" '>" +policyCandidates[i].Status+ "</span> </td>" +
            "</tr>");
    }

    $("tr").click(function() {
        var id = $(this).find('td:first').text()
        searchByID(id)
    });
}

function fillData(policy){
    document.getElementById("costid").innerText = policy.metrics.Cost;
    document.getElementById("overid").innerText = policy.metrics.OverProvision;
    document.getElementById("underid").innerText = policy.metrics.UnderProvision;
    document.getElementById("reconfid").innerText = policy.metrics.NumberConfigurations;

    document.getElementById("policyid").innerText = policy.id;
    document.getElementById("startperiod").innerText =  new Date( policy.window_time_start).toLocaleString();
    document.getElementById("endperiod").innerText = new Date( policy.window_time_end).toLocaleString();
}

function fillParameters(policy){
    $("#lParameters").children().remove()
    $("#lParameters").append(
        "<li>"+"Algorithm:"+"<span>"+policy.algorithm+"</span></li>");
    parameters = policy.Parameters
    for (var key in parameters) {
        $("#lParameters").append(
            "<li>"+key+":"+"<span>"+parameters[key]+"</span></li>");
    }
}

function clickedCompareAll(){
    hideSinglePolicyPannels()
    units = getVirtualUnitsAll(allPolicies)
    plotCapacityAll(units.time, units.trnAll)
    plotVMsAll(units.time, units.vmsAll)
    plotReplicasAll(units.time, units.replicasAll)
    plotCPUAll(units.time, units.cpuCoresAll)
    plotMemAll(units.time, units.memGBAll)
}

function hideSinglePolicyPannels() {
    var x = document.getElementById("singlePolicyDiv");
    x.style.display = "none";

    var m = document.getElementById("metricsDiv");
    m.style.display = "none";

    var d = document.getElementById("detailsDiv");
    d.style.display = "none";

    var y = document.getElementById("multiplePolicyDiv");
    if (y.style.display === "none") {
        y.style.display = "block";
    }
}

function showSinglePolicyPannels() {
    var x = document.getElementById("multiplePolicyDiv");
    x.style.display = "none";
    var y = document.getElementById("singlePolicyDiv");
    if (y.style.display === "none") {
        y.style.display = "block";
    }
    var m = document.getElementById("metricsDiv");
    m.style.display = "block";

    var d = document.getElementById("detailsDiv");
    d.style.display = "block";
}

function getVirtualUnitsAll(policies) {
    vmsAll = [];
    replicasAll = [];
    TRNAll = [];
    cpuCoresAll = [];
    memGBAll = [];
    tracesAll = []
    policies.forEach(function (policy) {
        time = [];
        vms = [];
        replicas = [];
        TRN = [];
        cpuCores = [];
        memGB = [];
        tracesAll.push(policy.id)
        arrayConfig = policy.configuration
        arrayConfig.forEach(function (conf) {
            time.push(conf.TimeStart)

            vmSet = conf.State.VMs
            for (var key in vmSet) {
                vms.push(vmSet[key])
            }
            services = conf.State.Services
            for (var key in services) {
                replicas.push(services[key].Scale)
                cpuCores.push(services[key].CPU * services[key].Scale)
                memGB.push(services[key].Memory * services[key].Scale)
            }
            TRN.push(conf.Metrics.CapacityTRN)
        })
        vmsAll.push(vms)
        replicasAll.push(replicas)
        TRNAll.push(TRN)
        cpuCoresAll.push(cpuCores)
        memGBAll.push(memGB)
    })

    return {
        time: time,
        tracesAll: tracesAll,
        vmsAll: vmsAll,
        replicasAll: replicasAll,
        trnAll: TRNAll,
        cpuCoresAll: cpuCoresAll,
        memGBAll: memGBAll
    }
}

function plotCapacityAll(time, supplyAll){
   console.log(requestDemand)
    var data = [];
    data.push(
        {
            x: time,
            y: requestDemand,
            name: 'Demand',
            type: 'scatter',
            line: {shape: 'spline'}
        }
    )
    supplyAll.forEach(function (item) {
        {
            data.push(
                {
                    x: time,
                    y: item,
                    name: 'supply',
                    type: 'scatter',
                    line: {shape: 'hv'}
                }
            )
        }
    })

    var layout = {
        title: 'Workload',
        autosize:true,
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        width: 640,
        height: 200
    };

    Plotly.newPlot('requestsUnitsAll', data,layout);
}

function plotMemAll(time, memGBAll) {

    var data = [];
    memGBAll.forEach(function (item) {
        {
            data.push(
                {
                    x: time,
                    y: item,
                    type: 'scatter',
                    name: 'Mem GB',

                }
            )
        }
    })


    var layout = {
        title: 'Utilization Memory',
        autosize:true,
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        width: 640,
        height: 200
    };

    Plotly.newPlot('memoryUtilizationAll', data,layout);
}

function plotCPUAll(time, cpuCoresAll) {

    var data = [];
    cpuCoresAll.forEach(function (item) {
        {
            data.push(
                {
                    x: time,
                    y: item,
                    type: 'scatter',
                    name: 'Mem GB',

                }
            )
        }
    })


    var layout = {
        title: 'Utilization CPU',
        autosize:true,
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        width: 640,
        height: 200
    };

    Plotly.newPlot('cpuUtilizationAll', data,layout);
}

function plotVMsAll(time, vmsAll) {

    var data = [];
    vmsAll.forEach(function (item) {
        {
            data.push(
                {
                    x: time,
                    y: item,
                    type: 'scatter',
                    name: 'N° VMs',
                    line: {shape: 'hv'}

                }
            )
        }
    })


    var layout = {
        title: 'N° VMs',
        autosize:true,
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        width: 640,
        height: 200
    };

    Plotly.newPlot('vmUnitsAll', data,layout);
}

function plotReplicasAll(time, replicasAll) {
    var data = [];
    replicasAll.forEach(function (item) {
        {
            data.push(
                {
                    x: time,
                    y: item,
                    type: 'scatter',
                    name: 'N° replias',
                    line: {shape: 'hv'}

                }
            )
        }
    })

    var layout = {
        title: 'N° Replicas',
        autosize:true,
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        width: 640,
        height: 200
    };

    Plotly.newPlot('replicaUnitsAll', data,layout);
}