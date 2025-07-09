import math
import time
import numpy as np
import pandas as pd
import matplotlib.pyplot as plt
from scipy.interpolate import griddata
from scipy import stats
import matplotlib.pylab as plt
import random
from tqdm import tqdm
from openpyxl import Workbook
import seaborn as sb
import matplotlib.pyplot as plt



def calclossPS1PS2(Floor,bwidth,stwidth,nb_bx,nb_by,res,h_builds,h_floor,htx,freq,MCL,x,y):
    # calclossPS1PS2 Calculate losses for the Urban microcell scenario (based
    #on the ITU-R U;i path loss model)
    #
    #   loss=calclossPS1PS2(bwidth,stwidth,nb_bx,nb_by,res,h_builds,h_floor,hs,freq,MCL,x,y)
    #   calculates the losses for the Urban Microcell scenario when the antenna
    #   transmitter is located in (x,y).
    #
    #   loss=calclossPS1PS2(bwidth,stwidth,nb_bx,nb_by,res,h_builds,h_floor,hs,freq,MCL)
    #   calculates the losses for the Urban Microcell scenario. In that case,
    #   the antenna transmitter is located at the center of the scenario.
    #
    #   where:
    #       - bwidth: width of buildings [m]
    #       - stwidth: width of streets [m]
    #       - nb_bx: number of buildings in x-direction
    #       - nb_by: number of buildings in y-direction
    #       - res: resolution of the map [m]
    #       - h_builds: [nb_bx X nb_by]-matrix with the building heights [floors] numpy matrix
    #       - h_floor: height of floors
    #       - htx: height of the BS above the ground [m]
    #       - freq: carrier frequency [Hz]
    #       - MCL: minimum coupling losses [dB]
    #       - x: the x-coordinate of the BS [m]
    #       - y: the y-coordinate of the BS [m]
    #
    #   For a map of (40x40) with a resolution of 3 meters, the value of
    #   loss(1,1,1) corresponds to the loss on the ground and (x,y)=(1.5,1.5) as
    #   each value of the matrix corresponds to the center of the 3m x 3m square.
    NAN = 500  # NAN Represents infinity.
    EIRP = 65  # Signal transmission power of antenna [dBm]

    #def processingState(l,j,it_current) :
    #    if l==1 :
    #        print('0 %')
    #       it_current = 0
    #    else :
    #        iter = l/j
    #        if it_current == 0 :
    #            if iter >= 0.1 :
    #                it_current = 10
    #                print('10 %')
    #        if it_current == 10 :
    #            if iter >= 0.2 :
    #                it_current = 20
    #                print('20 %')
    #        if it_current == 20 :
    #            if iter >= 0.3 :
    #                it_current = 30
    #                print('30 %')
    #        if it_current == 30 :
    #            if iter >= 0.4 :
    #                it_current = 40
    #                print('40 %')
    #        if it_current == 40 :
    #            if iter >= 0.5 :
    #                it_current = 50
    #                print('50 %')
    #        if it_current == 50 :
    #            if iter >= 0.6 :
    #                it_current = 60
    #                print('60 %')
    #        if it_current == 60 :
    #            if iter >= 0.7 :
    #                it_current = 70
    #                print('70 %')
    #        if it_current == 70 :
    #            if iter >= 0.8 :
    #                it_current = 80
    #                print('80 %')
    #        if it_current == 80 :
    #            if iter >= 0.9 :
    #                it_current = 90
    #                print('90 %')
    #        if it_current == 90 :
    #            if iter >= 1.0 :
    #                it_current = 100
    #                print('100 %')
    #    return it_current


    def isLoS(j,k,mt,nt,x_p,y_p,x_bs,y_bs,wall_p_x,wall_p_y,slope,ordi,normal_to_x) : #行人index，基站index，行人坐标，基站坐标，墙的信息，斜率截距，是否bs和p在同一 x 上    “视距判断”
        LoS=1
        #print(j,k,mt,nt,x_p,y_p,x_bs,y_bs,slope,ordi,normal_to_x)
        #  Find crossing walls
        if j < mt :  #x方向
            #  East
            wall_sighted_in_x = [(wall_p_x[i] < x_bs and wall_p_x[i] > x_p) for i in range(len(wall_p_x))] # p 墙n bs，记录隔了哪几个墙；
        elif j > mt :
            #  West
            wall_sighted_in_x = [(wall_p_x[i] > x_bs and wall_p_x[i] < x_p) for i in range(len(wall_p_x))] # bs 墙 p，同上；
        else :
            #  Same x-coordinate
            wall_sighted_in_x = [(wall_p_x[i] == x_bs) for i in range(len(wall_p_x))] #bs在哪个墙上吗？

        if k < nt :  #y方向 ，同上
            #  North
            wall_sighted_in_y = [(wall_p_y[i] < y_bs and wall_p_y[i] > y_p) for i in range(len(wall_p_y))]
        elif k > nt :
            #  South
            wall_sighted_in_y = [(wall_p_y[i] > y_bs and wall_p_y[i] < y_p) for i in range(len(wall_p_y))]
        else :
            #  Same y-coordinate
            wall_sighted_in_y = [(wall_p_y[i] == y_bs) for i in range(len(wall_p_y))]

        #print(wall_sighted_in_x,wall_sighted_in_y)
        for w in range(len(wall_p_x)) : #x方向每一堵墙
            if wall_sighted_in_x[w] : #x方向，隔了这个墙
                x_wall_x = wall_p_x[w]
                y_wall_x = slope*x_wall_x + ordi #X方向被隔着的墙的xy坐标，坐标在建筑上，y在建筑区域则非直视；反之直视；
                build_y = []
                for z in range(int(len(wall_p_y)/2)) : #判断这个y是建筑还是街道；
                    if y_wall_x >= wall_p_y[2*z] and  y_wall_x <= wall_p_y[2*z+1] :
                        build_y.append(1)
                    else :
                        build_y.append(0)
                if np.array(build_y).sum() != 0 :
                    LoS = 0
                    return LoS

        for w in range(len(wall_p_y)) :
            if wall_sighted_in_y[w] :
                y_wall_y = wall_p_y[w]
                if normal_to_x == 0 :
                    x_wall_y = (y_wall_y - ordi)/slope
                else :
                    x_wall_y = x_p
                build_x = []
                for z in range(int(len(wall_p_x)/2)) :
                    if x_wall_y >= wall_p_x[2*z] and  x_wall_y <= wall_p_x[2*z+1] :
                        build_x.append(1)
                    else :
                        build_x.append(0)
                if np.array(build_x).sum() != 0 :
                    LoS = 0
                    return LoS
        return LoS

    nargin = 12
    if (nargin == 10 or nargin == 12):
        if bwidth <= 0:
            print('Invalid width of buildings: it must be greater than zero')
        if stwidth <= 0:
            print('Invalid width of streets: it must be greater than zero')
        if (nb_bx <= 0 or nb_by <= 0 or nb_bx != nb_by):
            print('Invalid number of buildings in the scenario')
        if res <= 0:
            print('Invalid resolution: it must be greater than zero.')
        h_builds_rows, h_builds_cols = h_builds.shape
        if (h_builds_rows != nb_bx or h_builds_cols != nb_by):
            print('Error in building heights matrix dimension: it must be [nb_bx,nb_by].')
        if h_floor <= 0:
            print('Invalid height of floors: it must be greater than zero.')
        if htx <= 0:
            print('Invalid height of the transmitter.')
    else:
        print('Invalid number of input arguments.')

    #  DEFAULT PARAMETERS
    hm = 1.5
    htx_eff = htx - 1.0
    hrx_eff = hm - 1.0

    #  Size of the map: (m x n)
    m = int(np.ceil((nb_bx*(bwidth+stwidth)+stwidth)/res))
    n = int(np.ceil((nb_by*(bwidth+stwidth)+stwidth)/res))
    #print(m,n)
    maps = np.ones((m,n))
    map_h = np.zeros((m,n))
    limit_map = [nb_bx*(bwidth+stwidth)+stwidth, nb_by*(bwidth+stwidth)+stwidth]

    #  Antenna transmitter location
    if nargin !=12 :
        mt = np.ceil(m/2)
        nt = np.ceil(n/2)
    else :
        mt= np.round(x/res)
        nt= np.round(y/res)

    #  Maximum number of floors
    maxnbfloors = np.max(h_builds)

    #  Streets in x-direction
    j = [(i-1)*res for i in range(1,m+1)]
    for i in range(len(j)) :
        j[i] = ((j[i]*10)%((bwidth+stwidth)*10))/10
        if j[i] < stwidth :
            maps[i,:] = 0

    #  Streets in y-direction
    j = [(i-1)*res for i in range(1,n+1)]
    for i in range(len(j)) :
        j[i] = ((j[i]*10)%((bwidth+stwidth)*10))/10
        if j[i] < stwidth :
            maps[:,i] = 0  #街道上值为0
    #print(maps)
    map_axis = maps

    #  Map with heights
    #map_h = maps
    for r in range(1,nb_bx+1) :
        i = [(x-1)*res for x in range(1,m+1)]
        #print(i)
        for x in range(len(i)) :  #X坐标轴上 第r栋建筑  ，有建筑标1，反之0
            if ((i[x] >= stwidth+(r-1)*(stwidth + bwidth)) and (i[x] < r*(stwidth + bwidth))) :
                i[x] = 1
            else :
                i[x] = 0
        #print(i)
        for s in range(1,nb_by+1) :   
            j = [(x-1)*res for x in range(1,n+1)]
            #print(j)
            for x in range(len(j)) :  #y坐标轴上 第s栋建筑
                if ((j[x] >= stwidth+(s-1)*(stwidth + bwidth)) and (j[x] < s*(stwidth + bwidth))) :
                    j[x] = 1
                else :
                    j[x] = 0
            #print(np.transpose(np.array(j).reshape(1,len(j))),np.array(i).reshape(1,len(i)))
            #map_hplus = [[i[x]*j[y] for x in range(1,m+1)]for y in range(1,n+1)]
            #map_hplus = np.outer(np.array(j).reshape(1,len(j)),np.array(i).reshape(1,len(i)))
            #map_hplus = np.matmul(np.transpose(np.array(j).reshape(1,len(j))),np.array(i).reshape(1,len(i)))
            map_hplus = []
            for f in i:
                map1 = []
                for k in j:
                    map1.append(f*k)
                #print(map1)
                map_hplus.append(map1)
            #print(np.array(map_hplus))
            map_hplus = np.transpose(np.array(map_hplus))  # 二维地图上建筑标1，街道标0
            #print(map_h.shape,h_builds[s-1,r-1],h_floor,map_hplus,map_hplus.shape)
            map_h = map_h + h_builds[s-1,r-1]*h_floor*map_hplus #标海拔的地图二维
            #print(map_h)

    #  Position of building walls in x-direction and y-direction
    wp = 0
    #print(nb_bx) 
    real_wall_points_x = np.transpose(np.zeros((1,2*nb_bx)))  #x坐标轴上建筑的左右墙；
    for r in range(1,nb_bx+1) :
        #print(wp,real_wall_points_x.shape)
        real_wall_points_x[wp] = (stwidth+(r-1)*(stwidth + bwidth))
        real_wall_points_x[wp+1] = (r*(stwidth + bwidth))
        wp = wp+2
    wp = 0
    real_wall_points_y = np.transpose(np.zeros((1,2*nb_by)))
    for r in range(1,nb_by+1) :
        real_wall_points_y[wp] = (stwidth+(r-1)*(stwidth + bwidth))
        real_wall_points_y[wp+1] = (r*(stwidth + bwidth))
        wp = wp+2
    #print(np.array([0]),real_wall_points_x.reshape(real_wall_points_x.shape[0]*real_wall_points_x.shape[1]),np.array([limit_map[0]]))
    street_limits_x = np.concatenate([np.array([0]),np.concatenate([real_wall_points_x.reshape(real_wall_points_x.shape[0]*real_wall_points_x.shape[1]),np.array([limit_map[0]])],axis=0)],axis=0)  #加上街道边界左右界
    street_limits_y = np.concatenate([np.array([0]),np.concatenate([real_wall_points_y.reshape(real_wall_points_y.shape[0]*real_wall_points_y.shape[1]),np.array([limit_map[1]])],axis=0)],axis=0)
    #print(street_limits_x)
    #  1) UMi: check if the transmission antenna is located in the street
    #print(mt,nt,int(mt),int(nt))
    if maps[int(mt),int(nt)] == 1 :
        print('The transmission antenna must be located in the street')  #must be located in the street!

    #  Initialize output variable
    loss = np.full((m,n), np.nan)  #初始化

    #  OUTDOOR PS#1
    #print('Micro - Outdoor calculations - PS#1')

    #  The index of each vector are ordered in the following directions:
    #  1 = x-positive (East)
    #  2 = y-positive (North)
    #  3 = x-negative (West)
    #  4 = y-negative (South)

    #  This model ditinguishes the main street, perpedicular streets
    #  and parallel streets
    [j,k] = np.array(np.where(maps == 0))  #jk是坐标集合 值为0的 街道点
    #print([j,k])

    x_bs = (mt)*res #基站坐标，真实！
    y_bs = (nt)*res
    #x_bs = (mt - 0.5)*res
    #y_bs = (nt - 0.5)*res

    #  Find direction and center of the street where the BS is located
    bs_street_x = []
    for i in range(int(len(street_limits_x)/2)) :  #在x坐标轴第i个街道
        #print(street_limits_x)
        if x_bs >= street_limits_x[2*i] and x_bs < street_limits_x[2*i+1] :
            bs_street_x.append(i)
    bs_street_y = []
    for i in range(int(len(street_limits_y)/2)) :  #在y坐标轴第i个街道
        #print(street_limits_y)
        if y_bs >= street_limits_y[2*i] and y_bs < street_limits_y[2*i+1] :
            bs_street_y.append(i)

    bs_str_centre = []
    #print(bs_street_x,bs_street_y)
    if len(bs_street_x) > 0 :
        if len(bs_street_y) > 0 :
            #  The transmitter is located in a intersection of perpendicular
            #  streets
            bs_str = 2  #在十字路口
            bs_str_centre.append((street_limits_x[2*bs_street_x[0]] + street_limits_x[2*bs_street_x[0]+1])/2)  #bs所在街道的中值，center
            bs_str_centre.append((street_limits_y[2*bs_street_y[0]] + street_limits_y[2*bs_street_y[0]+1])/2)
        else : 
            #  The transmitter is located in a street in y-direction
            bs_str = 1  #在纵向街道上
            bs_str_centre.append((street_limits_x[2*bs_street_x[0]] + street_limits_x[2*bs_street_x[0]+1])/2)
            bs_str_centre.append(-1)
    else :
        #  The transmitter is located in a street in x-direction
        bs_str = 0   #在横向街道上
        bs_str_centre.append(-1)
        bs_str_centre.append((street_limits_y[2*bs_street_y[0]] + street_limits_y[2*bs_street_y[0]+1])/2)

    #it_current = 0
    for l in range(len(j)) : #街道的每一个点
    #if maps[j_p//res,k_p//res] == 0 and Floor == 1:

        #print(l,len(k),it_current)
        #it_current = processingState(l+0,len(j),it_current)
        #print(j[l],k[l]) 
        x_p = (j[l])*res #点坐标
        y_p = (k[l])*res
        #x_p = (j_p // res)*res
        #y_p = (k_p // res)*res
        #x_p = (j[l] - 0.5)*res
        #y_p = (k[l] - 0.5)*res
        #print(x_p,y_p)

        #生成LOS和NLOS下的随机浮动值；分别为5.8和8.7
        #生成均值为0，标准差为1:N(μ，σ2) = N(0，1)的正态分布的1个随机变量，0 1 1
        LOS_scale = stats.norm.rvs(loc=0, scale=5.8, size=1)
        NLOS_scale = stats.norm.rvs(loc=0, scale=8.7, size=1)

        #  Find the street where the rx is located
        rx_street_x = []
        for i in range(int(len(street_limits_x)/2)) : #计算每一个街道点上的人的街道信息；
            #print(street_limits_x)
            if x_p >= street_limits_x[2*i] and x_p < street_limits_x[2*i+1] :
                rx_street_x.append(i)
        rx_street_y = []
        for i in range(int(len(street_limits_y)/2)) :

            if y_p >= street_limits_y[2*i] and y_p < street_limits_y[2*i+1] :
                rx_street_y.append(i)

        #print(rx_street_x,rx_street_y)
        rx_str_centre = []
        if len(rx_street_x) > 0 :   #判断在十字路口，纵街还是横街上，以及中心位置
            if len(rx_street_y) > 0 :
                #  The receiver is located in perpendicular street
                rx_str = 2
                rx_str_centre.append((street_limits_x[2*rx_street_x[0]] + street_limits_x[2*rx_street_x[0]+1])/2)
                rx_str_centre.append((street_limits_y[2*rx_street_y[0]] + street_limits_y[2*rx_street_y[0]+1])/2)
            else :
                #  The receiver is located in a street in y-direction
                rx_str = 1
                rx_str_centre.append((street_limits_x[2*rx_street_x[0]] + street_limits_x[2*rx_street_x[0]+1])/2)
                rx_str_centre.append(-1)
        else :
            #  The receiver is located in a street in y-direction
            rx_str = 0
            rx_str_centre.append(-1)
            rx_str_centre.append((street_limits_y[2*rx_street_y[0]] + street_limits_y[2*rx_street_y[0]+1])/2)

        #  Rect from BS to point
        if (x_p - x_bs) != 0 :  #计算斜率和截距，如果同x坐标，直接是直视距离；
            slope = (y_p - y_bs)/(x_p - x_bs)
            ordi = (x_p*y_bs - x_bs*y_p)/(x_p-x_bs)
            normal_to_x = 0
        else :
            #  Rect perpendicular to x-axis
            normal_to_x = 1

        #  Check if the receiver is in LoS
        isInLoS = isLoS(j[l],k[l],mt,nt,x_p,y_p,x_bs,y_bs,real_wall_points_x,real_wall_points_y,slope,ordi,normal_to_x)
        #print(isInLoS)
        if isInLoS :  #直视
            #  Receiver in the main street (LoS)
            d1 = math.sqrt((x_p-x_bs)**2 + (y_p-y_bs)**2)
            if d1 == 0 :
                Llos = 0
            else:
                Llos = 40*(math.log10(d1)) + 7.8 - 18*(math.log10(htx_eff)) - 18*(math.log10(hrx_eff)) + 2*(math.log10(freq/1e6))

            LOS_scale = stats.norm.rvs(loc=0, scale=5.8, size=1)
            loss[k[l],j[l]] = max(Llos,MCL) + LOS_scale
        else :
            #  Check if the receiver is either in a perpendicular or a paralel
            #  street
            parallel = False
            if (rx_str == bs_str) and (rx_str < 2) : #标记属于哪种街道的；
                parallel = True

            if parallel :
                NLOS_scale = stats.norm.rvs(loc=0, scale=8.7, size=1)
                loss[k[l],j[l]] = NAN + NLOS_scale
                #np.nan
            else :
                if (rx_str * bs_str) < 4 :
                    if bs_str == 0 :
                        d1 = abs(x_bs-rx_str_centre[0])
                        d2 = abs(y_p-bs_str_centre[1])
                    if bs_str == 1 :
                        d1 = abs(y_bs-rx_str_centre[1])
                        d2 = abs(x_p-bs_str_centre[0])
                    if bs_str == 2 :
                        if rx_str == 0 :
                            d1 = abs(y_bs-rx_str_centre[1])
                            d2 = abs(x_p-bs_str_centre[0])
                        else :
                            d1 = abs(x_bs-rx_str_centre[0])
                            d2 = abs(y_p-bs_str_centre[1])
                    Llos1 = 40*(math.log10(d1)) + 7.8 - 18*(math.log10(htx_eff)) - 18*(math.log10(hrx_eff)) + 2*(math.log10(freq/1e6))
                    Llos2 = 40*(math.log10(d2)) + 7.8 - 18*(math.log10(hrx_eff)) - 18*(math.log10(hrx_eff)) + 2*(math.log10(freq/1e6))
                    nj1 = max(2.8-0.0024*d1,1.84)
                    nj2 = max(2.8-0.0024*d2,1.84)
                    PL1 = Llos1 + 17.9 - 12.5*nj1 + 10*nj1*(math.log10(d2)) + 3*(math.log10(freq/1e6))
                    PL2 = Llos2 + 17.9 - 12.5*nj2 + 10*nj2*(math.log10(d1)) + 3*(math.log10(freq/1e6))
                    PL = min(PL1,PL2)                
                else :
                    d1_1 = abs(x_bs-rx_str_centre[0])
                    d2_1 = abs(y_p-bs_str_centre[1])
                    Llos1_1 = 40*(math.log10(d1_1)) + 7.8 - 18*(math.log10(htx_eff)) - 18*(math.log10(hrx_eff)) + 2*(math.log10(freq/1e6))
                    Llos2_1 = 40*(math.log10(d2_1)) + 7.8 - 18*(math.log10(hrx_eff)) - 18*(math.log10(hrx_eff)) + 2*(math.log10(freq/1e6))
                    nj1_1 = max(2.8-0.0024*d1_1,1.84)
                    nj2_1 = max(2.8-0.0024*d2_1,1.84)
                    PL1_1 = Llos1_1 + 17.9 - 12.5*nj1_1 + 10*nj1_1*(math.log10(d2_1)) + 3*(math.log10(freq/1e6))
                    PL2_1 = Llos2_1 + 17.9 - 12.5*nj2_1 + 10*nj2_1*(math.log10(d1_1)) + 3*(math.log10(freq/1e6))
                    PL_1 = min(PL1_1,PL2_1)

                    d1_2 = abs(y_bs-rx_str_centre[1])
                    d2_2 = abs(x_p-bs_str_centre[0])
                    Llos1_2 = 40*(math.log10(d1_2)) + 7.8 - 18*(math.log10(htx_eff)) - 18*(math.log10(hrx_eff)) + 2*(math.log10(freq/1e6))
                    Llos2_2 = 40*(math.log10(d2_2)) + 7.8 - 18*(math.log10(hrx_eff)) - 18*(math.log10(hrx_eff)) + 2*(math.log10(freq/1e6))
                    nj1_2 = max(2.8-0.0024*d1_2,1.84)
                    nj2_2 = max(2.8-0.0024*d2_1,1.84)
                    PL1_2 = Llos1_2 + 17.9 - 12.5*nj1_2 + 10*nj1_2*(math.log10(d2_2)) + 3*(math.log10(freq/1e6))
                    PL2_2 = Llos2_2 + 17.9 - 12.5*nj2_2 + 10*nj2_2*(math.log10(d1_2)) + 3*(math.log10(freq/1e6))
                    PL_2 = min(PL1_2,PL2_2)

                    PL = min(PL_1,PL_2)
                NLOS_scale = stats.norm.rvs(loc=0, scale=8.7, size=1)
                loss[k[l],j[l]] = max(PL,MCL) + NLOS_scale
        #print(j[l],k[l],loss[1,k[l],j[l]],isInLoS,parallel,rx_str, bs_str)
    #loss[2:,:,:] = NAN
    #np.nan
    #else :
    #    loss = NAN

    #  INDOOR: only for buiding in LoS
    #print('Micro - Indoor calculations - PS#2')

    #  The index of each vector are ordered in the following directions:
    #  1 = x-positive (East)
    #  2 = y-positive (North)
    #  3 = x-negative (West)
    #  4 = y-negative (South)

    [j,k] = np.array(np.where(maps == 1))
 
    normal_vect = [[0,0],[1,0],[0,1],[-1,0],[0,-1]]

    #it_current = 0
    for l in range(len(j)) :
    #if maps[j_p//res,k_p//res] == 1 :
        #it_current=processingState(l+1,len(j),it_current)

        x_p = (j[l])*res
        y_p = (k[l])*res
        #x_p = (j_p // res)*res
        #y_p = (k_p // res)*res
        #x_p = (j[l] - 0.5)*res
        #y_p = (k[l] - 0.5)*res
        LOS_scale = stats.norm.rvs(loc=0, scale=5.8, size=1)
        NLOS_scale = stats.norm.rvs(loc=0, scale=8.7, size=1)

        #  Find the build where the rx is located
        rx_build_x = []
        for i in range(int(len(real_wall_points_x)/2)) :
            if x_p >= real_wall_points_x[2*i] and x_p <= real_wall_points_x[2*i+1] :
                rx_build_x.append(i)
        rx_build_y = []
        for i in range(int(len(real_wall_points_y)/2)) :
            if y_p >= real_wall_points_y[2*i] and y_p <= real_wall_points_y[2*i+1] :
                rx_build_y.append(i)

        #  Check if this building is in the street of the transmitter
        isInLoS = 0
        wall_sighted = np.zeros((5,1))   # street vs building forward NSWE 4bit
        wall_points = -1*np.ones((5,2))  # position of the wall
        d_to_wall = -1*np.ones((5,1))    # distance to the wall
        if len(bs_street_x) > 0 :
            if (rx_build_x[0] == bs_street_x[0]) or (rx_build_x[0] == bs_street_x[0] - 1) : # street up or down drom the building
                isInLoS = 1
                if (rx_build_x[0] + 1 == bs_street_x[0]) :
                    wall_sighted[1] = 1
                    #print([real_wall_points_x[2*rx_build_x[0] + 1], y_p])
                    wall_points[1,:] = [int(real_wall_points_x[2*rx_build_x[0] + 1]), y_p]
                    d_to_wall[1] = abs(x_p - real_wall_points_x[2*rx_build_x[0] + 1])
                else :
                    wall_sighted[3] = 1
                    #print([real_wall_points_x[2*rx_build_x[0]], y_p])
                    wall_points[3,:] = [int(real_wall_points_x[2*rx_build_x[0]]), y_p]
                    d_to_wall[3] = abs(x_p - real_wall_points_x[2*rx_build_x[0]])  # +1 -1标号可能有问题，再检查；
        if len(bs_street_y) > 0 :
            if (rx_build_y[0] == bs_street_y[0]) or (rx_build_y[0] == bs_street_y[0] - 1) :
                isInLoS = 1
                if (rx_build_y[0] + 1 == bs_street_y[0]) :
                    wall_sighted[2] = 1
                    wall_points[2,:] = [x_p,int(real_wall_points_y[2*rx_build_y[0] + 1])]
                    d_to_wall[2] = abs(y_p - real_wall_points_y[2*rx_build_y[0] + 1])
                else :
                    wall_sighted[4] = 1
                    wall_points[4,:] = [x_p,int(real_wall_points_y[2*rx_build_y[0]])]
                    d_to_wall[4] = abs(y_p - real_wall_points_y[2*rx_build_y[0]])

        if isInLoS :
            if sum(wall_sighted) < 2 :
                num_opt = 1
            else :
                #  Two options because the transmitter is in a intersection
                num_opt = 2

            if num_opt == 1 :
                wall_sighted = wall_sighted.reshape(wall_sighted.shape[0]*wall_sighted.shape[1])
                idx = np.where(wall_sighted == 1)
                idx = int(idx[0])
                d_in = d_to_wall[idx]
                #print(idx, wall_sighted, wall_points)
                d_out = math.sqrt((x_bs - wall_points[idx,0])**2 + (y_bs - wall_points[idx,1])**2)

                #print(type(idx),type(normal_vect),)
                #print(np.array(normal_vect),normal_vect)
                normal_vect = np.array(normal_vect)
                ps_v1_v2 = ((x_bs-wall_points[idx,0])*normal_vect[idx][0] + (y_bs-wall_points[idx,1])*normal_vect[idx][1])
                normv = math.sqrt((x_bs-wall_points[idx,0])**2 + (y_bs-wall_points[idx,1])**2)
                tetha = math.pi/2 - math.acos(ps_v1_v2/normv)

                Lth = 9.82 + 5.98*(math.log10(freq/1e6)) + 15*(1-math.sin(tetha))**2
                Lin = 0.5*d_in

                #for nf in range(maxnbfloors) :
                if Floor <= map_h[j[l],k[l]]/h_floor :   # nf or nf + 1 ??
                    hrx_floor = h_floor*(Floor - 1) + hrx_eff  #nf = Floor or Floor-1??
                    Lout = 40*(math.log10(d_out + d_in)) + 7.8 - 18*(math.log10(htx_eff)) - 18*(math.log10(hrx_floor)) + 2*(math.log10(freq/1e6))
                    Lout = max(Lout,MCL)
                    PL = Lout + Lth + Lin
                    LOS_scale = stats.norm.rvs(loc=0, scale=5.8, size=1)
                    loss[k[l],j[l]] = max(PL,MCL) + LOS_scale
                    #if nf == 0 :
                        #print(j[l],k[l],loss[1,k[l],j[l]])
                else :
                    LOS_scale = stats.norm.rvs(loc=0, scale=5.8, size=1)
                    loss[k[l],j[l]] = NAN + LOS_scale
                        #np.nan
            else :

                #  First option
                idx = list(wall_sighted).index(1)
                #print(type(idx))
                #idx = int(idx[0])
                #idx = int(idx)
                #print(type(idx))

                d_in1 = d_to_wall[idx]
                d_out1 = math.sqrt((x_bs - wall_points[idx,0])**2 + (y_bs - wall_points[idx,1])**2)

                ps_v1_v2 = ((x_bs-wall_points[idx,0])*normal_vect[idx][0] + (y_bs-wall_points[idx,1])*normal_vect[idx][1] )
                normv =math.sqrt((x_bs-wall_points[idx,0])**2 + (y_bs-wall_points[idx,1])**2)
                tetha = math.pi/2 - math.acos(ps_v1_v2/normv)

                Lth1 = 9.82 + 5.98*(math.log10(freq/1e6)) + 15*(1-math.sin(tetha))**2
                Lin1 = 0.5*d_in1

                #  Second option
                idx = len(list(wall_sighted)) - 1 - list(wall_sighted)[::-1].index(1)
                #idx = int(idx[0])
                #list(wall_sighted).LastIndexOf(1)

                d_in2 = d_to_wall[idx]
                d_out2 = math.sqrt((x_bs - wall_points[idx,0])**2 + (y_bs - wall_points[idx,1])**2 )

                ps_v1_v2 = ((x_bs-wall_points[idx,0])*normal_vect[idx][0] + (y_bs-wall_points[idx,1])*normal_vect[idx][1] )
                normv = math.sqrt((x_bs-wall_points[idx,0])**2 + (y_bs-wall_points[idx,1])**2)
                tetha = math.pi/2 - math.acos(ps_v1_v2/normv)

                Lth2 = 9.82 + 5.98*(math.log10(freq/1e6)) + 15*(1-math.sin(tetha))**2
                Lin2 = 0.5*d_in2

                #for nf in range(maxnbfloors) :
                if Floor <= map_h[j[l],k[l]]/h_floor :  # nf or nf + 1
                    hrx_floor = h_floor*(Floor - 1) + hrx_eff

                    Lout1 = 40*(math.log10(d_out1 + d_in1)) + 7.8 - 18*(math.log10(htx_eff)) - 18*(math.log10(hrx_floor)) + 2*(math.log10(freq/1e6))
                    Lout1 = max(Lout1,MCL)
                    PL1 = Lout1 + Lth1 + Lin1

                    Lout2 = 40*(math.log10(d_out2 + d_in2)) + 7.8 - 18*(math.log10(htx_eff)) - 18*(math.log10(hrx_floor)) + 2*(math.log10(freq/1e6))
                    Lout2 = max(Lout2,MCL)
                    PL2 = Lout2 + Lth2 + Lin2

                    PL = min(PL1,PL2)

                    LOS_scale = stats.norm.rvs(loc=0, scale=5.8, size=1)
                    loss[k[l],j[l]] = max(PL,MCL) + LOS_scale
                    #if nf == 0 :
                        #print(j[l],k[l],loss[1,k[l],j[l]])
                else :
                    LOS_scale = stats.norm.rvs(loc=0, scale=5.8, size=1)
                    loss[k[l],j[l]] = NAN + LOS_scale
                        #np.nan
        else :
            NLOS_scale = stats.norm.rvs(loc=0, scale=8.7, size=1)
            loss[k[l],j[l]] = NAN + NLOS_scale
            #np.nan
        #print(j[l],k[l],loss[1,k[l],j[l]])
    
    #print(loss)

    #x_r = []
    #for i in range(maps.shape[0]):
    #    x_r.append(res*i)
    #x_r = np.array(x_r).reshape(1,len(x_r))
    #print(x_r, maps.shape[0])
    #x_r = np.repeat(x_r, maps.shape[1], axis=0)
    #y_r = []
    #for i in range(maps.shape[1]):
    #    y_r.append(res*i)
    #y_r.reverse()
    #print(x_r,y_r)
    #y_r = np.array(y_r).reshape(1,len(y_r)).T
    #print(x_r,y_r)
    #y_r = np.repeat(y_r, maps.shape[0], axis=1)
    #print(x_r,y_r)
        
    #print(loss)
    #df = pd.DataFrame(loss[1,:,:])
    #df = pd.DataFrame(map_h)
    #df.to_excel('output.xlsx',index = False)

    #c = plt.pcolormesh(x_r, y_r, loss[1,:,:], cmap='PuBu')
    #plt.colorbar(c, label='Sig_BS1')
    #plt.xlabel('X')
    #plt.ylabel('y')
    #plt.savefig('heatmap.tif', dpi=300)
    #plt.show()
    loss = [[EIRP-i-80 for i in row] for row in loss]
    return loss

