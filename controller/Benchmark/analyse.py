import os
import matplotlib.pyplot as plt
import time
import numpy as np
import pandas as pd
import random

MAX = 100 #None
MAX_INDEX = 20000000#35450
MIN_INDEX = 0#32700 # 100000

def analyse_ping_rtt(file):
    f = open(file, "r",errors='ignore')
    lines = f.readlines()
    Delay = []
    round = []
    round_recv = []
    count = 0
    ran_now = []
    #print(len(lines))
    for line in lines:
        #print(line)
        line = line.split(",")
        if len(line) <= 1 :
            continue 
        #print(line[1])
        if line[1] == "Connect UE":
            delay_ue = []
            Delay.append(delay_ue)
            #print(len(delay_ue),len(Delay[0]))
            round.append(0)
            round_recv.append(0)
            #print(len(Delay))
        if len(line) == 4 and line[1].isdigit():
            ue_index = int(line[0])
            packet_seq = int(line[1]) + 65536 * round[ue_index-1]
            #print(ue_index,int(line[1]),packet_seq,len(Delay[ue_index-1]),round)
            if packet_seq < len(Delay[ue_index-1]) - 60000:
                round[ue_index-1] = round[ue_index-1] + 1
                packet_seq = int(line[1]) + 65536 * round[ue_index-1]
                time.sleep(1)
            while len(Delay[ue_index-1]) < packet_seq -1 :
            #     # if len(Delay[ue_index-1]) == packet_seq -3 and packet_seq != 3 :
            #     #     Delay[ue_index-1].append(Delay[ue_index-1][packet_seq -4])
            #     #     Delay[ue_index-1].append(Delay[ue_index-1][packet_seq -4])
            #     # elif len(Delay[ue_index-1]) == packet_seq -2 and packet_seq != 2 :
            #     #     Delay[ue_index-1].append(Delay[ue_index-1][packet_seq -3])
            #     # else :
                #print(len(Delay[ue_index-1]),packet_seq-1)
                Delay[ue_index-1].append(MAX)
                if ue_index == 4 :
                   ran_now.append(None)
            #     #print(ue_index, len(Delay[ue_index-1]), packet_seq, "loss!")
            if len(Delay[ue_index-1]) > packet_seq -1:
                Delay[ue_index-1][packet_seq-1] = MAX
                if ue_index == 4 :
                    ran_now[packet_seq-1] = None
            else :
                Delay[ue_index-1].append(MAX)
                if ue_index == 4 :
                    ran_now.append(None)
        #     packet_seq_start = int(line[1]) + 65536 * round[ue_index-1]
            
        #     while len(Delay[ue_index]) < packet_seq -1 :
        #         Delay[ue_index].append(0)
        #     Delay[ue_index].append(line[3])
        if len(line) == 14:
            ue_index = int(line[0])
            ran_id = int((line[4].split(" ")[0]).split("ran")[1])
            # if ue_index == 2:
            #     count = count + 1
            #print(ue_index,Delay[ue_index-1])
            packet_seq_recv = int(line[1])
            #print(ue_index,int(line[1]),packet_seq,len(Delay[ue_index-1]),round)
            if packet_seq_recv < len(Delay[ue_index-1]) - 60000*(round[ue_index-1]):
                #print(packet_seq_recv,len(Delay[ue_index-1]) - 60000*(round[ue_index-1]))
                packet_seq_recv = int(line[1]) + 65536 * round[ue_index-1]
                #time.sleep(1)
            #print(ue_index-1, packet_seq_recv-1)
            Delay[ue_index-1][packet_seq_recv-1] = time_handle(line[13]) - time_handle(line[3])
            if Delay[ue_index-1][packet_seq_recv-1] >= MAX:
                Delay[ue_index-1][packet_seq_recv-1] = MAX
            if ue_index == 4 :
                ran_now[packet_seq_recv-1] = ran_id+1
            # print(ue_index,int(line[1]),packet_seq,len(Delay[ue_index-1]),round)
            
            # while len(Delay[ue_index-1]) < packet_seq -1 :
            #     # if len(Delay[ue_index-1]) == packet_seq -3 and packet_seq != 3 :
            #     #     Delay[ue_index-1].append(Delay[ue_index-1][packet_seq -4])
            #     #     Delay[ue_index-1].append(Delay[ue_index-1][packet_seq -4])
            #     # elif len(Delay[ue_index-1]) == packet_seq -2 and packet_seq != 2 :
            #     #     Delay[ue_index-1].append(Delay[ue_index-1][packet_seq -3])
            #     # else :
            #     Delay[ue_index-1].append(200)
            #     if ue_index == 4 :
            #         ran_now.append(8)
            #     #print(ue_index, len(Delay[ue_index-1]), packet_seq, "loss!")
            # if len(Delay[ue_index-1]) > packet_seq -1:
            #     Delay[ue_index-1][packet_seq-1] = time_handle(line[13]) - time_handle(line[3])
            #     if ue_index == 4 :
            #         ran_now[packet_seq-1] = ran_id
            # else :
            #     Delay[ue_index-1].append(time_handle(line[13]) - time_handle(line[3]))
            #     if ue_index == 4 :
            #         ran_now.append(ran_id)
    #print (count)
    f.close()
    return Delay,ran_now



