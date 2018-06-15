from flask import Flask, request, jsonify
import json
from processing import getMeassures

app = Flask(__name__)

@app.route("/api/peaks", methods=['POST'])
def processSignal():
    threshold = request.json['threshold']
    serie = request.json['serie']
    y ={"PoI": getMeassures(serie, threshold)}
    return jsonify(y)


if __name__ == "__main__":
    app.run(port=5003)