def Path_Fitting(Vel,Start_x,Start_y,LOSS_Lib,res,a,Fn_L3,N,Path_NUM):
    RSRP_sum = []
    RSRP_L13 = []
    X_list = []
    Y_list = []
    First_Data = True

    Y = Start_x/res
    X = Start_y/res
    X_list.append(X)
    Y_list.append(Y)
    if Path_NUM == 1:
        while X < 290/res:
            X = X + Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
        while Y < 386/res:
            Y = Y + Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
        while X > 90/res:
            X = X - Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
    if Path_NUM == 2:
        while Y > 110/res :
            Y = Y - Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
        while X < 18/res:
            X = X + Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
        while Y < 150/res:
            Y = Y + Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
    if Path_NUM == 3:
        while Y < 300/res :
            Y = Y + Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
        while X > 100/res :
            X = X - Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
    if Path_NUM == 4:
        while X < 250/res :
            X = X + Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
    if Path_NUM == 5:
        while Y < 385/res :
            Y = Y + Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
        while X > 250/res :
            X = X - Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
    if Path_NUM == 6:
        while  Y > 10/res :
            Y = Y - Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
        while X < 50/res :
            X = X + Vel*0.04
            X_list.append(X)
            Y_list.append(Y)
            
    Random_Start = random.randrange(0, 9, 1)
    Random_Step = random.randrange(0, 9, 1)
    Random_index = Random_Start
    for i in range(len(X_list)):
        RSRP = []  #40ms的一组三点值
        for j in range(N):
            rsrp = LOSS_Lib[Random_index, j, int(X_list[i]), int(Y_list[i])]
            RSRP.append(rsrp)
        #随机对数正态分布噪声的加入；
        #print(RSRP)
        RSRP_sum.append(RSRP)  #200ms的3x5值
        Random_index = (Random_index + Random_Step) % 10
        #每200ms的L1滤波 // 或者说为每产生5个数据；
        if i % 5 == 4 :
            RSRP_sum = np.array(RSRP_sum)
            RSRP_sum = [[math.pow(10,i/10) for i in row] for row in RSRP_sum]
            RSRP_sum = np.array(RSRP_sum)
            RSRP_avg = RSRP_sum.mean(axis=0)  # L1滤波
            #RSRP_avg = np.sum([math.pow(10,RSRP_sum[i,:]/10) for i in range(len(RSRP_sum))])/5
            RSRP_sum = []
            #L3滤波_200ms // L1后的L3滤波 // 其中包含滤波系数 a；
            #Fn_L3 = [(1-a)*Fn_L3[i] + a*10*(math.log10(abs(RSRP_avg[i]))) for i in range(N)] #表达式有误，想清楚是log前的均值还是后的均值，然后考虑此处表达式怎么写；包括avg
            if First_Data :
                Fn_L3 = [10*(math.log10(RSRP_avg[i])) for i in range(N)]
                First_Data = False
            else:
                Fn_L3 = [(1-a)*Fn_L3[i] + a*10*(math.log10(RSRP_avg[i])) for i in range(N)]
            #print(RSRP_avg,Fn_L3)
            RSRP_L13.append(Fn_L3)
        #以一定的时间间隔上报RAN // 由RAN指定时间间隔；收发包的代码实现；
    return RSRP_L13

