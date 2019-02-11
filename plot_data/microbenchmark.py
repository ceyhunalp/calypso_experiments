#!/usr/bin/env python
import numpy as np
import matplotlib.pyplot as plt
from colors import *

gr = "#138d75"
prp = "#8e44ad"
yel = "#d68910"

cent_data = read_datafile('centralized.csv')
semi_data = read_datafile('semi.csv')
cali_data = read_datafile('calypso.csv')

x=cent_data[:,0]
cent_write=cent_data[:,1]
cent_read=cent_data[:,2]
semi_write=semi_data[:,1]
semi_read=semi_data[:,2]
cali_write=cali_data[:,1]
cali_read=cali_data[:,2]

cent_read_mask=np.isfinite(cent_read)
cent_write_mask=np.isfinite(cent_write)
semi_write_mask=np.isfinite(semi_write)
semi_read_mask=np.isfinite(semi_read)
cali_write_mask=np.isfinite(cali_write)
cali_read_mask=np.isfinite(cali_read)

plt.plot(x[cent_write_mask],cent_write[cent_write_mask],
        label="Fully-centralized (Insecure)", linestyle='--', markersize=8,
        marker='s', color=gr)
plt.plot(x[semi_write_mask],semi_write[semi_write_mask], label="Semi-centralized (Insecure)", linestyle='--', markersize=8,
        marker='v', color=yel)
plt.plot(x[cali_write_mask],cali_write[cali_write_mask], label="Calypso", linestyle='--', markersize=8,
        marker='D', color=prp)

plt.yscale('log')
plt.xscale('log')
plt.ylabel('Latency (s)', fontsize=fs_label)
plt.xlabel('Number of transactions', fontsize=fs_label)
plt.grid(True)
plt.ylim((0.01,100))
plt.xlim((2,300))
plt.legend(loc=4, fontsize=fs_axis)

x_ticks = []
for xx in x:
    x_ticks.append(int(xx))

plt.axes().set_xticks(x[:])
plt.axes().set_xticklabels(x_ticks)
save("write-usenix.eps")

plt.plot(x[cent_read_mask],cent_read[cent_read_mask],
        label="Fully-centralized (Insecure)", linestyle='--', markersize=8,
        marker='s', color=gr)
plt.plot(x[semi_read_mask],semi_read[semi_read_mask], label="Semi-centralized (Insecure)", linestyle='--', markersize=8,
        marker='v', color=yel)
plt.plot(x[cali_read_mask],cali_read[cali_read_mask], label="Calypso", linestyle='--', markersize=8,
        marker='D', color=prp)
# plt.plot(x[cent_read_mask],cent_read[cent_read_mask], label="Fully-centralized (Insecure)", )
# plt.plot(x[semi_read_mask],semi_read[semi_read_mask], label="Semi-centralized (Insecure)", )
# plt.plot(x[cali_read_mask],cali_read[cali_read_mask], label="Calypso", )

plt.yscale('log')
plt.xscale('log')
plt.ylabel('Latency (s)', fontsize=fs_label)
plt.xlabel('Number of transactions', fontsize=fs_label)
plt.grid(True)
plt.ylim((0.01,500))
plt.xlim((2,300))
plt.legend(loc=4, fontsize=fs_axis)
plt.axes().set_xticks(x[:])
plt.axes().set_xticklabels(x_ticks)
save("read-usenix.eps")
