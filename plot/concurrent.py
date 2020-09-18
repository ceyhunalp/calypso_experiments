#!/usr/bin/env python
import numpy as np
import matplotlib
import matplotlib.pyplot as plt
from colors import *

yel = "#F9A825"
red = "#AD1457"
blu = "#283593"
gre = "#108F32"

cdata = read_datafile('calypso-concurrent.csv')
sdata = read_datafile('semi-concurrent.csv')

x = cdata[:,0]
c_avg = cdata[:,1]
c_f = cdata[:,2]
c_nf = cdata[:,3]
s_avg = sdata[:,1]
s_f = sdata[:,2]
s_nf = sdata[:,3]

c_avg_mask = np.isfinite(c_avg)
c_f_mask = np.isfinite(c_f)
c_nf_mask = np.isfinite(c_nf)
s_avg_mask = np.isfinite(s_avg)
s_f_mask = np.isfinite(s_f)
s_nf_mask = np.isfinite(s_nf)

plt.plot(x[c_avg_mask], c_avg[c_avg_mask], label="Avg (Calypso)", linestyle='--',
        markersize=6, marker='x', color=gre)
# plt.plot(x[c_f_mask], c_f[c_f_mask], label="$5^{th}$ (Calypso)", linestyle='--',
        # markersize=6, marker='s', color=blu)
plt.plot(x[c_nf_mask], c_nf[c_nf_mask], label="$95^{th}$ (Calypso)", linestyle='--',
        markersize=6, marker='o', color=gre)
plt.plot(x[s_avg_mask], s_avg[s_avg_mask], label="Avg (Semi-centralized)", linestyle='--',
        markersize=6, marker='x', color=red)
# plt.plot(x[s_f_mask], s_f[s_f_mask], label="$5^{th}$ (Semi-centralized)", linestyle='--',
        # markersize=6, marker='v', color=red)
plt.plot(x[s_nf_mask], s_nf[s_nf_mask], label="$95^{th}$ (Semi-centralized)", linestyle='--',
        markersize=6, marker='o', color=red)

plt.ylabel('Latency (sec)', fontsize=fs_label)
plt.xlabel('Number of outstanding requests', fontsize=fs_label)
plt.grid(True)
plt.ylim((0,90))
plt.xlim(90,510)
# plt.legend(loc=9, fontsize=fs_axis)
plt.legend(loc=0, fontsize=11)

x_ticks = []
for xx in x:
    x_ticks.append(int(xx))

# y_ticks = ['0.1','1','10','100']

plt.xticks(x_ticks)
# plt.axes().set_xticks(x[:])
# plt.axes().set_xticklabels(x_ticks)
# plt.gca().set_yticklabels(y_ticks)
# plt.yscale('log')
save("concurrent.eps")
