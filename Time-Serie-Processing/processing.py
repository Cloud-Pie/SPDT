import scipy.signal
import numpy as np
import matplotlib.pyplot as plt

def getMeassures(x, threshold):

    vector = np.array(x)
    invvector = vector * -1
    peaks, properties = scipy.signal.find_peaks(vector, distance=2, prominence=1, width=1, height=threshold, rel_height=1)
    valleys, propValleys = scipy.signal.find_peaks(invvector,  distance=2, prominence=1, width=1,rel_height=1)

    #plotGraph(x,peaks,valleys,properties,propValleys,vector,invvector)

    response = []
    i = 0
    j = 0
    index = 0
    while True:
        try:
            interval = {}
            if i >= peaks.size and j >= valleys.size:
                return response
            if (i < peaks.size and j >= valleys.size):
                interval["peak"] = True
                interval["index"] = int(peaks[i])
                index = int(peaks[i])
                i += 1
            elif( i >= peaks.size and j < valleys.size):
                interval["peak"] = False
                interval["index"] = int(valleys[j])
                index = int(valleys[j])
                j += 1
            elif(peaks[i] > valleys[j]):
                interval["peak"] = False
                interval["index"] = int(valleys[j])
                index = int(valleys[j])
                j += 1
            elif(peaks[i] < valleys[j]):
                interval["peak"] = True
                interval["index"] = int(peaks[i])
                index = int(peaks[i])
                i += 1

           #left intersection
            interval["start"] = findIntersection(x, index, threshold, -1)
            # right intersection
            interval["end"] = findIntersection(x, index, threshold, 1)
            response.append(interval)
        except IndexError:
            print("IndexError ","i:", i, " j:", j)

    return response

def findIntersection(x, index, threshold, step):
    if (index + step) >= len(x):
        return len(x)-1
    elif (index+step) <=0:
        return 0
    if (x[index + step] <= threshold and x[index ] > threshold) or (x[index] <= threshold and x[index +step] > threshold):
        return calculateX(threshold, x[index], index, x[index+step], index+step)
    else:
        return findIntersection(x, index+step, threshold, step)


def calculateX(y,y1,x1,y2,x2):
    try:
        m = (y2-y1)/(x2-x1)
        x = (y-y1 + m*x1)/m
    except ZeroDivisionError:
        print ("x1:", x1, " y1:", y1, " x2:", x2, " y2:", y2)
    return x

def plotGraph(x, peaks, valleys, properties, propValleys, vector, invvector):
    plt.plot(x)
    plt.hlines(y=12, xmin=0, xmax=25, color="C6")
    plt.plot(peaks, vector[peaks], "x", color="C1")
    plt.vlines(x=peaks, ymin=vector[peaks] - properties["prominences"], ymax=vector[peaks], color="C1")
    plt.hlines(y=vector[peaks] - properties["prominences"], xmin=properties["left_bases"], xmax=properties["right_bases"],color="C4")
    plt.hlines(y=properties["width_heights"], xmin=properties["left_ips"], xmax=properties["right_ips"], color = "C1")

    plt.plot(valleys, invvector[valleys] * -1, "o", color="C2")
    plt.vlines(x=valleys, ymin=(invvector[valleys] - propValleys["prominences"]) * -1, ymax=(invvector[valleys]) * -1,
               color="C2")
    plt.hlines(y= - invvector[valleys] + propValleys["prominences"], xmin=propValleys["left_bases"],xmax=propValleys["right_bases"], color="C5")
    plt.hlines(y=propValleys["width_heights"] * -1, xmin=propValleys["left_ips"], xmax=propValleys["right_ips"],color="C2")
    plt.show()
