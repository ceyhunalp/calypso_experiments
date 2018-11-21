#!/usr/bin/env python
import numpy as np
import matplotlib.pyplot as plt
from colors import *

zero_data = read_datafile('zero_lottery.csv')
cali_data = read_datafile('calypso_lottery.csv')

x=zero_data[:,0]
zero_time=zero_data[:,1]
cali_time=cali_data[:,1]

zero_time_mask=np.isfinite(zero_time)
cali_time_mask=np.isfinite(cali_time)

# plot(x,y4,'-yD', "Flat/MAC 0.25$\,$MB (PBFT)", purple)
plot(x[zero_time_mask],zero_time[zero_time_mask],'-bD', "Tournament", yellow)
plot(x[cali_time_mask],cali_time[cali_time_mask],'-rD', "Calypso", purple)

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
plt.xscale('log')
plt.ylabel('Latency (sec)', fontsize=fs_label)
plt.xlabel('Number of participants', fontsize=fs_label)
plt.grid(True)
plt.ylim((10,1000))
plt.xlim((1,512))
plt.legend(loc=4, fontsize=fs_axis)
save("lottery.eps")
