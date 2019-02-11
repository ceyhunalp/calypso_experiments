#!/usr/bin/env python
import numpy as np
import matplotlib.pyplot as plt
from colors import *

raw_data = read_datafile('plot.csv')

caly_data = raw_data[:,1]
semi_data = raw_data[:,2]
fully_data = raw_data[:,3]

index = np.arange(2)
bwid = 0.1

fig, ax = plt.subplots()
rects1 = ax.bar(index-bwid, fully_data, bwid, color=cig_orange, bottom=0.001, label='Fully')
rects2 = ax.bar(index, semi_data, bwid, color=pie_green, bottom=0.001, label='Semi')
rects3 = ax.bar(index+bwid, caly_data, bwid, color=warm_purp, bottom=0.001, label='Calypso')

ax.set_xticks(index+bwid/3)
ax.set_xticklabels(('Write', 'Read'))
# ax.set_yscale('log')
# ax.set_ylim([0.001,10])

ax.set_xlabel('Transaction', fontsize=14)
ax.set_ylabel('Time (s)', fontsize=14)
ax.legend(loc=9)
# ax.set_ylim((0,10))
# ax.grid(True)
# ax.set_yscale('log')

save("byzgen-nolog.eps")
