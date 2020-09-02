#!/usr/bin/env python
import numpy as np
import matplotlib.pyplot as plt
from colors import *

dark_1 = "#880e4f"
dark_2 = "#43a047"

zero_data = read_datafile('tournament_lottery_4to128.csv')
cali_data = read_datafile('calypso_lottery_4to128.csv')

x = zero_data[:,0]
zero_time = zero_data[:,1]
cali_time = cali_data[:,1]

zero_time_mask=np.isfinite(zero_time)
cali_time_mask=np.isfinite(cali_time)

# plot(x[zero_time_mask],zero_time[zero_time_mask],'-bD', "Tournament", yellow)
# plot(x[cali_time_mask],cali_time[cali_time_mask],'-rD', "Calypso", purple)
plt.plot(x[zero_time_mask],zero_time[zero_time_mask],linestyle='--',
        label="Tournament", marker = 's', color=dark_2)
plt.plot(x[cali_time_mask],cali_time[cali_time_mask],linestyle='--',
        label="Calypso", marker = 'o', color=dark_1)


# plt.xscale('log')
plt.ylabel('Latency (sec)', fontsize=fs_label)
plt.xlabel('Number of participants', fontsize=fs_label)
plt.grid(True)
plt.ylim((0,250))
plt.xlim((1,130))
plt.legend(loc=4, fontsize=fs_axis)

x_ticks = []
for xx in x:
    x_ticks.append(int(xx))

plt.axes().set_xticks(x[:])
plt.axes().set_xticklabels(x_ticks)
save("lottery_nolog.eps")
