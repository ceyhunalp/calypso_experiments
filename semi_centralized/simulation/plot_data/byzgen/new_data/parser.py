#!/usr/bin/python
import string
import os
import sys

def getNonBlockMeasurements(fn):
    fd = open(fn, "r")
    relevantCols = list()
    line = fd.readline()
    names = line.split(",")
    for i in range(len(names)):
        name = names[i]
        if "Decrypt_wall_sum" in name or "WriteProof_wall_sum" in name:
            relevantCols.append(i)

    line = fd.readline()
    values = line.split(",")

    for i in range(len(relevantCols)):
        colIdx = relevantCols[i]
        print(names[colIdx] + "," + values[colIdx])
        # label = names[colIdx].split("_")[1]
        # print(label + "," + values[colIdx])

    fd.close()

def getBlockMeasurements(fn):
    fd = open(fn, "r")
    relevantCols = list()
    line = fd.readline()
    names = line.split(",")
    for i in range(len(names)):
        name = names[i]
        if "wall_avg" in name and "Block" in name:
            relevantCols.append(i)

    line = fd.readline()
    values = line.split(",")

    for i in range(len(relevantCols)):
        colIdx = relevantCols[i]
        label = names[colIdx].split("_")[1]
        print(label + "," + values[colIdx])

    fd.close()

fn = sys.argv[1]
option = int(sys.argv[2])

if option == 1:
    getBlockMeasurements(fn)
elif option == 2:
    getNonBlockMeasurements(fn)
else:
    print("Invalid parameter")

