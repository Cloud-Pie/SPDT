const policiesEndpoint = 'http://localhost:8083/api/policies/'
const request_url = 'http://localhost:8083/api/forecast/5b70aab9780b410a1ccce3dc'
const url_forecast_capacity = 'http://localhost:8083/api/forecast/5b73f235780b41401462afb3/5b73f453780b41428cabef03'

app = {
  spdInit: function(){
      fetch(policiesEndpoint)
          .then((response) => response.json())
          .then(function (data){
              units = getVirtualUnits(data)
              plotVirtualUnits(units.time, units.vms, units.replicas)
          })
          .catch(function(err) {
              console.log('Fetch Error :-S', err);
          });
  },
  capacity: function() {
        fetch(url_forecast_capacity)
            .then((response) => response.json())
            .then(function (data2){
                plotCapacity(data2)
            })
            .catch(function(err) {
                console.log('Fetch Error :-S', err);
            });
    },
};


function getVirtualUnits(data){
    console.log(data)
    time = [];
    vms = [];
    replicas = [];
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
        }
    })
   return {
        time: time,
        vms: vms,
        replicas: replicas
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

function plotCapacity(time, demand, supply){
    var trace1 = {
        x: time,
        y: demand,
        name: 'Demand',
        type: 'scatter'
    };

    var trace2 = {
        x: time,
        y: supply,
        name: 'Supply',
        type: 'scatter'
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

function searchByID() {
    var policyId = document.getElementById("searchpolicyid").value;
    requestURL=policiesEndpoint+policyId

    fetch(requestURL)
        .then((response) => response.json())
        .then(function (data){
            units = getVirtualUnits(data)
            plotVirtualUnits(units.time, units.vms, units.replicas)
            fillData(data)
            fillParameters(data)
        })
        .catch(function(err) {
            console.log('Fetch Error :-S', err);
        });
}

function searchByTimestamp() {
    var timeStart = document.getElementById("datetimestart").value;
    var timeEnd = document.getElementById("datetimeend").value;
    requestURL=policiesEndpoint
    fetch(requestURL)
        .then((response) => response.json())
        .then(function (data){
            fillCandidateTable(data)
        })
        .catch(function(err) {
            console.log('Fetch Error :-S', err);
        });
}

function fillCandidateTable(policyCandidates) {
    $("tBodyCandidates").children().remove()
    for(var i = 0; i < policyCandidates.length; i++) {
        $("#tCandidates > tbody").append("<tr>" +
            "<td>"+policyCandidates[i].id+"</td>" +
            "<td>"+policyCandidates[i].algorithm+"</td>" +
            "<td>"+policyCandidates[i].metrics.Cost+"</td>" +
            "<td>"+policyCandidates[i].Status+"</td>" +
            "</tr>");
    }
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
    parameters = policy.Parameters
    console.log(parameters)
    for (var key in parameters) {
        $("#lParameters").append(
            "<li>"+key+":"+"<span>"+parameters[key]+"</span></li>");
    }
}