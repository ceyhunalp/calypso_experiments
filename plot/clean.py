import sys
import re

patterns = ['Client_\d+_wall_avg']
match_idxs = set()

fname = sys.argv[1]
fd = open(fname, 'r')

label_line = fd.readline()
labels = label_line.split(',')

for label in labels:
    match = re.search(patterns[0], label)
    if match:
        match_idxs.add(labels.index(label))

sorted_idxs = sorted(match_idxs)

for idx in sorted_idxs:
    print("%s," % (labels[idx]), end = '')
print()

for line in fd:
    tokens = line.split(',')
    for idx in sorted_idxs:
        print("%s," % (tokens[idx]), end = '')
    print()
