import sys
import re
import argparse

# patterns = ['(Client_(\d+))_(read|decrypt)_wall_sum','(Client_(\d+))_(read|decrypt)_user_sum']
pattern = '(Client_(\d+))_(read|decrypt)_wall_sum'
match_idxs = dict()
names = dict()
out_lines = dict()

def readFile(fname):
    fd = open(fname, 'r')

    label_line = fd.readline()
    labels = label_line.split(',')
    for label in labels:
        match = re.search(pattern, label)
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

    print("label, read_wall_sum, decrypt_wall_sum, total")
    round_cnt = 1

    for line in fd:
        tokens = line.split(',')
        for k,v in sorted(match_idxs.items()):
            v1 = tokens[v[0]]
            v2 = tokens[v[1]]
            tot = float(v1) + float(v2)
            tot_str = "%.6f" % tot
            label = names[k] + "_" + str(round_cnt)
            print("%s, %s, %s, %s" % (label, v1, v2, tot_str))
        round_cnt += 1

    # for line in fd:
        # tokens = line.split(',')
        # for k,v in sorted(match_idxs.items()):
            # v1 = tokens[v[0]]
            # v2 = tokens[v[1]]
            # tot = float(v1) + float(v2)
            # tot_str = "%.6f" % tot
            # if k not in out_lines:
                # out_lines[k] = [v1, v2, tot_str]
            # else:
                # val = out_lines[k]
                # val.extend([v1, v2, tot_str])
                # out_lines[k] = val

    # for k,v in sorted(out_lines.items()):
        # tmp = ','.join(v)
        # print("%s,%s" % (names[k], tmp))
    

def main():
    parser = argparse.ArgumentParser(description='Parsing csv files')
    parser.add_argument('fname', type=str)
    # parser.add_argument('--user', action='store_true') 
    args = parser.parse_args()
    # readFile(args.fname, args.user)
    readFile(args.fname)

if __name__ == '__main__':
    main()