def time_handle(time_string):
    t = time_string.split(":")[:3]
    # print(t)
    t = (int(t[0]) * 3600 + int(t[1]) * 60 + float(t[2])) * 1000
    return t

if __name__ == "__main__":
    file_dir = "./log/" + "05-07_19-03-51-single-xn.log"  #07-10_17-20-18-single-xn.log   06-17_14-34-51-single-xn.log  07-11_15-24-25-single-xn 06-18_15-43-38-single-xn
    Delay ,ran_now = analyse_ping_rtt(file_dir)
    x1 = []
    x2 = []
    x3 = []
    x4 = []
    x = []
    for i in range(len(Delay[0])):
        x1.append(i+1)
    for i in range(len(Delay[1])):
        x2.append(i+1)
    for i in range(len(Delay[2])):
        x3.append(i+1)
    for i in range(len(Delay[3])):
        x4.append(i+1)
    #print(len(x1),len(x2),len(x3),len(x4))
    x.append(len(x1))
    x.append(len(x2))
    x.append(len(x3))
    x.append(len(x4))
    x_max = max(x)
    while len(x1) < x_max:
        Delay[0].append(MAX)
        x1.append(len(x1)+1)
    while len(x2) < x_max:
        Delay[1].append(MAX)
        x2.append(len(x2)+1)
    while len(x3) < x_max:
        Delay[2].append(MAX)
        x3.append(len(x3)+1)
    while len(x4) < x_max:
        Delay[3].append(MAX)
        ran_now.append(None)
        x4.append(len(x4)+1)
    
    X_index = min(x_max,MAX_INDEX)
    X0_index = min(x_max,MIN_INDEX)
    print(len(x4[:X_index]),len(Delay[3]),len(ran_now))
    
    for i in range(len(ran_now)):
        if ran_now[i] == None:
            ran_now[i] = ran_now[i-1]


    Delay_AVER = sum(Delay[3][X0_index:X_index])/(X_index-X0_index)
    print(Delay_AVER)

    df = pd.DataFrame(ran_now[X0_index:X_index])
    df.to_csv('Analysis_Data_3.csv', index=False)
    df = pd.DataFrame(Delay[3][X0_index:X_index])
    df.to_excel('Analysis_Delay_cho.xlsx', index=False)

    fig = plt.figure(figsize=(24, 15))
    #plt.subplot(221)
    #plt.scatter(x1, Delay[0], color='b', linestyle='-')
    plt.fill_between(x1[X0_index:X_index], Delay[0][X0_index:X_index],color = "skyblue",alpha = 0.4)
    plt.scatter(x1[X0_index:X_index], Delay[0][X0_index:X_index], c='blue', s=0.4) #0.1 
    plt.savefig('zz1_test.png')
    #plt.subplot(222)
    #plt.plot(x2, Delay[1], color='b', linestyle='-')
    fig = plt.figure(figsize=(24, 15))
    plt.fill_between(x2[X0_index:X_index], Delay[1][X0_index:X_index],color = "skyblue",alpha = 0.4)
    plt.savefig('zz2_test.png')
    #plt.subplot(223)
    #plt.plot(x3, Delay[2], color='b', linestyle='-')
    fig = plt.figure(figsize=(24, 15))
    plt.fill_between(x3[X0_index:X_index], Delay[2][X0_index:X_index],color = "skyblue",alpha = 0.4)
    plt.scatter(x3[X0_index:X_index], Delay[2][X0_index:X_index], c='blue', s=0.4)
    plt.savefig('zz3_test.png')
    #plt.subplot(224)
    #plt.plot(x4, Delay[3], color='r', linestyle='-')
    fig = plt.figure(figsize=(24, 15))
    plt.fill_between(x4[X0_index:X_index], Delay[3][X0_index:X_index],color = "red",alpha = 0.3)
    plt.scatter(x4[X0_index:X_index], Delay[3][X0_index:X_index], c='red', s=0.4)
    #plt.show()
    plt.savefig('zz4_test.png')

    fig = plt.figure(figsize=(24, 15))
    #plt.fill_between(x4[X0_index:X_index], ran_now[X0_index:X_index],color = "green",alpha = 0.3)
    plt.plot(x4[X0_index:X_index], ran_now[X0_index:X_index], c='green') #, s=0.1 0.001
    plt.yticks([0,1,2,3,4,5], ['0','1','2','3','4','5'])
    #plt.show()
    plt.savefig('ran_now_4_test.png')

