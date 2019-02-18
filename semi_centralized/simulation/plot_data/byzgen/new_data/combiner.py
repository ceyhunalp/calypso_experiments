#!/usr/bin/python
import string
import os
import sys

parsed_file = sys.argv[1]
count_file = sys.argv[2]

fd = open(parsed_file, "r")
block_ids = list()
block_times = list()
write_counts = list()
read_counts = list()

for line in fd:
    line = line.rstrip()
    tokens = line.split(",")
    block_ids.append(tokens[0])
    block_times.append(tokens[1].replace(r'\r',''))
fd.close()

fd = open(count_file, "r")
for line in fd:
    line = line.rstrip()
    tokens = line.split(",")
    write_counts.append(tokens[0])
    read_counts.append(tokens[1])

print("Block,WriteCount,ReadCount,TotalCount,BlockTime,IsWrite,IsRead")
for i in range(len(block_ids)):
    wint = int(write_counts[i])
    rint = int(read_counts[i])
    total_count = wint + rint
    # total_count = str(total_count).strip("\n")
    iswrite = 0
    isread = 0
    if wint > 0:
        iswrite = 1
    if rint > 0:
        isread = 1
    # iswrite = str(iswrite).strip("\n")
    # isread = str(isread).strip("\n")
    print(block_ids[i]+","+write_counts[i]+","+read_counts[i]+","+str(total_count)+","+block_times[i]+","+str(iswrite)+","+str(isread))
