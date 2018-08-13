const url = 'http://localhost:8083/api/policies/5b70aab9780b410a1ccce3e3'
const request_url = 'http://localhost:8083/api/forecast/5b70aab9780b410a1ccce3dc'
const url_forecast_capacity = 'http://localhost:8083/api/forecast/5b70aab9780b410a1ccce3dc/5b7118d3780b4104104572fc'

app = {
  spdInit: function(){
      fetch(url)
          .then((response) => response.json())
          .then(function (data){
              units = app.getVirtualUnits(data)
              app.plotVirtualUnits(units)
          })
          .catch(function(err) {
              console.log('Fetch Error :-S', err);
          });
  },
  capacity: function() {
        fetch(url_forecast_capacity)
            .then((response) => response.json())
            .then(function (data2){
                app.plotCapacity(data2)
            })
            .catch(function(err) {
                console.log('Fetch Error :-S', err);
            });
    },
  getVirtualUnits: function(data){
    newData = [];
    arrayConfig = data.configuration
      arrayConfig.forEach(function (conf) {
        timestamp = new Date(conf.TimeStart)
        vmSet = conf.State.VMs
          for (var key in vmSet) {
                  vms = vmSet[key]
          }
          services = conf.State.Services
          for (var key in services) {
            rep = services[key].Scale
          }
          newData.push({
              "timestamp": timestamp,
              "vms":vms,
              "replicas": rep
          })
      })
      return newData
  },
  plotVirtualUnits: function(data){
      var chart = AmCharts.makeChart("virtualUnits", {
          "type": "serial",
          "theme": "light",
          "autoMarginOffset":10,
          "dataProvider": data,
          "valueAxes": [{
              "axisAlpha": 0,
              "position": "right"
          }],
          "graphs": [{
              "id":"g1",
              "balloonText": "[[category]]<br><b>[[value]] VM(s)</b>",
              "type": "step",
              "lineThickness": 2,
              "bullet":"square",
              "bulletAlpha":0,
              "bulletSize":4,
              "bulletBorderAlpha":0,
              "valueField": "vms"
          },{
              "id":"g2",
              "balloonText": "[[category]]<br><b>[[value]] Replica(s)</b>",
              "type": "step",
              "lineThickness": 2,
              "bullet":"square",
              "bulletAlpha":0,
              "bulletSize":4,
              "bulletBorderAlpha":0,
              "valueField": "replicas"
          }],
          "chartScrollbar": {
              "graph":"g1",
              "gridAlpha":0,
              "color":"#888888",
              "scrollbarHeight":25,
              "backgroundAlpha":0,
              "selectedBackgroundAlpha":0.1,
              "selectedBackgroundColor":"#888888",
              "graphFillAlpha":0,
              "autoGridCount":true,
              "selectedGraphFillAlpha":0,
              "graphLineAlpha":1,
              "graphLineColor":"#c2c2c2",
              "selectedGraphLineColor":"#888888",
              "selectedGraphLineAlpha":1
          },
          "chartCursor": {
              "fullWidth":true,
              "categoryBalloonDateFormat": "YYYY-MM-DD JJ:NN:SS",
              "cursorAlpha": 0.05,
              "graphBulletAlpha": 1
          },
          "dataDateFormat": "YYYY-MM-DD JJ:NN:SS",
          "categoryField": "timestamp",
          "categoryAxis": {
              "minPeriod": "ss",
              "parseDates": true,
              "gridAlpha": 0
          },
          "export": {
              "enabled": false
          }
      });
  },
  plotCapacity: function(data){
        var chart = AmCharts.makeChart("requestsUnits", {
            "type": "serial",
            "theme": "light",
            "autoMarginOffset":10,
            "dataProvider": data,
            "valueAxes": [{
                "axisAlpha": 0,
                "position": "right"
            }],
            "graphs": [{
                "id":"g1",
                "balloonText": "[[category]]<br><b>[[value]] Requests</b>",
                "type": "line",
                "lineThickness": 2,
                "bullet":"square",
                "bulletAlpha":0,
                "bulletSize":4,
                "bulletBorderAlpha":0,
                "valueField": "Capacity"
            },{
                "id":"g2",
                "balloonText": "[[category]]<br><b>[[value]] Requests</b>",
                "type": "line",
                "lineThickness": 2,
                "bullet":"square",
                "bulletAlpha":0,
                "bulletSize":4,
                "bulletBorderAlpha":0,
                "valueField": "Requests"
            }],
            "chartScrollbar": {
                "graph":"g1",
                "gridAlpha":0,
                "color":"#888888",
                "scrollbarHeight":25,
                "backgroundAlpha":0,
                "selectedBackgroundAlpha":0.1,
                "selectedBackgroundColor":"#888888",
                "graphFillAlpha":0,
                "autoGridCount":true,
                "selectedGraphFillAlpha":0,
                "graphLineAlpha":1,
                "graphLineColor":"#c2c2c2",
                "selectedGraphLineColor":"#888888",
                "selectedGraphLineAlpha":1
            },
            "chartCursor": {
                "fullWidth":true,
                "categoryBalloonDateFormat": "YYYY-MM-DD JJ:NN:SS",
                "cursorAlpha": 0.05,
                "graphBulletAlpha": 1
            },
            "dataDateFormat": "YYYY-MM-DD JJ:NN:SS",
            "categoryField": "Timestamp",
            "categoryAxis": {
                "minPeriod": "ss",
                "parseDates": true,
                "gridAlpha": 0
            },
            "export": {
                "enabled": false
            }
        });
    },
};