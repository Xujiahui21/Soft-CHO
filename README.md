# Soft-CHO 

A simulation tool that includes User Equipments(UEs) and Radio Access Network(RAN), combined with Free5GC, can achieve the entire process of network resource access, which is parameter-controlled.

<!--- BADGES: START --->

![Github](https://img.shields.io/github/license/Xujiahui21/Soft-CHO)


---

## Overview
The overall system contains UE Simulator, RANs, 5G Core and Data Network. UE Simulator and 5G Core are deployed at localhost (Ubuntu LTS 20.04), while each RAN is encapsulated in Docker. DN is the real external network, e.g. baidu.com.

## UE Simulator

UE simulator is designed based on the dense urban environment, Madrid. It includes path synthesis and channel simulation modules. 
- UE simulator can be quickly started:
```bash
# localhost
cd ./controller
go run .
```

## RAN
Each RAN should be deployed seperately in a Docker, with its core modules including control plane and data plane.
- Control Plane:
  This part is responsible for determining the channel state detection interval, UE reporting interval and several HO/CHO related parameters.
- Data Plane:
  This part is used to build the data access path from UE to Data Network and simulate channel packet loss based on signal strength.
```bash
# Docker
cd ./ran_cho
go run .
```

## 5G Core Network
The simulation of 5G Core Network is based on Free5GC, an open-source project on GitHub. The version of Free5GC in our testbed is v3.2.1. It should be deployed at localhost.

## Benchmark
CHO performance is composed of resource utilization and network performance. 
> Resource utilization includes communication resources and storage resources.
> - Storage resources occupation can be calculated based on the log files.
> - Communication resources occupation can be evaluated using PingPong times.

> Network performance can be assessed by Handover interruption time, PingPong times and average packet loss rate.
> - Handover interruption time:
> ```bash
> python ./controller/DataPlane_InterruptTime_PATHa.py
> ```
> - PingPong times can be calculated based on the log files.
> - Average packet loss rate:
> ```bash
> python ./controller/Packet_Loss_Rate_analyse.py
> ```
