var chart;
var xAxis;
var yAxis;

var chartData = [
    // data for first series
    {
        "date1": "2014-12-02",
        "y1": 6.5
    }, {
        "date1": "2014-12-04",
        "y1": 12.3
    }, {
        "date1": "2014-12-06",
        "y1": 12.3
    }, {
        "date1": "2014-12-08",
        "y1": 2.8
    }, {
        "date1": "2014-12-09",
        "y1": 3.5
    }, {
        "date1": "2014-12-12",
        "y1": 5.1
    }, {
        "date1": "2014-12-15",
        "y1": 6.7
    }, {
        "date1": "2014-12-17",
        "y1": 8
    }, {
        "date1": "2014-12-19",
        "y1": 8.9
    }, {
        "date1": "2014-12-22",
        "y1": 9.7
    }, {
        "date1": "2014-12-23",
        "y1": 10.4
    }, {
        "date1": "2014-12-24",
        "y1": 1.7
    },
    // data for second series
    {
        "date2": "2014-12-01",
        "y2": 2.2
    }, {
        "date2": "2014-12-03",
        "y2": 4.9
    }, {
        "date2": "2014-12-06",
        "y2": 5.1
    }, {
        "date2": "2014-12-07",
        "y2": 13.3
    }, {
        "date2": "2014-12-09",
        "y2": 6.1
    }, {
        "date2": "2014-12-10",
        "y2": 8.3
    }, {
        "date2": "2014-12-12",
        "y2": 10.5
    }, {
        "date2": "2014-12-13",
        "y2": 12.3
    }, {
        "date2": "2014-12-18",
        "y2": 4.5
    }, {
        "date2": "2014-12-21",
        "y2": 15
    }, {
        "date2": "2014-12-24",
        "y2": 10.8
    }, {
        "date2": "2014-12-26",
        "y2": 19
    }
];

AmCharts.ready(function () {
    // XY Chart
    chart = new AmCharts.AmXYChart();
    chart.dataProvider = chartData;

    // AXES
    // X
    xAxis = new AmCharts.ValueAxis();
    xAxis.position = "bottom";
    xAxis.axisAlpha = 0;
    xAxis.type = "date"
    chart.addValueAxis(xAxis);

    // Y
    yAxis = new AmCharts.ValueAxis();
    yAxis.position = "left";
    yAxis.axisAlpha = 0;
    chart.addValueAxis(yAxis);

    // GRAPHS
    // first graph
    var graph = new AmCharts.AmGraph();
    graph.lineColor = "#b0de09";
    graph.xField = "date1";
    graph.yField = "y1";
    graph.lineAlpha = 1;
    graph.bullet = "diamond";
    graph.balloonText = "<b>[[x]]</b> <b>[[y]]</b>"
    chart.addGraph(graph);

    // second graph
    graph = new AmCharts.AmGraph();
    graph.lineColor = "#fcd202";
    graph.xField = "date2";
    graph.yField = "y2";
    graph.lineAlpha = 1;
    graph.bullet = "diamond";
    graph.balloonText = "<b>[[x]]</b> <b>[[y]]</b>"
    chart.addGraph(graph);

    // CURSOR
    var chartCursor = new AmCharts.ChartCursor();
    chartCursor.addListener("moved", handleMove);
    chart.addChartCursor(chartCursor);


    // SCROLLBAR
    var chartScrollbar = new AmCharts.ChartScrollbar();
    chart.addChartScrollbar(chartScrollbar);

    // WRITE
    chart.write("chartdiv");
});

function handleMove(event){
    var xValue = AmCharts.roundTo(xAxis.coordinateToValue(event.x - xAxis.axisX), 2);
    var yValue = AmCharts.roundTo(yAxis.coordinateToValue(event.y - yAxis.axisY), 2);
}


plotRequests: function(data){
    <!-- Chart code -->
    var chart = AmCharts.makeChart("requestsUnits", {
        "type": "xy",
        "theme": "light",
        "autoMarginOffset":10,
        "dataProvider": data,
        "valueAxes": [
            {
                "id": "v1",
                "axisAlpha": 0
            }, {
                "id": "v2",
                "axisAlpha": 0,
                "position": "bottom",
                "type": "date",
            }
        ],
        "graphs": [{
            "id":"g1",
            "lineThickness": 2,
            "bullet":"diamond",
            "bulletAlpha":0,
            "bulletSize":4,
            "bulletBorderAlpha":0,
            "xField":"timestamp1",
            "yField":"capacity",
            "balloonText": "<b>[[x]]</b> <b>[[y]] requests</b>"
        },{
            "id":"g2",
            "lineThickness": 2,
            "bullet":"diamond",
            "bulletAlpha":0,
            "bulletSize":4,
            "bulletBorderAlpha":0,
            "xField":"timestamp",
            "yField":"requests",
            "balloonText": "<b>[[x]]</b> <b>[[y]] requests</b>"
        }],
        "dataDateFormat": "YYYY-MM-DD JJ:NN:SS",
        "categoryAxis": {
            "minPeriod": "hh",
            "parseDates": true,
            "gridAlpha": 1
        }
    });
}