def Plot_RSRP(LOSS_Lib,maps,maps_index):
    mycolor = sb.diverging_palette(-31,-156,n=126)
    cmap = sb.diverging_palette(220,10,as_cmap=True)
    RSRP_ALL = []
    for i in range(LOSS_Lib.shape[1]):
        Random_Start = random.randrange(0, 9, 1)
        Random_Step = random.randrange(0, 9, 1)
        Random_index = Random_Start
        RSRP = []
        for j in range(LOSS_Lib.shape[2]):
            rsrp_1 = []
            for k in range(LOSS_Lib.shape[3]):
                rsrp_2 = LOSS_Lib[Random_index, i,LOSS_Lib.shape[2] - j-1, k]
                if rsrp_2 < -156 :
                    rsrp_2 = -156
                if rsrp_2 > -31 :
                    rsrp_2 = -31
                rsrp_1.append(rsrp_2)
                Random_index = (Random_index + Random_Step) % 10
            RSRP.append(rsrp_1)
        RSRP_ALL.append(RSRP)
    
    RSRP_MAX = []
    RSRP_MAX_INDEX = []
    for j in range(LOSS_Lib.shape[2]):
        rsrp_max = []
        rsrp_max_index = []
        for k in range(LOSS_Lib.shape[3]):
            rsrp_3 = max(RSRP_ALL[0][j][k],RSRP_ALL[1][j][k],RSRP_ALL[2][j][k])
            rsrp_index = list([RSRP_ALL[0][j][k],RSRP_ALL[1][j][k],RSRP_ALL[2][j][k]]).index(rsrp_3)
            rsrp_max.append(rsrp_3)
            if maps_index[j][k] == 0:
                rsrp_max_index.append(rsrp_index)
            else:
                rsrp_max_index.append(4)
        RSRP_MAX.append(rsrp_max)
        RSRP_MAX_INDEX.append(rsrp_max_index)
    
    dx = 400/LOSS_Lib.shape[2]
    x_label_plot = []
    for j in range(LOSS_Lib.shape[2]):
        #print(dx,dx*j)
        x_label_plot.append(str(dx*j))
    
    y_label_plot = ['0','100','200','300','400']

    fig = plt.figure(figsize=(38, 7))
    gs = fig.add_gridspec(1, 4)

    ax1 = fig.add_subplot(gs[0, 0])
    
    # im=ax1.imshow(RSRP_ALL[0],cmap=cmap, aspect='auto')#cmap='Greys'),设置热力图颜色，一般默认为蓝黄色，aspect='auto'热力块大小随着画布大小自动变化，如果不设置的话为方框。
    # cbar=ax1.figure.colorbar(im, ax=ax1)
    # cbar.ax.set_ylabel('score', rotation=90, va="bottom",fontsize=10)

    sb.heatmap(RSRP_ALL[0], ax=ax1,cmap=cmap,yticklabels="(dBm)")

    ax1.set_xticks([0,500,1000,1500,2000])
    ax1.set_yticks([0,500,1000,1500,2000])
    ax1.set_xticklabels(y_label_plot)
    ax1.set_yticklabels(y_label_plot[::-1])
    ax1.set_xlabel('(a) RAN 0',fontsize=16)
    #sb.heatmap(RSRP,cmap=mycolor)
    ax2 = fig.add_subplot(gs[0, 1])
    sb.heatmap(RSRP_ALL[1], ax=ax2,cmap=cmap)
    ax2.set_xticks([0,500,1000,1500,2000])
    ax2.set_yticks([0,500,1000,1500,2000])
    ax2.set_xticklabels(y_label_plot)
    ax2.set_yticklabels(y_label_plot[::-1])
    ax2.set_xlabel('(b) RAN 1',fontsize=16)

    ax3 = fig.add_subplot(gs[0, 2])
    sb.heatmap(RSRP_ALL[2], ax=ax3,cmap=cmap)
    ax3.set_xticks([0,500,1000,1500,2000])
    ax3.set_yticks([0,500,1000,1500,2000])
    ax3.set_xticklabels(y_label_plot)
    ax3.set_yticklabels(y_label_plot[::-1])
    ax3.set_xlabel('(c) RAN 2',fontsize=16)
    #plt.savefig("BS_"+str(i)+".png")
    ax4 = fig.add_subplot(gs[0, 3])
    sb.heatmap(RSRP_MAX, ax=ax4,cmap=cmap)
    ax4.set_xticks([0,500,1000,1500,2000])
    ax4.set_yticks([0,500,1000,1500,2000])
    ax4.set_xticklabels(y_label_plot)
    ax4.set_yticklabels(y_label_plot[::-1])
    ax4.set_xlabel('(d) Best RSRP',fontsize=16)

    plt.savefig("BS.png")

    fig,ax = plt.subplots(figsize=(9, 7))
    sb.heatmap(RSRP_MAX_INDEX, ax=ax,cmap="PuBu")
    ax.set_xticks([0,500,1000,1500,2000])
    ax.set_yticks([0,500,1000,1500,2000])
    ax.set_xticklabels(y_label_plot)
    ax.set_yticklabels(y_label_plot[::-1])
    plt.savefig("BS_INDEX.png")

    [j,k] = np.array(np.where(maps == 0))

    RSRP_80 = 0
    RSRP_90 = 0
    RSRP_100 = 0
    RSRP_110 = 0
    RSRP_130 = 0
    RSRP_150 = 0
    for i in range(len(j)):
        RSRP_ABOVE = RSRP_MAX[j[i]][k[i]]
        if RSRP_ABOVE >= -80 :
            RSRP_80 = RSRP_80 +1
        elif RSRP_ABOVE >= -90 :
            RSRP_90 = RSRP_90 +1
        elif RSRP_ABOVE >= -100 :
            RSRP_100 = RSRP_100 +1
        elif RSRP_ABOVE >= -110 :
            RSRP_110 = RSRP_110 +1
        elif RSRP_ABOVE >= -130 :
            RSRP_130 = RSRP_130 +1
        else :
            RSRP_150 = RSRP_150 +1
    RSRP_together = RSRP_80 + RSRP_90 + RSRP_100 + RSRP_110 + RSRP_130 + RSRP_150
    RSRP_80 = RSRP_80/RSRP_together
    RSRP_90 = RSRP_90/RSRP_together
    RSRP_100 = RSRP_100/RSRP_together
    RSRP_110 = RSRP_110/RSRP_together
    RSRP_130 = RSRP_130/RSRP_together
    RSRP_150 = RSRP_150/RSRP_together
    print(RSRP_80, RSRP_90, RSRP_100, RSRP_110, RSRP_130, RSRP_150)


