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
    cpuCores = [];
    memGB = [];
    typesVmSet = [];
    arrayConfig = data.configuration
    arrayConfig.forEach(function (conf) {
        time.push(conf.TimeStart)
        let text = ""
        let totalVMS = 0
        vmSet = conf.State.VMs
        for (var key in vmSet) {
            text = text + key + ":" + vmSet[key] + ", "
            totalVMS = totalVMS + vmSet[key]

        }
        vms.push(totalVMS)
        typesVmSet.push(text)
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
        memGB: memGB,
        typesVmSet: typesVmSet
    }
}

function plotVirtualUnits(time, vms, replicas, textHover) {
    var trace1 = {
        x: time,
        y: vms,
        type: 'scatter',
        name: 'N° VMs',
        text: textHover,
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
        titlefont: {
            size: 20
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
        titlefont: {
            size: 20
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
    var data = [trace1, trace2];
    Plotly.newPlot('requestsUnits', data,layout);
}

function plotMemCPU(time, memGB, cpuCores) {
    var trace1 = {
        x: time,
        y: memGB,
        type: 'scatter',
        name: 'Mem GB',
        line: {shape: 'hv'}
    };

    var trace2 = {
        x: time,
        y: cpuCores,
        type: 'scatter',
        name: 'CPU Cores',
        line: {shape: 'hv'}
    };

    var layout = {
        title: 'Resources provisioned',
        titlefont: {
            size: 20
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
    var data = [trace1, trace2];
    Plotly.newPlot('resourceUtilization', data,layout);
}

function searchByID(policyId) {
    if (policyId == null) {
        policyId = document.getElementById("searchpolicyid").value;
    }

    requestURL=policiesEndpoint+policyId
    var units
    fetch(requestURL)
        .then((response) => response.json())
        .then(function (data){

            showResultsPannel()
            showSinglePolicyPannels()
            var timeStart = new Date(data.window_time_start).toISOString();
            var timeEnd = new Date( data.window_time_end).toISOString();
            units = getVirtualUnits(data)
            plotVirtualUnits(units.time, units.vms, units.replicas, units.typesVmSet)
            plotMemCPU(units.time, units.memGB, units.cpuCores)
            fillData(data)
            fillDetailsTable(data)
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
                    plotCapacity(data.Timestamp, data.Requests, units.trn, units.time)
                })
                .catch(function(err) {
                    console.log('Fetch Error :-S', err);
                });
        })
        .catch(function(err) {
            console.log('Fetch Error :-S', err);
            showNoResultsPannel()
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
    console.log(requestURL)
    fetch(requestURL)
        .then((response) => response.json())
        .then(function (data){
            requestDemand = data.Requests
            fetch(policiesEndpoint)
                .then((response) => response.json())
                .then(function (data){
                    allPolicies = data
                    if(allPolicies.length > 0) {
                        showResultsPannel()
                        fillCandidateTable(data)
                        searchByID(allPolicies[0].id)
                    }else{
                        showNoResultsPannel()
                    }
                })
                .catch(function(err) {
                    console.log('Fetch Error :-S', err);
                });

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

}

function fillDetailsTable(policy) {
    $("#tBodyDetails").children().remove()
    parameters = policy.Parameters

    $("#tDetails > tbody").append("<tr>" +
        "<td><b>"+"Policy ID"+"</b></td>" +
        "<td><b>"+policy.id+"</b></td>" +
        "</tr>");

    $("#tDetails > tbody").append("<tr>" +
        "<td><b>"+"Algorithm"+"</b></td>" +
        "<td>"+policy.algorithm+"</td>" +
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
    parameters = policy.Parameters
    for (var key in parameters) {
        $("#lParameters").append(
            "<li><label>"+key+":"+"</label><span>"+parameters[key]+"</span></li>");
    }
}

function clickedCompareAll(){
    hideSinglePolicyPannels()

    units = getVirtualUnitsAll(allPolicies)
    plotCapacityAll(units.time,requestDemand, units.trnAll, units.tracesAll)
    plotVMsAll(units.time, units.vmsAll, units.tracesAll)
    plotReplicasAll(units.time, units.replicasAll,units.tracesAll)
    plotCPUAll(units.time, units.cpuCoresAll, units.tracesAll)
    plotMemAll(units.time, units.memGBAll, units.tracesAll)
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
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
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
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
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
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
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
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
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
        margin: {l: 25,r: 35,b: 45,t: 35, pad: 0},
        paper_bgcolor:'rgba(0,0,0,0)',
        plot_bgcolor:'rgba(0,0,0,0)',
        height: 200
    };

    Plotly.newPlot('replicaUnitsAll', data,layout);
}

function showNoResultsPannel(){
    var x = document.getElementById("searchOutputDiv");
    x.style.display = "none";

    var m = document.getElementById("noResultsDiv");
    m.style.display = "block";
}

function showResultsPannel(){
    var x = document.getElementById("searchOutputDiv");
    x.style.display = "block";

    var m = document.getElementById("noResultsDiv");
    m.style.display = "none";
}