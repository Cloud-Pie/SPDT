var trace1 = {
    x: ["2014-12-02 10:30:00", "2014-12-03 10:30:00", "2014-12-04 10:30:00", "2014-12-05 10:30:00","2014-12-06 10:30:00", "2014-12-07 10:30:00", "2014-12-08 10:30:00", "2014-12-09 10:30:00"],
    y: [10, 15, 13, 17,20,22,25,20],
    type: 'scatter',
    line: {shape: 'hv'}
};

var trace2 = {
    x: ["2014-12-01 10:30:00", "2014-12-02 10:30:00", "2014-12-03 10:30:00", "2014-12-04 10:30:00"],
    y: [16, 5, 11, 9],
    type: 'scatter',
    line: {shape: 'hv'}
};

var layout = {
    autosize:true,
    margin: {
        l: 25,
        r: 35,
        b: 45,
        t: 35,
        pad: 0
    },
    paper_bgcolor:'rgba(0,0,0,0)',
    plot_bgcolor:'rgba(0,0,0,0)',
    width: 640,
    height: 200
};

var data = [trace1, trace2];

Plotly.newPlot('myDiv', data,layout);