bwidth = 80
stwidth = 16
nb_bx = 4
nb_by = 4
res = 0.2
h_builds = np.array([[5,9,8,7],[7,14,10,11],[5,9,12,8],[3,5,7,9]])   
h_floor = 3
htx = 21
freq = 60*1e9
MCL = 20
x = 94
y = 194
Floor = 1
a = 0.5
Fn_L3 = []

#sb.palplot(sb.diverging_palette(200, 0, n=11))
#plt.savefig('test.png')
#plt.show()
Saved = input("Use Saved Model or Data? Y/N \n")
Model = False
if Saved == 'Y' or Saved == "y" :
    Model = False
elif Saved == "N" or Saved == "n" :
    Model = True
else :
    print("Invalid input! \n") 

if Model:
    N = int(input("The Number of Base-Station:\n ")) 
    print ("Enter the coordinates [x,y] of each base station, such as '50,50', and separate or end with 'Enter':")
    BS_X = []
    BS_Y = []
    BS_set = []
    for i in range(N):
        p,q = input().split(",")
        #print(p,q)
        BS_set.append([p,q])
        while BS_set.count(BS_set[len(BS_set)-1]) > 1 :
            print("The Position of the BS cannot be Repeated!")
            del(BS_set[-1])
            p,q = input().split(",")
            BS_set.append([p,q])
        BS_X.append(int(p))
        BS_Y.append(int(q))
        Fn_L3.append(0)   
    #print(BS_X,BS_Y)
    Vel = float(input("The Velocity of UE: 'x' m/s (E.g. '1' for Human; '5' for Bycicle; '10' for Vehicle) 1~10 \n")) 
    LOSS_Lib = []
    for n_lib in tqdm(range(10)):
        LOSS_sum = []
        for i in range(N):
            LOSS_sum.append(calclossPS1PS2(Floor,bwidth,stwidth,nb_bx,nb_by,res,h_builds,h_floor,htx,freq,MCL,BS_X[i],BS_Y[i]))
        LOSS_Lib.append(LOSS_sum)
        LOSS_sum = np.array(LOSS_sum)
        #print(LOSS_sum.shape,LOSS_sum)
    
    LOSS_Lib = np.array(LOSS_Lib)
    np.save('LOSS_Lib', LOSS_Lib)
