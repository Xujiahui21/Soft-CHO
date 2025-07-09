import os
import matplotlib.pyplot as plt
import time
import numpy as np
import pandas as pd
import random
import math

def handle_data (raw_data):
    raw_data = raw_data + 30 #30
	#fmt.Printf("%f\n", num)
	#整化 or 定上下界 ： 上下界
    if raw_data > -31 :
        raw_data = -31
    elif raw_data < -156:
        raw_data = -156
    return raw_data

def Xlsx_to_Data(filename):
    BS_RSRP = []
    df = pd.read_excel(filename, sheet_name='Sheet1')
 
    # 遍历数据框中的每一行
    for index, row in df.iterrows():
        rsrp_row = []
        # 打印每一行的数据
        #print(row)
        rsrp_row.append(handle_data(row[0]))
        rsrp_row.append(handle_data(row[1]))
        rsrp_row.append(handle_data(row[2]))
        BS_RSRP.append(rsrp_row)
    #print(BS_RSRP)
    return BS_RSRP

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
    BS1_ALL = []
    BS_ALL_STAMP = []
    BS2_ALL = []
    BS3_ALL = []
    BS1_156_flag = 0
    BS2_156_flag = 0
    BS3_156_flag = 0
    RACH_XnComplete = []
    ERROR_flag = 0
    for line in lines:
        #print(line)
        line = line.split(",")
        if ERROR_flag == 1:
            RACH_XnComplete.append(time_handle(line[2]))
            ERROR_flag = 0
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
        if line[0] == "RSRP:BS1":
            #print(line)
            timestamp_now = time_handle(line[19])
            for i in range(5):
                BS1_ALL.append(float(line[i+1]))
                BS_ALL_STAMP.append(timestamp_now-200*(4-i))
                if line[i+1] == "-156.0000000000" and BS1_156_flag == 0 :
                    BS1_timestamp.append(timestamp_now)
                    BS1_156_flag = 1
                if BS1_156_flag == 1 and line[i+1] != "-156.0000000000":
                    BS1_timestamp.append(timestamp_now)
                    BS1_156_flag = 0
            for i in range(5):
                BS2_ALL.append(float(line[i+7]))
                if line[i+7] == "-156.0000000000" and BS2_156_flag == 0 :
                    BS2_timestamp.append(timestamp_now)
                    BS2_156_flag = 1
                if BS2_156_flag == 1 and line[i+7] != "-156.0000000000":
                    BS2_timestamp.append(timestamp_now)
                    BS2_156_flag = 0
            for i in range(5):
                BS3_ALL.append(float(line[i+13]))
                if line[i+13] == "-156.0000000000" and BS3_156_flag == 0 :
                    BS3_timestamp.append(timestamp_now)
                    BS3_156_flag = 1
                if BS3_156_flag == 1 and line[i+13] != "-156.0000000000":
                    BS3_timestamp.append(timestamp_now)
                    BS3_156_flag = 0
        if line[1].split("||")[0] == "RACH Target_RAN":
            #print(line)
            if len(line) < 4:
                ERROR_flag = 1
            else:
                RACH_XnComplete.append(time_handle(line[3]))
        if line[1] == "Xn Handover Complete":
            RACH_XnComplete.append(time_handle(line[5]))
    f.close()
    return BS1_timestamp,BS2_timestamp,BS3_timestamp,RACH_XnComplete,BS1_ALL,BS2_ALL,BS3_ALL,BS_ALL_STAMP

def SelectUEDuration(file,Start_timestamp,End_timestamp):
    f = open(file, "r",errors='ignore')
    lines = f.readlines()
    ue_timeset = []
    ue_seqset = []
    ran_id = []
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
                    ran_id.append(int((line[4].split(" ")[0]).split("ran")[1]))
    for i in range(len(ue_seqset)-1):
        if ue_seqset[i+1] > ue_seqset[i] + 1:
            Interruption=ue_timeset[i+1]-ue_timeset[i]
            next_ran = ran_id[i+1]
    f.close()
    return Interruption,next_ran

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

def RSRP_TO_PACKETLOSSRATE(RSRP):
	if RSRP > -90 :
		Packet_Loss_Rate_in = 0
	elif RSRP > -110 :
		Packet_Loss_Rate_in = 0.000001 / 20 * (-90 - RSRP)
	else :
		Packet_Loss_Rate_in = 0.000001 + ((math.exp(-110-RSRP)-1)*0.999999)/(math.exp(46)-1)
	return Packet_Loss_Rate_in

if __name__ == "__main__":
    file_dir = "./log/" + "05-09_13-11-31-single-xn.log"  #07-10_17-20-18-single-xn.log   06-17_14-34-51-single-xn.log  07-11_15-24-25-single-xn 06-18_15-43-38-single-xn
    BS1_timestamp,BS2_timestamp,BS3_timestamp,RACH_XnComplete,BS1_ALL,BS2_ALL,BS3_ALL,BS_ALL_STAMP = analyse_dataplane_interrupttime(file_dir)
    print(BS1_timestamp)
    print(RACH_XnComplete)

    Xlsx_to_Data("Path_RSRP_200_Data_1_1.xlsx")
    Interruption = []
    Next_ran = []
    Next_ran.append(0)
    release = 1
    PLR = []
    index = 0
    for i in range(math.floor(len(RACH_XnComplete)/2)) :
        Start_timestamp,End_timestamp = Transfer_SE(BS1_timestamp,BS2_timestamp,BS3_timestamp,RACH_XnComplete,i)
        interruption,next_ran = SelectUEDuration(file_dir,Start_timestamp-2000,End_timestamp+2000)
        Interruption.append(interruption)
        Next_ran.append(next_ran)
    for i in range(len(BS1_ALL)):
        if BS_ALL_STAMP[i] <= RACH_XnComplete[2*index]:
            index = index
        else:
            index = index + 1
        if Next_ran[index] == 0:
            PLR.append(RSRP_TO_PACKETLOSSRATE(BS1_ALL[i]))
        if Next_ran[index] == 1:
            PLR.append(RSRP_TO_PACKETLOSSRATE(BS2_ALL[i]))
        if Next_ran[index] == 2:
            PLR.append(RSRP_TO_PACKETLOSSRATE(BS3_ALL[i]))
        if index >= math.floor(len(RACH_XnComplete)/2):
            index = math.floor(len(RACH_XnComplete)/2)-1

    print(Start_timestamp,End_timestamp)
    print(Interruption)
    print(PLR, sum(PLR)/len(PLR))

    df = pd.DataFrame(PLR)
    df.to_excel('PLR_tempo.xlsx', index=False)