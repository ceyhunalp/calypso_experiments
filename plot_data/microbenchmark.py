#!/usr/bin/env python
import numpy as np
import matplotlib.pyplot as plt
from colors import *

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

# plot(x,y4,'-yD', "Flat/MAC 0.25$\,$MB (PBFT)", purple)
plot(x[cent_write_mask],cent_write[cent_write_mask],'-bD', "Centralized (Insecure)", green)
plot(x[semi_write_mask],semi_write[semi_write_mask],'-rD', "Semi-centralized (Insecure)", yellow)
plot(x[cali_write_mask],cali_write[cali_write_mask],'-gD', "Calypso", purple)

# x=data[:,0]
# y1=data[:,1]
# y2=data[:,2]
# y3=data[:,3]
# y4=data[:,4]
# y1mask=np.isfinite(y1)
# y2mask=np.isfinite(y2)
# y3mask=np.isfinite(y3)
# plot(x,y4,'-yD', "Flat/MAC 0.25$\,$MB (PBFT)", purple)
# plot(x[y1mask],y1[y1mask],'-bD', "Flat/CoSi 1$\,$MB", green)
# plot(x[y3mask],y3[y3mask],'-gD', "Tree/Individual", red)
# plot(x[y2mask],y2[y2mask],'-rD', "Tree/CoSi (ByzCoin)", yellow)
plt.yscale('log')
plt.xscale('log', basex=2)
plt.ylabel('Latency (sec)', fontsize=fs_label)
plt.xlabel('Number of transactions', fontsize=fs_label)
plt.grid(True)
plt.ylim((0.01,200))
plt.xlim((1,512))
plt.legend(loc=4, fontsize=fs_axis)
save("write.eps")

plot(x[cent_read_mask],cent_read[cent_read_mask],'-bD', "Centralized (Insecure)", green)
plot(x[semi_read_mask],semi_read[semi_read_mask],'-rD', "Semi-centralized (Insecure)", yellow)
plot(x[cali_read_mask],cali_read[cali_read_mask],'-gD', "Calypso", purple)

plt.yscale('log')
plt.xscale('log', basex=2)
plt.ylabel('Latency (sec)', fontsize=fs_label)
plt.xlabel('Number of transactions', fontsize=fs_label)
plt.grid(True)
plt.ylim((0.01,300))
plt.xlim((1,512))
plt.legend(loc=4, fontsize=fs_axis)
save("read.eps")
