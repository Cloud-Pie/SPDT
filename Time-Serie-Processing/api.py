from flask import Flask
import json
from processing import getMeassures

app = Flask(__name__)

@app.route("/api/peaks", methods=['GET'])
def processSignal():
    x = [10, 16, 20, 22, 15, 10, 15, 10, 8, 6, 7, 5, 10, 13, 14, 10, 8, 13, 8, 10, 7, 12, 15, 10, 12, 10]
    return json.dumps( {"response": getMeassures(x, 12)})


if __name__ == "__main__":
    app.run(port=5003)