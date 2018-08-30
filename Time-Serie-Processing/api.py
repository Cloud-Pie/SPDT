from flask import Flask, request, jsonify
import json
from processing import getMeassures, plotGraph

app = Flask(__name__)

@app.route("/api/peaks", methods=['POST'])
def processSignal():
    threshold = request.json['threshold']
    serie = request.json['serie']
    response, peaks, valleys, properties, propValleys, vector, invvector = getMeassures(serie, threshold)
    y ={"PoI": response}
    return jsonify(y)

@app.route("/api/peaks/plot", methods=['POST'])
def processAndPlotSignal():
    try:
        threshold = request.json['threshold']
        serie = request.json['serie']
        response,peaks, valleys, properties, propValleys, vector, invvector = getMeassures(serie, threshold)
        if len(peaks)>0:
            plotGraph(serie, peaks, valleys, properties, propValleys, vector, invvector, threshold, response)
        y ={"PoI": response}
        return jsonify(y)
    except ValueError:
        return "error"


if __name__ == "__main__":
    app.run(port=5003)