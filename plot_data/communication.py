#!/usr/bin/env python
import numpy as np
import matplotlib.pyplot as plt
from colors import *

tour_data = read_datafile('tournament_interaction.csv')
caly_data = read_datafile('calypso_interaction.csv')


dark_1 = "#7b1fa2"
dark_2 = "#ff8f00"

x = tour_data[:,0]
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


plt.xscale('log')
plt.ylabel('Time (s)', fontsize=14)
plt.ylim((0,1500))
plt.xlim((3,300))
plt.grid(True)
ax1.set_xticks(x_ticks)
ax1.set_xticklabels(x_ticks)
ax1.legend(loc=0, fontsize=16)

tourb_data = read_datafile('tournament_bytes.csv')
calyb_data = read_datafile('calypso_bytes.csv')
tour_bytes = tourb_data[:,1]
caly_bytes = calyb_data[:,1]

ax2 = plt.subplot(212, sharex=ax1)
plt.plot(x, tour_bytes, color=dark_1, linestyle='-', marker='s', markersize=8, label='Tournament')
plt.plot(x, caly_bytes, color=dark_2, linestyle='-', marker='o', markersize=8, label='Calypso')
plt.ylabel('Communication (KB)', fontsize=14)
ax2.set_ylim((0,310))
ax2.set_xlim((3,300))
ax2.set_xticks(x_ticks)
ax2.set_xticklabels(x_ticks)
ax2.legend(loc=0, fontsize=16)
plt.xlabel('Number of participants', fontsize=14)
plt.grid(True)


# x = tourb_data[:,0]
# print(x)
# x_ticks = []
# for xx in x:
    # x_ticks.append(int(xx))

# tour_bytes = tourb_data[:,1]
# caly_bytes = calyb_data[:,1]

# ax2.plot(x, tour_bytes, color='yellow', linestyle='-', marker='o', label='Tournament')
# ax2.plot(x, caly_bytes, color='purple', linestyle='-', marker='o', label='Calypso')

# ax2.set_xlabel('Number of participants', fontsize=14)
# ax2.set_ylabel('bytes', fontsize=14)
# ax2.set_xticks(x_ticks)
# ax2.set_xticklabels(x_ticks)
# ax2.set_xlim((0,512))
# ax2.set_ylim((0,300000))
# ax2.set_xscale('log')
# ax2.grid(True)
# ax2.legend(loc=0, fontsize=16)

save("client_interaction.eps")
