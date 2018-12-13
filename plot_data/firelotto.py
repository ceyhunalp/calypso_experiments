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
x = caly_data[:,0]
num_txn = caly_data[:,1]

caly_time=caly_data[:,2]
tournament_time=tournament_data[:,2]

labels = ['{}'.format(int(num_txn[i])) for i in range(len(num_txn))]

tournament_time_mask=np.isfinite(tournament_time)
caly_time_mask=np.isfinite(caly_time)

# plot(x,y4,'-yD', "Flat/MAC 0.25$\,$MB (PBFT)", purple)
# plot(x[tournament_time_mask],tournament_time[tournament_time_mask],'-bD', "Tournament", yellow)
# plot(x[caly_time_mask],caly_time[caly_time_mask],'-rD', "Calypso", purple)
plt.plot(x[caly_time_mask],caly_time[caly_time_mask], linestyle='--', label="Calypso", marker='o', color='g')
plt.plot(x[tournament_time_mask],tournament_time[tournament_time_mask],
        linestyle='--', label="Tournament", marker='s', color='r')

for label, xval, yval in zip(labels, x,  caly_time):
    plt.annotate('%s' % label, xy = (xval, yval))
for label, xval, yval in zip(labels, x,  tournament_time):
    plt.annotate('%s' % label, xy = (xval, yval))

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
# plt.yscale('log')
# plt.xscale('log')
plt.ylabel('Latency (sec)', fontsize=fs_label)
plt.xlabel('Day', fontsize=fs_label)
# plt.xlabel('Number of participants', fontsize=fs_label)
plt.grid(True)
plt.ylim((0,120))
plt.xlim((0,31))
plt.legend(loc=4, fontsize=fs_axis)
save("firelotto.eps")
