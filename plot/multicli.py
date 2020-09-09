import sys
import re
import argparse

patterns = ['(Client_(\d+))_(read|decrypt)_wall_sum','(Client_(\d+))_(read|decrypt)_user_sum']
match_idxs = dict()
names = dict()
out_lines = dict()

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
        print("label, read_user_sum, decrypt_user_sum, total")
    else:
        print("label, read_wall_sum, decrypt_wall_sum, total")

    for line in fd:
        tokens = line.split(',')
        for k,v in sorted(match_idxs.items()):
            v1 = tokens[v[0]]
            v2 = tokens[v[1]]
            tot = float(v1) + float(v2)
            tot_str = "%.6f" % tot
            if k not in out_lines:
                out_lines[k] = [v1, v2, tot_str]
            else:
                val = out_lines[k]
                val.extend([v1, v2, tot_str])
                out_lines[k] = val

    for k,v in sorted(out_lines.items()):
        tmp = ','.join(v)
        print("%s,%s" % (names[k], tmp))
    

def main():
    parser = argparse.ArgumentParser(description='Parsing csv files')
    parser.add_argument('fname', type=str)
    parser.add_argument('--user', action='store_true') 
    args = parser.parse_args()
    # print(args.user)
    readFile(args.fname, args.user)

if __name__ == '__main__':
    main()
