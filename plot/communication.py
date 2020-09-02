#!/usr/bin/env python
import numpy as np
import matplotlib.pyplot as plt
from colors import *

tour_data = read_datafile('tint_4_128.csv')
caly_data = read_datafile('cint_4_128.csv')
# tour_data = read_datafile('tournament_interaction.csv')
# caly_data = read_datafile('calypso_interaction.csv')


dark_1 = "#7b1fa2"
dark_2 = "#ff8f00"

x = tour_data[:,0]
print(x)
x_ticks = []
for xx in x:
    x_ticks.append(int(xx))

tour_time = tour_data[:,1]
caly_time = caly_data[:,1]

ax1 = plt.subplot(211)
plt.plot(x, tour_time, linestyle='-', color=dark_1,
        marker='s', markersize=8, label='Tournament')
plt.plot(x, caly_time, linestyle='-', color=dark_2,
        marker='o', markersize=8, label='Calypso')


# plt.xscale('log')
plt.ylabel('Time (sec)', fontsize=14)
plt.ylim((0,1500))
plt.xlim((0,130))
plt.grid(True)
ax1.set_xticks(x_ticks)
ax1.set_xticklabels(x_ticks)
ax1.legend(loc=0, fontsize=14)

tourb_data = read_datafile('tbytes_4to128.csv')
calyb_data = read_datafile('cbytes_4to128.csv')
# tourb_data = read_datafile('tournament_bytes.csv')
# calyb_data = read_datafile('calypso_bytes.csv')
tour_bytes = tourb_data[:,1]
caly_bytes = calyb_data[:,1]
print(tour_bytes)

ax2 = plt.subplot(212, sharex=ax1)
plt.plot(x, tour_bytes, color=dark_1, linestyle='-', marker='s', markersize=8, label='Tournament')
plt.plot(x, caly_bytes, color=dark_2, linestyle='-', marker='o', markersize=8, label='Calypso')
plt.ylabel('Bandwidth (KB)', fontsize=14)
ax2.set_ylim((0,160))
ax2.set_xlim((0,130))
ax2.set_xticks(x_ticks)
ax2.set_xticklabels(x_ticks)
ax2.legend(loc=0, fontsize=14)
plt.xlabel('Number of participants', fontsize=14)
plt.grid(True)

save("client_interaction.eps")
