#!/usr/bin/env python
import numpy as np
import matplotlib.pyplot as plt
from colors import *

tournament_data = read_datafile('firelotto_tournament.csv')
caly_data = read_datafile('firelotto_calypso.csv')

# x=tournament_data[:,0]
# tournament_time=tournament_data[:,1]

# x and num_txn is the same for calypso and tournament
# x is day #
# num_txn is the number of txns for that day

dark_1 = "#880e4f"
dark_2 = "#43a047"


x = caly_data[:,0]
num_txn = caly_data[:,1]

caly_time=caly_data[:,2]
tournament_time=tournament_data[:,2]

labels = ['{}'.format(int(num_txn[i])) for i in range(len(num_txn))]

tournament_time_mask=np.isfinite(tournament_time)
caly_time_mask=np.isfinite(caly_time)

plt.plot(x[tournament_time_mask],tournament_time[tournament_time_mask],
        linestyle='--', label="Tournament", marker='s', markersize=8, color=dark_2)
plt.plot(x[caly_time_mask],caly_time[caly_time_mask], linestyle='--',
        label="Calypso", marker='o', markersize=8, color=dark_1)

plt.ylabel('Latency (s)', fontsize=fs_label)
plt.xlabel('Day', fontsize=fs_label)
# plt.xlabel('Number of participants', fontsize=fs_label)
plt.grid(True)
plt.ylim((0,120))
plt.xlim((0,31))
plt.legend(loc=4, fontsize=fs_axis)
save("firelotto.eps")