else:
    LOSS_Lib = np.load('LOSS_Lib.npy')
    Vel = float(input("The Velocity of UE: 'x' m/s (E.g. '1' for Human; '5' for Bycicle; '10' for Vehicle) 1~10 \n")) 
    N = LOSS_Lib.shape[1]
    #print(N)


Path_NUM = 4
if Path_NUM == 1:
    Start_x = 110
    Start_y = 190
elif Path_NUM == 2:
    Start_x = 208
    Start_y = 14
elif Path_NUM == 3:
    Start_x = 200
    Start_y = 200
elif Path_NUM == 4:
    Start_x = 300
    Start_y = 100
elif Path_NUM == 5:
    Start_x = 370
    Start_y = 300
elif Path_NUM == 6:
    Start_x = 50
    Start_y = 14
Path_RSRP_200_Data = Path_Fitting(Vel,Start_x,Start_y,LOSS_Lib,res,a,Fn_L3,N,Path_NUM)
print(Path_RSRP_200_Data)
Path_RSRP_200_Data = np.array(Path_RSRP_200_Data)
x = []
for i in range(Path_RSRP_200_Data.shape[0]):
    x.append(i+1)
y = Path_RSRP_200_Data.transpose()

df = pd.DataFrame(Path_RSRP_200_Data)
df.to_excel('Path_RSRP_200_Data_'+ str(Vel) + '_' + str(Path_NUM) + '_.xlsx', index=False)

