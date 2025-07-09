import os
import matplotlib.pyplot as plt
import time
import numpy as np
import pandas as pd
import random
import math


def analyse_dataplane_interrupttime(file):
    f = open(file, "r",errors='ignore')
    lines = f.readlines()
    Delay = []
    round = []
    round_recv = []
    count = 0
    ran_now = []
    #print(len(lines))
    BS1_timestamp = []
    BS2_timestamp = []
    BS3_timestamp = []
    BS1_156_flag = 0
    BS2_156_flag = 0
    BS3_156_flag = 0
    RACH_XnComplete = []
    ERROR_flag = 0
    meas_flag = False
    for line in lines:
        #print(line)
        line = line.split(",")
        if ERROR_flag == 1:
            RACH_XnComplete.append(time_handle(line[2]))
            ERROR_flag = 0
            if meas_flag == False:
                Pre_Time = time_handle(line[2]) - time_measure
                meas_flag = True
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
        if line[1] == "Measurement Report||" and meas_flag == False:
            time_measure = time_handle(line[9])

        if line[0] == "RSRP:BS1":
            print(line)
            timestamp_now = time_handle(line[19])
            for i in range(5):
                if line[i+1] == "-156.0000000000" and BS1_156_flag == 0 :
                    BS1_timestamp.append(timestamp_now)
                    BS1_156_flag = 1
                if BS1_156_flag == 1 and line[i+1] != "-156.0000000000":
                    BS1_timestamp.append(timestamp_now)
                    BS1_156_flag = 0
            for i in range(5):
                if line[i+7] == "-156.0000000000" and BS2_156_flag == 0 :
                    BS2_timestamp.append(timestamp_now)
                    BS2_156_flag = 1
                if BS2_156_flag == 1 and line[i+7] != "-156.0000000000":
                    BS2_timestamp.append(timestamp_now)
                    BS2_156_flag = 0
            for i in range(5):
                if line[i+13] == "-156.0000000000" and BS3_156_flag == 0 :
                    BS3_timestamp.append(timestamp_now)
                    BS3_156_flag = 1
                if BS3_156_flag == 1 and line[i+13] != "-156.0000000000":
                    BS3_timestamp.append(timestamp_now)
                    BS3_156_flag = 0
        if line[1].split("||")[0] == "RACH Target_RAN":
            print(line)
            if len(line) < 4:
                ERROR_flag = 1
            else:
                RACH_XnComplete.append(time_handle(line[3]))
                if meas_flag == False:
                    Pre_Time = time_handle(line[3]) - time_measure
                    meas_flag = True
        if line[1] == "Xn Handover Complete":
            RACH_XnComplete.append(time_handle(line[5]))
    f.close()
    return BS1_timestamp,BS2_timestamp,BS3_timestamp,RACH_XnComplete,Pre_Time

def SelectUEDuration(file,Start_timestamp,End_timestamp):
    f = open(file, "r",errors='ignore')
    lines = f.readlines()
    ue_timeset = []
    ue_seqset = []
    #Interruption = []
    for line in lines:
        line = line.split(",")
        if len(line) == 14:
            ue_index = int(line[0])
            if ue_index == 4:
                ue_timestamp = time_handle(line[3])
                if time_handle(line[13]) > Start_timestamp and time_handle(line[13]) < End_timestamp:
                    ue_timeset.append(ue_timestamp)
                    ue_seqset.append(int(line[1]))
    #print(ue_seqset)
    for i in range(len(ue_seqset)-1):
        if ue_seqset[i+1] > ue_seqset[i] + 1:
            Interruption=ue_timeset[i+1]-ue_timeset[i]
    f.close()
    return Interruption

def Transfer_SE(BS1_timestamp,BS2_timestamp,BS3_timestamp,RACH_XnComplete,index): #RAN0->RAN1 First Handover
    End_timestamp = RACH_XnComplete[2*index + 1]
    for i in range(int(len(BS1_timestamp)/2)):
        #print(RACH_XnComplete[0],BS1_timestamp[0],BS1_timestamp[2*i],BS1_timestamp[2*i+1])
        if RACH_XnComplete[2*index] < BS1_timestamp[0]:
            Start_timestamp = RACH_XnComplete[2*index]
            return Start_timestamp,End_timestamp
        if RACH_XnComplete[2*index] > BS1_timestamp[2*i] and RACH_XnComplete[2*index] <= BS1_timestamp[2*i+1] :
            Start_timestamp = BS1_timestamp[2*i]
            print(RACH_XnComplete[0]-BS1_timestamp[2*i])
            return Start_timestamp,End_timestamp
        if RACH_XnComplete[2*index] > BS1_timestamp[2*i-1] and RACH_XnComplete[2*index] <= BS1_timestamp[2*i] :
            print(BS1_timestamp[2*i-1])
            Start_timestamp = RACH_XnComplete[2*index]
            return Start_timestamp,End_timestamp
    return RACH_XnComplete[2*index],End_timestamp
    

def time_handle(time_string):
    t = time_string.split(":")[:3]
    # print(t)
    t = (int(t[0]) * 3600 + int(t[1]) * 60 + float(t[2])) * 1000  #ms_level
    return t


if __name__ == "__main__":
    file_dir = "./log/" + "05-07_19-03-51-single-xn.log"  #07-10_17-20-18-single-xn.log   06-17_14-34-51-single-xn.log  07-11_15-24-25-single-xn 06-18_15-43-38-single-xn
    BS1_timestamp,BS2_timestamp,BS3_timestamp,RACH_XnComplete, Pre_Time = analyse_dataplane_interrupttime(file_dir)
    print(BS1_timestamp)
    print(RACH_XnComplete)
    Interruption = []
    release = 1
    sum = 0
    for i in range(math.floor(len(RACH_XnComplete)/2)) :
        Start_timestamp,End_timestamp = Transfer_SE(BS1_timestamp,BS2_timestamp,BS3_timestamp,RACH_XnComplete,i)
        Interruption.append(SelectUEDuration(file_dir,Start_timestamp-2000,End_timestamp+2000))
    for i in range(len(Interruption)-1):
        if Interruption[i+1] > 100 :
            release = release + 1
            continue
        sum = sum + Interruption[i+1]

    print(Start_timestamp,End_timestamp)
    print(Pre_Time/1000)
    if len(Interruption)>1 :
        Aver_Interrupt = sum/(len(Interruption)-release)
        print(Interruption,Aver_Interrupt)
    else:
        print(Interruption)

    print(Interruption)