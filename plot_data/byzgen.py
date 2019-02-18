#!/usr/bin/env python
from matplotlib.ticker import ScalarFormatter
import numpy as np
import matplotlib.pyplot as plt

yel = "#FFB300"
red = "#E91E63"
blu = "#3F51B5"

raw_data = read_datafile('plot.csv')

caly_data = raw_data[:,1]
semi_data = raw_data[:,2]
fully_data = raw_data[:,3]

index = np.arange(2)
bwid = 0.1

plt.bar(index-bwid, fully_data, bwid, color=yel, bottom=0.01,
        label='Fully-centralized')
plt.bar(index, semi_data, bwid, color=red, bottom=0.01,
        label='Semi-centralized')
plt.bar(index+bwid, caly_data, bwid, color=blu, bottom=0.01, label='Calypso')

plt.yscale('log')
plt.ylabel('Time (sec)', fontsize=fs_label)
plt.xlabel('Transaction type', fontsize=fs_label)
plt.grid(True)
plt.ylim((0,20))

y_ticks = ['0.01','0.01','0.1','1','10']

plt.axes().set_xticks(index+bwid/3)
plt.axes().set_xticklabels(('Write', 'Read'))
plt.gca().set_yticklabels(y_ticks)
plt.legend(loc=9)

save("byzgen-log.eps")