m = int(np.ceil((nb_bx*(bwidth+stwidth)+stwidth)/res))
n = int(np.ceil((nb_by*(bwidth+stwidth)+stwidth)/res))

maps = np.ones((m,n))
maps_index = np.ones((m,n))
 #  Streets in x-direction
j = [(i-1)*res for i in range(1,m+1)]
for i in range(len(j)) :
    j[i] = ((j[i]*10)%((bwidth+stwidth)*10))/10
    if j[i] < stwidth :
        maps[i,:] = 0
        maps_index[i,:] = 0

#  Streets in y-direction
j = [(i-1)*res for i in range(1,n+1)]
for i in range(len(j)) :
    j[i] = ((j[i]*10)%((bwidth+stwidth)*10))/10
    if j[i] < stwidth :
        maps[:,i] = 0  #街道上值为0
        maps_index[:,i] = 0


for i in range(maps.shape[0]):
    for j in range(maps.shape[1]):
        if maps[i,j] == 0:
            if (i-1000)*(i-1000) + (j-1000)*(j-1000) >= 1000000 :
                maps[i,j] = 1

print(maps.shape)

Plot_RSRP(LOSS_Lib,maps,maps_index)

#print(y)
#plt.xticks(x)
#plt.plot(x,y[0],'g-o',x,y[1],'r--s',x,y[2],'m-.8',linewidth=1)
# Test_A, = plt.plot(x,y[0],'g-o',label='BS_1')
# plt.savefig('BS1.png')
# Test_B, = plt.plot(x,y[1],'r--s',label='BS_2')
# plt.savefig('BS2.png')
# Test_C, = plt.plot(x,y[2],'m-.8',label='BS_3')
# plt.savefig('BS3.png')

