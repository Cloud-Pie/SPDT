const url = 'http://localhost:8083/api/policies/5b6c6ad6780b41207425ebf0'
ts = [{
    "timestamp": "2018-03-02 08:20:00",
    "value": -0.307
}, {
    "timestamp": "2018-03-02 09:20:00",
    "value": -0.168
}, {
    "timestamp": "2018-03-02 10:20:00",
    "value": -0.073
}, {
    "timestamp": "2018-03-02 11:20:00",
    "value": -0.027
}, {
    "timestamp": "2018-03-02 12:20:00",
    "value": 0.251
}, {
    "timestamp": "2018-03-02 13:20:00",
    "value": -0.281
}, {
    "timestamp": "2018-03-02 14:20:00",
    "value": -0.348
}, {
    "timestamp": "2018-03-02 15:20:00",
    "value": 0.074
}]
app = {
  test: function(){
      fetch(url)
          .then((response) => response.json())
          .then(function (data){

              console.log(data)
              t3 = app.getVmData(data)
              t2 = app.getReplicasData(data)
              app.plotVMs(t3)
          })
          .catch(function(err) {
              console.log('Fetch Error :-S', err);
          });
  },
  getVmData: function(data){
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
      console.log(newData)
      return newData
  },
  getReplicasData: function(data){
        newData = [];
        arrayConfig = data.configuration
        arrayConfig.forEach(function (conf) {
            timestamp = new Date(conf.TimeStart)
            services = conf.State.Services
            for (var key in services) {
                newData.push({
                    "timestamp": timestamp,
                    "value": services[key].Scale
                })
            }
        })
      console.log(newData)
      return newData
  },
  plotVMs: function(data){
      <!-- Chart code -->
      var chart = AmCharts.makeChart("chartdiv", {
          "type": "serial",
          "theme": "light",
          "autoMarginOffset":25,
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
              "scrollbarHeight":55,
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
  initPickColor: function() {
    $('.pick-class-label').click(function() {
      var new_class = $(this).attr('new-class');
      var old_class = $('#display-buttons').attr('data-class');
      var display_div = $('#display-buttons');
      if (display_div.length) {
        var display_buttons = display_div.find('.btn');
        display_buttons.removeClass(old_class);
        display_buttons.addClass(new_class);
        display_div.attr('data-class', new_class);
      }
    });
  },
  showNotification: function(from, align) {
    color = 'primary';

    $.notify({
      icon: "now-ui-icons ui-1_bell-53",
      message: "Welcome to the <b> Scaling Policy Derivation Tool </b>"
    }, {
      type: color,
      timer: 8000,
      placement: {
        from: from,
        align: align
      }
    });
  }
};