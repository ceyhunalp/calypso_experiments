import sys
import re
import argparse

patterns = ['(Client_(\d+))_(read|decrypt)_wall_avg','(Client_(\d+))_(read|decrypt)_user_avg']
match_idxs = dict()
names = dict()

def readFile(fname, isUser):
    fd = open(fname, 'r')

    label_line = fd.readline()
    labels = label_line.split(',')
    if isUser:
        pattern_idx = 1
    else:
        pattern_idx = 0
    for label in labels:
        match = re.search(patterns[pattern_idx], label)
        if match is not None:
            cliNum = int(match.group(2))
            names[cliNum] = match.group(1)
            colIdx = labels.index(label)
            if cliNum not in match_idxs:
                match_idxs[cliNum] = [-1, -1]
            val = match_idxs[cliNum]
            if match.group(3) == "read":
                val[0] = colIdx
            elif match.group(3) == "decrypt":
                val[1] = colIdx
            match_idxs[cliNum] = val

    if isUser:
        print("label, read_user_avg, decrypt_user_avg")
    else:
        print("label, read_wall_avg, decrypt_wall_avg")

    for line in fd:
        tokens = line.split(',')
        for k,v in sorted(match_idxs.items()):
            print("%s,%s,%s" % (names[k], tokens[v[0]], tokens[v[1]]))

def main():
    parser = argparse.ArgumentParser(description='Parsing csv files')
    parser.add_argument('fname', type=str)
    parser.add_argument('--user', action='store_true') 
    args = parser.parse_args()
    # print(args.user)
    readFile(args.fname, args.user)

if __name__ == '__main__':
    main()