# plt.legend(handles=[Test_A,Test_B,Test_C],loc='best',bbox_to_anchor=(0,0))
# plt.savefig('BS_BEST.png')


# #plt.tick_params(axis='x',labelsize=15,colors='pink')
# plt.tick_params(axis='y',labelsize=10,colors='black')
# #fig = plt.figure(figsize=(30, 12))
# #设置窗口留白，同时设置图表和坐标轴标题
# plt.tight_layout(pad=5)
# plt.title('Path_RSRP_200ms',fontsize=15)
# plt.xlabel('Time/200ms',fontsize=10)
# plt.ylabel('RSRP_L13',fontsize=10)
# plt.savefig('RSRP.png')
# plt.show()


#print(LOSS_sum)
#此处将用路径生成工具实现，j_p以及k_p的序列；
#j_p = 200
#k_p = 200
#RSRP_sum = []
#First_Data = True
#每40ms计算一次路径点上的RSRP值，作为40ms采集间隔的假设；
#for j in range(10):
#    RSRP = []
#    timestamp = time.time()
#    milliseconds_timestamp_1 = int(timestamp * 1000)
#    for i in range(N):
#        rsrp = LOSS_sum[i,int(k_p/res),int(j_p/res)]
#        RSRP.append(rsrp)
#    timestamp = time.time()
#    milliseconds_timestamp_2 = int(timestamp * 1000)
#    print(milliseconds_timestamp_1,milliseconds_timestamp_2 - milliseconds_timestamp_1)
#    #差一步随机对数正态分布噪声的加入；
#    print(RSRP)
#    RSRP_sum.append(RSRP)
#    #每200ms的L1滤波 // 或者说为每产生5个数据；
#    if j % 5 == 4 :
#        RSRP_sum = np.array(RSRP_sum)
#        RSRP_sum = [[math.pow(10,i/10) for i in row] for row in RSRP_sum]
#        RSRP_sum = np.array(RSRP_sum)
#        RSRP_avg = RSRP_sum.mean(axis=0)  # L1滤波
#        #RSRP_avg = np.sum([math.pow(10,RSRP_sum[i,:]/10) for i in range(len(RSRP_sum))])/5
#        RSRP_sum = []
#        #L3滤波_200ms // L1后的L3滤波 // 其中包含滤波系数 a；
#        #Fn_L3 = [(1-a)*Fn_L3[i] + a*10*(math.log10(abs(RSRP_avg[i]))) for i in range(N)] #表达式有误，想清楚是log前的均值还是后的均值，然后考虑此处表达式怎么写；包括avg
#        if First_Data :
#            Fn_L3 = [10*(math.log10(RSRP_avg[i])) for i in range(N)]
#            First_Data = False
#        else:
#            Fn_L3 = [(1-a)*Fn_L3[i] + a*10*(math.log10(RSRP_avg[i])) for i in range(N)]
#        print(RSRP_avg,Fn_L3)
#    k_p = k_p + 2
#    #以一定的时间间隔上报RAN // 由RAN指定时间间隔；收发包的代码实现；
#timestamp = time.time()
#milliseconds_timestamp = int(timestamp * 1000)
#print(milliseconds_timestamp)
#此代码最大的问题在于时间的并行等处理，不能让程序耽误总进程；
#calclossPS1PS2(j_p,k_p,Floor,bwidth,stwidth,nb_bx,nb_by,res,h_builds,h_floor,htx,freq,MCL,x,y) 