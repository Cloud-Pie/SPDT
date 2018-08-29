const policiesEndpoint = 'http://localhost:8083/api/policies/'
const forecastRequestsEndpoint = 'http://localhost:8083/api/forecast?'

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
        width: 640,
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
        })
        .catch(function(err) {
            console.log('Fetch Error :-S', err);
        });
}

function fillCandidateTable(policyCandidates) {
    $("#tBodyCandidates").children().remove()
    for(var i = 0; i < policyCandidates.length; i++) {
        $("#tCandidates > tbody").append("<tr>" +
            "<td>"+policyCandidates[i].id+"</td>" +
            "<td>"+policyCandidates[i].algorithm+"</td>" +
            "<td>"+policyCandidates[i].metrics.Cost+"</td>" +
            "<td>"+policyCandidates[i].Status+"</td>" +
            "</tr>");
    }

    $("tr").click(function() {
        var id = $(this).find('td:first').text()
        var x = document.getElementById("multiplePolicyDiv");
        x.style.display = "none";
        var y = document.getElementById("singlePolicyDiv");
        if (y.style.display === "none") {
            y.style.display = "block";
        }
        searchByID(id)
    });
}

function fillData(policy){
    document.getElementById("costid").innerText = policy.metrics.Cost.toFixed(2);
    document.getElementById("overid").innerText = policy.metrics.OverProvision.toFixed(2);;
    document.getElementById("underid").innerText = policy.metrics.UnderProvision.toFixed(2);
    document.getElementById("reconfid").innerText = policy.metrics.NumberConfigurations;

    document.getElementById("policyid").innerText = policy.id;
    document.getElementById("startperiod").innerText = policy.window_time_start;
    document.getElementById("endperiod").innerText = policy.window_time_end;
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
    var x = document.getElementById("singlePolicyDiv");
    x.style.display = "none";

    var y = document.getElementById("multiplePolicyDiv");
    if (y.style.display === "none") {
        y.style.display = "block";
    }
}