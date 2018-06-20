from  scipy import signal
import numpy as np
import matplotlib.pyplot as plt

def getMeassures(x, threshold):

    vector = np.array(x)
    invvector = vector * -1
    peaks, properties = signal.find_peaks(vector, distance=2, prominence=1, width=1, height=1, rel_height=1, threshold=1)
    valleys, propValleys = signal.find_peaks(invvector,  distance=2, prominence=1, width=1,rel_height=1, threshold=0.1)

    response = []
    i = 0
    j = 0
    while i < peaks.size:
        try:
            interval = {}
            interval["peak"] = True
            interval["index_peak"] = int(peaks[i])
            interval["widht_heights"] = properties["width_heights"][i]
            interval["left_ips"] = properties["left_ips"][i]
            interval["right_ips"] = properties["right_ips"][i]

            if (j) < len(valleys):
                if(peaks[i] < valleys[j] and j==0):
                    valley = {}
                    valley["index_left_valley"] = 0
                    interval["start"] = valley

                if(peaks[i] > valleys[j]):
                    valley = {}
                    valley["index_left_valley"] = int(valleys[j])
                    valley["widht_heights"] = propValleys["width_heights"][j] * -1
                    valley["left_ips"] = propValleys["left_ips"][j]
                    valley["right_ips"] = propValleys["right_ips"][j]
                    interval["start"] = valley
                    j+=1

            if (j) < len(valleys):
                if(peaks[i] < valleys[j]):
                    valley = {}
                    valley["index_right_valley"] = int(valleys[j])
                    valley["widht_heights"] = propValleys["width_heights"][j] * -1
                    valley["left_ips"] = propValleys["left_ips"][j]
                    valley["right_ips"] = propValleys["right_ips"][j]
                    interval["end"] = valley
            else:
                valley = {}
                valley["index_right_valley"] = len(x)-1
                interval["end"] = valley
            i += 1

            start = interval["start"]["index_left_valley"]
            end = interval["end"]["index_right_valley"]
            interval ["index_in_interval"] = list(range(start+1,end))
            response.append(interval)

        except IndexError:
            print("IndexError ","i:", i, " j:", j)

    return response,peaks, valleys, properties, propValleys, vector, invvector

#DEPRECATED
def findIntersection(x, index, threshold, step):
    if (index + step) >= len(x):
        return len(x)-1
    elif (index+step) <=0:
        return 0
    if (x[index + step] <= threshold and x[index ] > threshold) or (x[index] <= threshold and x[index +step] > threshold):
        return calculateX(threshold, x[index], index, x[index+step], index+step)
    else:
        return findIntersection(x, index+step, threshold, step)

#DEPRECATED
def calculateX(y,y1,x1,y2,x2):
    try:
        m = (y2-y1)/(x2-x1)
        x = (y-y1 + m*x1)/m
    except ZeroDivisionError:
        print ("x1:", x1, " y1:", y1, " x2:", x2, " y2:", y2)
    return x

def plotGraph(x, peaks, valleys, properties, propValleys, vector, invvector, threshold, response):
    plt.plot(x)
    plt.hlines(y=threshold, xmin=0, xmax=25, color="C6")
    plt.plot(peaks, vector[peaks], "x", color="C1")
   #plt.vlines(x=peaks, ymin=vector[peaks] - properties["prominences"], ymax=vector[peaks], color="C1")
   #plt.hlines(y=vector[peaks] - properties["prominences"], xmin=properties["left_bases"], xmax=properties["right_bases"],color="C4")
    plt.hlines(y=properties["width_heights"], xmin=properties["left_ips"], xmax=properties["right_ips"], color = "C1")

    plt.plot(valleys, invvector[valleys] * -1, "o", color="C2")
    plt.vlines(x=valleys, ymin=(invvector[valleys] - propValleys["prominences"]) * -1, ymax=(invvector[valleys]) * -1, color="C2")
   # plt.hlines(y= - invvector[valleys] + propValleys["prominences"], xmin=propValleys["left_bases"],xmax=propValleys["right_bases"], color="C5")
    plt.hlines(y=propValleys["width_heights"] * -1, xmin=propValleys["left_ips"], xmax=propValleys["right_ips"],color="C2")

    xcoords = []
    for i in response:
        a = i["start"]["index_left_valley"]
        xcoords.append(a)
        a = i["end"]["index_right_valley"]
        xcoords.append(a)

    for xc in xcoords:
        plt.axvline(x=xc, color="C8")
    plt.show()

