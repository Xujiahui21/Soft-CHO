package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/tealeg/xlsx"
)

var (
	Logname      = "/root/controller_cho_new/log/" + time.Now().Format("2006-01-02_15-04-05")[5:]
	open_uevm    bool
	uevm_ip      = "10.156.168.52:9000"
	uevm_conn    net.Conn
	ue_nic_limit string
	// Interval  = 3
	Interval               = 1
	ue_list                []UE
	conn_list              []net.Conn
	background_ping        bool
	auto_run               bool
	scenario               string
	handover_type          string
	ip_allocated           = 0
	ue_number              int
	ran_number             int
	ch_command             = make(chan string, 128)
	ch_control             = make(chan string, 128)
	ch_scenario            = make(chan string, 128)
	ch_log                 = make(chan string, 4096)
	ch_quit                = make(chan struct{}, 1)
	next_list              = make(map[string][]int)
	RSRP                   [][]float64
	RSRP_ALL_NOW           []float64
	RSRP_INTERVAL          = 100
	RSRP_Meas_Period       = 4
	Trigger_Type           string
	Trigger_Threshold              = 20
	TTT_Pre                float64 = 640
	TTT_Exc                float64 = 640
	RSRP_state                     = 0
	Packet_Loss_Rate       float64 = 0.0
	PLR_switch             int     = 1 // 0 close 1 open
	HO_Decision            bool    = false
	HO_threshold           float64 = 5
	UL_DL                  string
	RSRP_NOW               float64 = 0
	RSRP_156               bool    = false
	FIRST_RAN              int     = 0
	CHO_Prepare            bool    = true
	CHO_Excution           bool    = true
	package_count          int     = 0
	Hys                    int     = -100
	Trigger_Threshold_copy int     = 20
)

func init() {
	println("Getting configuraton...")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/root/controller_cho_new/")
	viper.ReadInConfig()

	scenario = viper.GetString("run.scenario")
	handover_type = viper.GetString("run.type")
	background_ping = viper.GetBool("run.background_ping")
	open_uevm = viper.GetBool("run.open_uevm")
	ue_nic_limit = viper.GetString("run.ue_nic_limit")
	auto_run = viper.GetBool("run.auto_run")
	// println(background_ping)

	Logname += "-" + scenario + "-" + handover_type + ".log"
	if !exists("/root/controller_cho_new/log/") {
		os.Mkdir("/root/controller_cho_new/log/", os.ModePerm)
	}
	if !exists(Logname) {
		os.Create(Logname)
	}
	go write_log(ch_log, ch_quit)
}

func main() {

	Xlsx_to_Data()
	FIRST_RAN, _ = MAX_index(RSRP[0])
	println("runing", scenario, handover_type, "...")
	if !init_scenario() {
		println("Failed to init scenario. Abort.")
		return
	}
	var command string

	go command_ran()
	go Send_RSRP()
	go Trigger_MeasurementReports()

	// fmt.Println("Please input command, 's' for start, 'q' for quit.")
	go func(ch_control chan string) {
		var command string
		for {
			fmt.Scanln(&command)
			// input := bufio.NewReader(os.Stdin)
			// s, _ := input.ReadString('\n')
			ch_control <- command
		}
	}(ch_control)

	// 自动开始，一改之前的指令输入s 代表start；
	ch_control <- "s"
	for {
		command = <-ch_control
		ch_command <- command
		if command == "q" {
			time.Sleep(2 * time.Second)
			release()
			break
		}
		// fmt.Println("Please input command, 'q' for quit.")
	}
}

func Xlsx_to_Data() {
	//Open Excel File
	xlFile, err := xlsx.OpenFile("Path_RSRP_200_Data_1_1.xlsx")
	if err != nil {
		fmt.Printf("Open error:%v\n", err)
		return
	}
	//defer xlFile.Close()

	//Go Over Sheet, Row, Col
	for _, sheet := range xlFile.Sheets {
		fmt.Printf("%s\n", sheet.Name)
		First_line := true
		var count_vert int = 0

		for _, row := range sheet.Rows {
			var rsrp []float64

			if First_line {
				First_line = false
				continue
			}
			count_vert = (count_vert + 1)
			// if count_vert < 9200 || count_vert > 10200 {
			// 	continue
			// }

			for _, cell := range row.Cells {
				text := cell.String()
				//fmt.Printf("%s\n", text)
				num, err := strconv.ParseFloat(text, 64)
				//fmt.Printf("%f\n", num)
				//整化 or 定上下界 ： 上下界
				if num > -31 {
					num = -31
				} else if num < -156 {
					num = -156
				}
				rsrp = append(rsrp, num)
				if err != nil {
					fmt.Printf("Convert error:%v\n", err)
					return
				}
			}
			// count_vert = (count_vert + 1) % 5 //%x xm/s
			// if count_vert == 1 {
			// 	RSRP = append(RSRP, rsrp)
			// }
			RSRP = append(RSRP, rsrp)
		}
	}
	// fmt.Printf("%v\n", RSRP)
}

// listen conn程序只做基本的收包，尽量不阻塞。然后通过chan发送给对应函数推进下一步
func listen_ran(conn net.Conn) {

	var UE_unique_PLR float64
	// ran第一次连接到controller，通知：总ran个数、HO类型、HO ue数量。默认注册1个ue用以发送背景流量
	conn.Write(add_tcp_length(viper.GetString("scenario."+scenario+".ran") + "," + handover_type + "," + viper.GetString("scenario."+scenario+".ue")))
	reader := bufio.NewReader(conn)
	// var buf [1024]byte

	for {
		raw_length, err := reader.Peek(2)
		// n, err := reader.Read(buf[:])
		if err != nil {
			if err == io.EOF {
				println("Ran close the connection.")
				conn.Close()
				return
			}
			select {
			case <-ch_quit:
				conn.Close()
				return
			default:
				fmt.Printf("Read from conn failed, err:%v\n", err)
			}
		}
		length := uint16(raw_length[0])<<8 + uint16(raw_length[1])
		buf := make([]byte, length+2)
		reader.Read(buf)

		recv_split := strings.Split(string(buf[2:]), ",")
		ueid_int, _ := strconv.Atoi(recv_split[0])
		ran_int := strings.Split(recv_split[2], " ")[0]

		if ueid_int == 0 {
			print(recv_split)
			continue
		}

		if len(RSRP_ALL_NOW) == ran_number && ueid_int != 0 {
			if strings.HasPrefix(ran_int, "ran") {
				ran_int_, _ := strconv.Atoi(strings.Split(ran_int, "ran")[1])
				UE_unique_PLR = RSRP_TO_PACKETLOSSRATE(RSRP_ALL_NOW[ran_int_], PLR_switch)
			} else {
				if ueid_int == 4 {
					println(ueid_int, ue_list[ueid_int].ran_now, ue_list[ueid_int].state)
				}
				UE_unique_PLR = RSRP_TO_PACKETLOSSRATE(RSRP_ALL_NOW[ue_list[ueid_int].ran_now], PLR_switch)
			}
		} else {
			UE_unique_PLR = 0
		}

		if PLR_switch == 0 { //收包丢包处理
			go handle_recv(string(buf[2:]))
		} else if rand.Float64() >= UE_unique_PLR && PLR_switch == 1 {
			go handle_recv(string(buf[2:]))
		} else {
			fmt.Printf("Packet Loss:%f\n", UE_unique_PLR)
			continue
		}
		// go handle_recv(string(buf[2:]))
		// go handle_recv(string(buf[:n]))
		// if RSRP_state == 1 {
		// 	go Send_RSRP(conn)
		// 	RSRP_state = 0
		//}
	}
}

func handle_recv(recv string) { //接收到RAN的消息后的动作以及一些状态更新；
	// fmt.Println("recv:", recv)
	recv_split := strings.Split(recv, ",")
	ueid_int, _ := strconv.Atoi(recv_split[0])
	full_command := recv_split[1]
	command := strings.Split(full_command, "||")[0]

	if command == "RSRP_Interval" {
		RSRP_INTERVAL, _ = strconv.Atoi(recv_split[3])
	}

	if command == "RSRP_Meas_Period" {
		RSRP_Meas_Period, _ = strconv.Atoi(recv_split[3])
	}

	if command == "MR_Trigger_Event_Para" {
		Trigger_Type = recv_split[4]
		Trigger_Threshold, _ = strconv.Atoi(recv_split[6])
		// Trigger_Threshold_copy, _ = strconv.Atoi(recv_split[6])
		// if package_count < 1000 {
		// 	Trigger_Threshold = -140
		// } else {
		// 	Trigger_Threshold = Trigger_Threshold_copy
		// }
		TTT_int, _ := strconv.Atoi(recv_split[8])
		TTT_Pre = float64(TTT_int)
	}

	// 处理ran发送的Authentication Request并广播其他ran，不做日志记录
	if command == "Authentication Request" {
		ch_command <- recv
		// return
	} else if command == "Allocate IP & TEID" {
		// 收到ip和teid广播给所有ran。同时也给到uevm
		ch_command <- recv
		ip_allocated++
		// go start_nic(ueid_int, recv_split[2], recv_split[3])
		// return
	}

	t := time.Now().Format("2006-01-02 15:04:05.000000")
	content := recv + ",controller recv from ran," + strings.Split(t, " ")[1]

	// from_ran := recv_split[len(recv_split)-2]
	// if ue_list[ueid_int].state == duringXn && from_ran == "ran"+strconv.Itoa(ue_list[ueid_int].ran_to)+" send to ue" {
	// 	ue_list[ueid_int].state = connected
	// 	ue_list[ueid_int].ran_now = ue_list[ueid_int].ran_to
	// 	ue_list[ueid_int].ran_to = next_ran(ue_list[ueid_int].ran_now)
	// }

	switch command {
	case "Registration Complete":
		ue_list[ueid_int].state = connected
		HO_Decision = false
		CHO_Prepare = true
		CHO_Excution = true
		RSRP_156 = false
	case "RSRP_Interval":
		RSRP_state = 1
	case "Xn Handover Complete", "N2 Handover Complete": //ran 状态更改的位置 //没收到
		ue_list[ueid_int].state = connected
		ue_list[ueid_int].ran_now = ue_list[ueid_int].ran_to
		//ue_list[ueid_int].ran_to = next_ran(ue_list[ueid_int].ran_now)
		if open_uevm {
			uevm_conn.Write(add_tcp_length(strconv.Itoa(ran_number+1) + ",HO to new ran," + strconv.Itoa(ue_list[ueid_int].ran_to)))
		}
		HO_Decision = false
		CHO_Prepare = true
		CHO_Excution = true
	case "RRC Reconfiguration (HO Command)":
		ch_command <- recv_split[0] + "," + full_command
	case "Handover Command":
		s_split := strings.Split(recv, "||")
		ULDL := s_split[1]
		NH := s_split[2]
		ch_command <- recv_split[0] + ",Handover Confirm||" + ULDL + "||" + NH + "||"
	case "App Receive":
		fmt.Printf("recv: %v\n", []byte(recv[len(recv_split[0])+13:]))
		// write_nic(ue_list[ueid_int].nic_fd, []byte(recv[len(recv_split[0])+13:]))
	case "CHO Configuration Delete ACK":
		HO_Decision = false
		CHO_Prepare = true
		CHO_Excution = true
		ue_list[ueid_int].state = connected
	case "RRC Reestablished":
		HO_Decision = false //表示当前没有在进行HO
		CHO_Prepare = true  //表示准备好进行preparation
		CHO_Excution = true //表示准备好进行excution
		ue_list[ueid_int].state = connected
	default:
		// if found, _ := regexp.MatchString("Handover Command", command); found {
		// 	s_split := strings.Split(recv, "||")
		// 	ULDL := s_split[1]
		// 	NH := s_split[2]
		// 	ch_command <- recv_split[0] + ",Handover Confirm||" + ULDL + "||" + NH + "||"
		// 	break
		// }
	}

	ch_log <- content + "\n"
}

// listen conn程序只做基本的收包，尽量不阻塞。然后通过chan发送给对应函数推进下一步
func command_ran() {
	var full_command string
	var command string
	//var plr_now float64
	// 等待注册ue等环节的初始化
	// time.Sleep(2 * time.Second)

	for {
		time.Sleep(time.Duration(Interval) * time.Millisecond)

		select {
		case full_command = <-ch_command:
			// fmt.Println("send command to ran:", command)
		default:
			// 都分配到ip后再开始发包
			if !background_ping || ip_allocated < ue_number+ran_number {
				continue
			}
			// fmt.Println("send package")
			// 需要对list内部元素进行更改时，range方式直接获取元素有风险(不是引用而是复制创建了新元素)
			for i := 1; i < len(ue_list); i++ { //第一个空ue，直接放掉；
				if ue_list[i].state != initial && ue_list[i].allow_send {
					// 先加编号，确保发送index与预期一致
					ue_list[i].pkg_index = (ue_list[i].pkg_index + 1) % 65536
					command = strconv.Itoa(i) + "," + strconv.Itoa(ue_list[i].pkg_index)
					// if len(RSRP_ALL_NOW) == ran_number {
					// 	plr_now = RSRP_TO_PACKETLOSSRATE(RSRP_ALL_NOW[ue_list[i].ran_now], PLR_switch)
					// } else {
					// 	plr_now = 0
					// }

					go send_message(conn_list[ue_list[i].ran_now], command)
					if ue_list[i].state == duringXn {
						go send_message(conn_list[ue_list[i].ran_to], command)
					}
					time.Sleep(3 * time.Millisecond)
				}
			}
			continue
		}

		full_command_split := strings.Split(full_command, ",")

		//if HO_Decision {
		//	new_command := full_command_split[0] + ",RACH Target_RAN"
		//	ueid_int, _ := strconv.Atoi(full_command_split[0])
		//	go send_message(conn_list[ue_list[ueid_int].ran_to], new_command)
		//}

		if len(full_command_split) == 1 {
			switch full_command {
			case "q":
				command = "-1" + ",Quit"
				for _, conn := range conn_list {
					send_message(conn, command)
				}
				if open_uevm {
					send_message(uevm_conn, command)
				}
				return

			case "s":
				println("start scenario:", scenario)
				switch scenario {
				case "single":
					go single()
				case "loop":
					go loop()
				case "batch":
					go batch()
				case "multi_connect":
					go multi_connect()
				case "test_run":
					go test_run()
				}

			case "h":
				ch_scenario <- ""
			default:
				println("unknown command:", full_command)
			}
		} else {
			command = strings.Split(full_command_split[1], "||")[0]
			switch command {
			// if found, _ := regexp.MatchString("RRC Reconfiguration (HO Command)", command_split[1]); found
			case "RRC Reconfiguration (HO Command)":
				ULDL := strings.Split(full_command_split[1], "||")[2]
				UL_DL = ULDL
				HO_threshold, _ = strconv.ParseFloat(strings.Split(full_command_split[1], "||")[4], 64)
				TTT_Exc, _ = strconv.ParseFloat(strings.Split(full_command_split[1], "||")[5], 64)
				new_command := full_command_split[0] + ",RRC Reconfiguration Complete||" + ULDL
				ueid_int, _ := strconv.Atoi(full_command_split[0])
				// println("ueid:", full_command_split[0], "ran to:", ue_list[ueid_int].ran_to)
				ue_list[ueid_int].ran_to, _ = strconv.Atoi(strings.Split(full_command_split[1], "||")[1])
				//println("ueid:", full_command_split[0], "ran to:", ue_list[ueid_int].ran_to)
				go send_message(conn_list[ue_list[ueid_int].ran_now], new_command)
				//println("ueid:", full_command_split[0], "ran now:", ue_list[ueid_int].ran_now, new_command)
				//conn.Write(add_tcp_length(full_command))
				HO_Decision = true
				// if package_count < 9100 {
				go CHO_Timer(full_command_split[0])
				// }
				// if open_uevm {
				// 	uevm_conn.Write(add_tcp_length(full_command_split[0] + ",HO to new ran," + strconv.Itoa(ue_list[ueid_int].ran_to)))
				// }
				// time.Sleep(10 * time.Millisecond)
				// Interval = 10
			case "Authentication Request":
				for i, conn := range conn_list {
					if i == 0 {
						continue
					}
					conn.Write(add_tcp_length(full_command))
				}
			case "Handover Confirm":
				ueid_int, _ := strconv.Atoi(full_command_split[0])
				send_message(conn_list[ue_list[ueid_int].ran_to], full_command)
				if open_uevm {
					uevm_conn.Write(add_tcp_length(full_command_split[0] + ",HO to new ran," + strconv.Itoa(ue_list[ueid_int].ran_to)))
				}
			case "Allocate IP & TEID":
				for _, conn := range conn_list {
					conn.Write(add_tcp_length(full_command))
				}
				if open_uevm {
					uevm_conn.Write(add_tcp_length(full_command))
				}

			default:
				println("unknown command:", full_command)
			}
		}
	}
}

func Send_RSRP() {

	for {
		season := RSRP_INTERVAL / RSRP_Meas_Period / 5
		if RSRP_state == 1 {
			for i := 0; i < len(RSRP)/season; i++ {
				fmt.Printf("%d\n", RSRP_INTERVAL)
				BS1_RSRP := ""
				BS2_RSRP := ""
				BS3_RSRP := ""
				time.Sleep(time.Duration(RSRP_INTERVAL) * time.Millisecond) //RSRP_INTERVAL
				for j := 0; j < season; j++ {
					//BS1_RSRP = BS1_RSRP + "," + strconv.FormatFloat(math.Floor(RSRP[i*season+j][0]), 'f', 10, 64)
					BS1_RSRP = BS1_RSRP + "," + strconv.FormatFloat(RSRP[i*season+j][0], 'f', 10, 64)
					BS2_RSRP = BS2_RSRP + "," + strconv.FormatFloat(RSRP[i*season+j][1], 'f', 10, 64)
					BS3_RSRP = BS3_RSRP + "," + strconv.FormatFloat(RSRP[i*season+j][2], 'f', 10, 64)
				}
				// RSRP_ALL_NOW = RSRP[i*season+season-1]
				// RSRP_NOW = RSRP[i*season+season-1][ue_list[ran_number+1].ran_now]
				// for l := 0; l < 3; l++ {
				// println(RSRP_ALL_NOW)
				// println(HO_Decision, ue_list[ran_number+1].ran_to)
				//RSRP_ran_to, RSRP_MAX := MAX_index(RSRP_ALL_NOW)

				// if PLR_switch == 1 { //open则向持续发包，告知当前信道情况；1表示通过，0表示扔掉
				// 	for r := 0; r < ran_number; r++ {
				// 		if rand.Float64() >= RSRP_TO_PACKETLOSSRATE(RSRP_ALL_NOW[r], PLR_switch) {
				// 			rsrp_command := "PLR:1"
				// 			go send_message(conn_list[r], rsrp_command) //1表示不丢包
				// 		} else {
				// 			rsrp_command := "PLR:0"
				// 			go send_message(conn_list[r], rsrp_command) //0表示丢包
				// 		}
				// 	}
				// }
				command := "RSRP:" + "BS1" + BS1_RSRP + ",BS2" + BS2_RSRP + ",BS3" + BS3_RSRP
				go send_message(conn_list[ue_list[ran_number+1].ran_now], command)
				fmt.Printf("Send RSRP: %s\n", command)

				// if RSRP[i*season][ue_list[ran_number+1].ran_to] > RSRP_NOW+HO_threshold && HO_Decision { //只能进来一次  i*season+season-1
				// 	println(ue_list[ran_number+1].ran_to, ue_list[ran_number+1].ran_now)
				// 	if open_uevm {
				// 		uevm_conn.Write(add_tcp_length(strconv.Itoa(ran_number+1) + ",HO to new ran," + strconv.Itoa(ue_list[ran_number+1].ran_to)))
				// 	}
				// 	// ue_list[4].ran_to = l
				// 	new_command := strconv.Itoa(ran_number+1) + ",RACH Target_RAN||" + UL_DL
				// 	fmt.Printf("%s\n", new_command)
				// 	go send_message(conn_list[ue_list[ran_number+1].ran_to], new_command)
				// 	RSRP_NOW = RSRP[i*season+season-1][ue_list[ran_number+1].ran_to]
				// 	//RSRP_NOW = RSRP[i*season][ue_list[ran_number+1].ran_to]
				// 	// break
				// 	go TIMER_10s(strconv.Itoa(ran_number + 1))

				// }
				// if RSRP_MAX > RSRP_NOW+HO_threshold && HO_Decision { //只能进来一次
				// 	ue_list[ran_number+1].ran_to = RSRP_ran_to
				// 	println(ue_list[ran_number+1].ran_to, ue_list[ran_number+1].ran_now)
				// 	if open_uevm {
				// 		uevm_conn.Write(add_tcp_length(strconv.Itoa(ran_number+1) + ",HO to new ran," + strconv.Itoa(ue_list[ran_number+1].ran_to)))
				// 	}
				// 	// ue_list[4].ran_to = l
				// 	new_command := strconv.Itoa(ran_number+1) + ",RACH Target_RAN||" + UL_DL
				// 	fmt.Printf("%s\n", new_command)
				// 	go send_message(conn_list[ue_list[ran_number+1].ran_to], new_command)
				// 	RSRP_NOW = RSRP[i*season+season-1][ue_list[ran_number+1].ran_to]
				// 	//RSRP_NOW = RSRP[i*season][ue_list[ran_number+1].ran_to]
				// 	// break
				// 	go TIMER_10s(strconv.Itoa(ran_number + 1))

				// }
				// }
				// if RSRP_NOW > -90 {
				// 	Packet_Loss_Rate = 0
				// } else if RSRP_NOW > -110 {
				// 	Packet_Loss_Rate = 0.000001 / 20 * (-90 - RSRP_NOW)
				// } else {
				// 	Packet_Loss_Rate = 0.000001 + ((math.Exp(-110-RSRP_NOW)-1)*0.999999)/(math.Exp(46)-1)
				// }

				// if RSRP_NOW == -156 && !RSRP_156 {
				// 	RSRP_156 = true
				// 	println("RSRP_156 Timer on!")
				// 	go TIMER_RSRP(strconv.Itoa(ran_number + 1))
				// }

			}
			RSRP_state = 0
			ch_control <- "q"
		}
	}
}

func Trigger_MeasurementReports() {
	var plr_vm_flag int
	for {
		if RSRP_state == 1 {
			for i := 0; i < len(RSRP); i++ {
				if i == 0 {
					time.Sleep(time.Duration(5*RSRP_Meas_Period) * time.Millisecond)
				} else {
					time.Sleep(time.Duration(5*RSRP_Meas_Period) * time.Millisecond) //RSRP_INTERVAL -20
				}
				package_count = package_count + 1
				RSRP_ALL_NOW = RSRP[i]
				RSRP_NOW = RSRP[i][ue_list[ran_number+1].ran_now]
				// if package_count < 1000 {
				// 	Trigger_Threshold = -140
				// } else {
				// 	Trigger_Threshold = Trigger_Threshold_copy
				// }
				// for l := 0; l < 3; l++ {
				// println(RSRP_ALL_NOW)
				println(CHO_Prepare, CHO_Excution, HO_Decision, ue_list[ran_number+1].ran_to, Trigger_Threshold, HO_threshold, TTT_Pre, TTT_Exc)
				//RSRP_ran_to, RSRP_MAX := MAX_index(RSRP_ALL_NOW)

				if PLR_switch == 1 { //open则向持续发包，告知当前信道情况；1表示通过，0表示扔掉
					for r := 0; r < ran_number; r++ {
						if rand.Float64() >= RSRP_TO_PACKETLOSSRATE(RSRP_ALL_NOW[r], PLR_switch) {
							rsrp_command := "PLR:1"
							plr_vm_flag = 1
							go send_message(conn_list[r], rsrp_command) //1表示不丢包
						} else {
							rsrp_command := "PLR:0"
							plr_vm_flag = 0
							go send_message(conn_list[r], rsrp_command) //0表示丢包
						}
						if r == ue_list[ran_number+1].ran_now && open_uevm {
							uevm_conn.Write(add_tcp_length(strconv.Itoa(ran_number+1) + ",PLR," + strconv.Itoa(plr_vm_flag) + "," + strconv.Itoa(ue_list[ran_number+1].ran_now)))
						}
					}
				}
				// command := "RSRP:" + "BS1" + BS1_RSRP + ",BS2" + BS2_RSRP + ",BS3" + BS3_RSRP
				// go send_message(conn_list[ue_list[ran_number+1].ran_now], command)
				// fmt.Printf("Send RSRP: %s\n", command)

				// CHO Preparation Phase MR Trigger; Set CHO Preparation flag (up);
				ran_index := 1
				for ran_index < ran_number {
					// if (ue_list[ran_number+1].ran_now+ran_index)%ran_number == 0 {
					// 	TTT_Pre = 640
					// 	Hys = 0
					// }
					// if (ue_list[ran_number+1].ran_now+ran_index)%ran_number == 2 {
					// 	TTT_Pre = 1024
					// }
					if RSRP[i][(ue_list[ran_number+1].ran_now+ran_index)%ran_number] >= float64(Trigger_Threshold) && CHO_Prepare { //只能进来一次  i*season+season-1   + float64(Hys)
						// if RSRP[i][(ue_list[ran_number+1].ran_now+ran_index)%ran_number] >= float64(Trigger_Threshold)+RSRP[i][ue_list[ran_number+1].ran_now] && CHO_Prepare { //&& RSRP[i][(ue_list[ran_number+1].ran_now+ran_index)%ran_number] >= float64(Hys)
						println((ue_list[ran_number+1].ran_now+ran_index)%ran_number, ue_list[ran_number+1].ran_now, RSRP[i][(ue_list[ran_number+1].ran_now+ran_index)%ran_number], CHO_Prepare, CHO_Excution)
						BS1_RSRP := strconv.FormatFloat(RSRP[i+int(math.Floor(TTT_Pre/float64(RSRP_Meas_Period)/5))][0], 'f', 10, 64)
						BS2_RSRP := strconv.FormatFloat(RSRP[i+int(math.Floor(TTT_Pre/float64(RSRP_Meas_Period)/5))][1], 'f', 10, 64)
						BS3_RSRP := strconv.FormatFloat(RSRP[i+int(math.Floor(TTT_Pre/float64(RSRP_Meas_Period)/5))][2], 'f', 10, 64)
						command := "||," + "BS1," + BS1_RSRP + ",BS2," + BS2_RSRP + ",BS3," + BS3_RSRP
						// go send_message(conn_list[ue_list[ran_number+1].ran_now], command)
						fmt.Printf("Send MR RSRP: %s\n", command)
						var RSRP_Detect []float64
						for j := 0; j < int(math.Floor(TTT_Pre/float64(RSRP_Meas_Period)/5))+1; j++ {
							print(RSRP[i+j][(ue_list[ran_number+1].ran_now+ran_index)%ran_number])
							RSRP_Detect = append(RSRP_Detect, RSRP[i+j][(ue_list[ran_number+1].ran_now+ran_index)%ran_number])
						}
						var RSRP_Base []float64
						for j := 0; j < int(math.Floor(TTT_Pre/float64(RSRP_Meas_Period)/5))+1; j++ {
							print(RSRP[i+j][ue_list[ran_number+1].ran_now])
							RSRP_Base = append(RSRP_Base, RSRP[i+j][ue_list[ran_number+1].ran_now])
						}
						fmt.Println(RSRP_Detect, RSRP_Base)

						go TIMER_TTT_Preparation(RSRP_Detect, RSRP_Base, command)
						ue_list[ran_number+1].ran_to = (ue_list[ran_number+1].ran_now + ran_index) % ran_number
						break
						// if open_uevm {
						// 	uevm_conn.Write(add_tcp_length(strconv.Itoa(ran_number+1) + ",HO to new ran," + strconv.Itoa(ue_list[ran_number+1].ran_to)))
						// }
						// // ue_list[4].ran_to = l
						// new_command := strconv.Itoa(ran_number+1) + ",RACH Target_RAN||" + UL_DL
						// fmt.Printf("%s\n", new_command)
						// go send_message(conn_list[ue_list[ran_number+1].ran_to], new_command)
						// RSRP_NOW = RSRP[i][ue_list[ran_number+1].ran_to]
						// //RSRP_NOW = RSRP[i*season][ue_list[ran_number+1].ran_to]
						// // break
						// go TIMER_10s(strconv.Itoa(ran_number + 1))

					}
					ran_index = ran_index + 1
				}

				// If CHO preparation phase flag True; Detect candidate RAN Trigger Event; Candidate RAN 1
				if !CHO_Prepare && RSRP[i][ue_list[ran_number+1].ran_to] > RSRP_NOW+HO_threshold && HO_Decision && CHO_Excution { ///改为TTT计时下的结果  && RSRP[i][ue_list[ran_number+1].ran_to] >= float64(Hys)
					println(ue_list[ran_number+1].ran_to, ue_list[ran_number+1].ran_now)
					var RSRP_Detect []float64
					var RSRP_Base []float64
					for j := 0; j < int(math.Floor(TTT_Exc/float64(RSRP_Meas_Period)/5))+1; j++ {
						print(RSRP[i+j][ue_list[ran_number+1].ran_to], RSRP[i+j][ue_list[ran_number+1].ran_now])
						RSRP_Detect = append(RSRP_Detect, RSRP[i+j][ue_list[ran_number+1].ran_to])
						RSRP_Base = append(RSRP_Base, RSRP[i+j][ue_list[ran_number+1].ran_now])
					}
					fmt.Println(RSRP_Detect, RSRP_Base, ue_list[ran_number+1].ran_to)
					go TIMER_TTT_Excution(RSRP_Detect, RSRP_Base, ue_list[ran_number+1].ran_to)
				}

				// If CHO preparation phase flag True; Detect candidate RAN Trigger Event; Candidate RAN all
				// ran_index_exc := 1
				// for ran_index_exc < ran_number {
				// 	if !CHO_Prepare && RSRP[i][(ue_list[ran_number+1].ran_now+ran_index_exc)%ran_number] > RSRP_NOW+HO_threshold && HO_Decision && CHO_Excution { ///改为TTT计时下的结果
				// 		println((ue_list[ran_number+1].ran_now+ran_index_exc)%ran_number, ue_list[ran_number+1].ran_now)
				// 		var RSRP_Detect []float64
				// 		var RSRP_Base []float64
				// 		for j := 0; j < int(math.Floor(TTT_Exc/200))+1; j++ {
				// 			print(RSRP[i+j][(ue_list[ran_number+1].ran_now+ran_index_exc)%ran_number], RSRP[i+j][ue_list[ran_number+1].ran_now])
				// 			RSRP_Detect = append(RSRP_Detect, RSRP[i+j][(ue_list[ran_number+1].ran_now+ran_index_exc)%ran_number])
				// 			RSRP_Base = append(RSRP_Base, RSRP[i+j][ue_list[ran_number+1].ran_now])
				// 		}
				// 		fmt.Println(RSRP_Detect, RSRP_Base)
				// 		go TIMER_TTT_Excution(RSRP_Detect, RSRP_Base, (ue_list[ran_number+1].ran_now+ran_index_exc)%ran_number)
				// 		ue_list[ran_number+1].ran_to = (ue_list[ran_number+1].ran_now + ran_index_exc) % ran_number
				// 		break
				// 	}
				// 	ran_index_exc = ran_index_exc + 1
				// }

				// if RSRP_MAX > RSRP_NOW+HO_threshold && HO_Decision { //只能进来一次
				// 	ue_list[ran_number+1].ran_to = RSRP_ran_to
				// 	println(ue_list[ran_number+1].ran_to, ue_list[ran_number+1].ran_now)
				// 	if open_uevm {
				// 		uevm_conn.Write(add_tcp_length(strconv.Itoa(ran_number+1) + ",HO to new ran," + strconv.Itoa(ue_list[ran_number+1].ran_to)))
				// 	}
				// 	// ue_list[4].ran_to = l
				// 	new_command := strconv.Itoa(ran_number+1) + ",RACH Target_RAN||" + UL_DL
				// 	fmt.Printf("%s\n", new_command)
				// 	go send_message(conn_list[ue_list[ran_number+1].ran_to], new_command)
				// 	RSRP_NOW = RSRP[i*season+season-1][ue_list[ran_number+1].ran_to]
				// 	//RSRP_NOW = RSRP[i*season][ue_list[ran_number+1].ran_to]
				// 	// break
				// 	go TIMER_10s(strconv.Itoa(ran_number + 1))

				// }
				// }
				if RSRP_NOW > -90 {
					Packet_Loss_Rate = 0
				} else if RSRP_NOW > -110 {
					Packet_Loss_Rate = 0.000001 / 20 * (-90 - RSRP_NOW)
				} else {
					Packet_Loss_Rate = 0.000001 + ((math.Exp(-110-RSRP_NOW)-1)*0.999999)/(math.Exp(46)-1)
				}

				if RSRP_NOW == -156 && !RSRP_156 {
					RSRP_156 = true
					println("RSRP_156 Timer on!")
					go TIMER_RSRP(strconv.Itoa(ran_number + 1))
				}

			}
			RSRP_state = 0
			ch_control <- "q"
		}
	}
}

func RSRP_TO_PACKETLOSSRATE(RSRP float64, PLR_switch int) float64 {

	var Packet_Loss_Rate_in float64
	if PLR_switch == 0 {
		Packet_Loss_Rate_in = 0.0
	} else {
		if RSRP > -90 {
			Packet_Loss_Rate_in = 0
		} else if RSRP > -110 {
			Packet_Loss_Rate_in = 0.000001 / 20 * (-90 - RSRP)
		} else {
			Packet_Loss_Rate_in = 0.000001 + ((math.Exp(-110-RSRP)-1)*0.999999)/(math.Exp(46)-1)
		}
	}

	return Packet_Loss_Rate_in

}

func TIMER_RSRP(ueid string) {
	t1 := time.Now()
	t2 := time.Now()
	T_target := t1.Add(time.Second * 1) // T310, 1000ms

	for t2.Before(T_target) {
		t2 = time.Now()
		if RSRP_NOW != -156 {
			RSRP_156 = false
			return
		}
	}
	ueid_int, _ := strconv.Atoi(ueid)
	command := ueid + ",RRC Reestablish Request"
	go send_message(conn_list[ue_list[ueid_int].ran_now], command)

	//等待重建立ACK，定时；

	t1 = time.Now()
	t2 = time.Now()
	T_target = t1.Add(time.Second * 2) // T301, 2000ms

	for t2.Before(T_target) {
		t2 = time.Now()
		if RSRP_NOW != -156 {
			RSRP_156 = false
			return
		}
	}

	//等待RRC ACK失败，则向新RAN发起注册；
	//等待complete失败，则发起RRC重建立；

	command = ueid + ",UE Reregistration"
	ue_list[ueid_int].ran_now, _ = MAX_index(RSRP_ALL_NOW)
	//ue_list[ueid_int].ran_to = next_ran(ue_list[ueid_int].ran_now)
	go send_message(conn_list[ue_list[ueid_int].ran_now], command)
}

func TIMER_10s(ueid string) {
	t1 := time.Now()
	t2 := time.Now()
	T_target := t1.Add(time.Millisecond * 500) // T304, 500ms定时

	for t2.Before(T_target) {
		t2 = time.Now()
		if !HO_Decision {
			return
		}
	}
	ueid_int, _ := strconv.Atoi(ueid)
	command := ueid + ",RRC Reestablish Request"
	go send_message(conn_list[ue_list[ueid_int].ran_now], command)

	//等待重建立ACK，定时；

	t1 = time.Now()
	t2 = time.Now()
	T_target = t1.Add(time.Second * 3)

	for t2.Before(T_target) {
		t2 = time.Now()
		if !HO_Decision {
			return
		}
	}

	//等待RRC ACK失败，则向新RAN发起注册；
	//等待complete失败，则发起RRC重建立；

	command = ueid + ",UE Reregistration"
	ue_list[ueid_int].ran_now, _ = MAX_index(RSRP_ALL_NOW)
	//ue_list[ueid_int].ran_to = next_ran(ue_list[ueid_int].ran_now)
	go send_message(conn_list[ue_list[ueid_int].ran_now], command)
}

func TIMER_TTT_Preparation(RSRP_detect []float64, RSRP_base []float64, MR_command string) {
	t1 := time.Now()
	t2 := time.Now()
	fmt.Println(TTT_Pre, RSRP_detect)
	T_target := t1.Add(time.Millisecond * time.Duration(math.Floor(TTT_Pre/float64(RSRP_Meas_Period)/5)*float64(RSRP_Meas_Period)*5)) // TTT, 640ms定时

	index := 1
	for t2.Before(T_target) {

		time.Sleep(time.Duration(float64(RSRP_Meas_Period)*5) * time.Millisecond)
		print(RSRP_detect[index], index, Trigger_Threshold, CHO_Prepare)
		if RSRP_detect[index] < float64(Trigger_Threshold) || !CHO_Prepare { // - float64(Hys)
			// if RSRP_detect[index] < float64(Trigger_Threshold)+RSRP_base[index] || !CHO_Prepare { // || RSRP_detect[index] < float64(Hys)
			return
		}
		index = index + 1
		t2 = time.Now()
	}

	time.Sleep(time.Duration(TTT_Pre-math.Floor(TTT_Pre/float64(RSRP_Meas_Period)/5)*float64(RSRP_Meas_Period)*5) * time.Millisecond)
	// ue_list[4].ran_to = l
	new_command := strconv.Itoa(ran_number+1) + ",Measurement Report" + MR_command
	fmt.Printf("%s\n", new_command)
	go send_message(conn_list[ue_list[ran_number+1].ran_now], new_command)
	// RSRP_NOW = RSRP[i][ue_list[ran_number+1].ran_to]
	//RSRP_NOW = RSRP[i*season][ue_list[ran_number+1].ran_to]
	// break
	// go TIMER_10s(strconv.Itoa(ran_number + 1)) //Measurement report一定有回应，不用等
	CHO_Prepare = false
}

func TIMER_TTT_Excution(RSRP_detect []float64, RSRP_base []float64, ran_exc int) {
	t1 := time.Now()
	t2 := time.Now()
	T_target := t1.Add(time.Millisecond * time.Duration(math.Floor(TTT_Exc/float64(RSRP_Meas_Period)/5)*float64(RSRP_Meas_Period)*5)) //  TTT, 640ms定时

	index := 1
	for t2.Before(T_target) {

		time.Sleep(time.Duration(float64(RSRP_Meas_Period)*5) * time.Millisecond)
		print(RSRP_detect[index], RSRP_base[index], index, HO_threshold, CHO_Prepare, HO_Decision, CHO_Excution)
		if CHO_Prepare || RSRP_detect[index] <= RSRP_base[index]+HO_threshold || !HO_Decision || !CHO_Excution { //|| RSRP_detect[index] < float64(Hys)
			return
		}
		index = index + 1
		t2 = time.Now()
	}
	time.Sleep(time.Duration(TTT_Exc-math.Floor(TTT_Exc/float64(RSRP_Meas_Period)/5)*float64(RSRP_Meas_Period)*5) * time.Millisecond)
	// if open_uevm {
	// 	uevm_conn.Write(add_tcp_length(strconv.Itoa(ran_number+1) + ",HO to new ran," + strconv.Itoa(ran_exc)))
	// }
	// ue_list[4].ran_to = l
	new_command := strconv.Itoa(ran_number+1) + ",RACH Target_RAN||" + UL_DL
	fmt.Printf("%s\n", new_command)
	fmt.Printf("%d\n", ran_exc)
	go send_message(conn_list[ran_exc], new_command)
	if open_uevm {
		uevm_conn.Write(add_tcp_length(strconv.Itoa(ran_number+1) + ",RACH Target_RAN," + strconv.Itoa(ran_exc)))
	}
	CHO_Excution = false
	// RSRP_NOW = RSRP_detect[index-1]
	// RSRP_NOW = RSRP[i][ue_list[ran_number+1].ran_to]
	//RSRP_NOW = RSRP[i*season][ue_list[ran_number+1].ran_to]
	// break
	// go TIMER_10s(strconv.Itoa(ran_number + 1))  先不计时
}

func CHO_Timer(ueid string) {
	println("CHO Timer!\n")
	t1 := time.Now()
	t2 := time.Now()
	T_target := t1.Add(time.Second * 10) // ？
	println(t1.String(), t2.String(), T_target.String())

	for t2.Before(T_target) {
		t2 = time.Now()
		//println(t2.String())
		if !HO_Decision {
			return
		}
	}
	//HO_Decision = false  ACK后才可取消；

	ueid_int, _ := strconv.Atoi(ueid)
	command := ueid + ",CHO Configuration Delete"
	println(command)
	go send_message(conn_list[ue_list[ueid_int].ran_now], command)
	//向RAN发包清空预配置；
	//等待ACK，定时
	t1 = time.Now()
	t2 = time.Now()
	T_target = t1.Add(time.Second * 3)
	println(t1.String(), t2.String(), T_target.String())

	for t2.Before(T_target) {
		t2 = time.Now()
		//println(t2.String())
		if !HO_Decision {
			return
		}
	}
	command = ueid + ",RRC Reestablish Request"
	println(command)
	go send_message(conn_list[ue_list[ueid_int].ran_now], command)

	//等待重建立ACK，定时；

	t1 = time.Now()
	t2 = time.Now()
	T_target = t1.Add(time.Second * 3)
	println(t1.String(), t2.String(), T_target.String())

	for t2.Before(T_target) {
		t2 = time.Now()
		//println(t2.String())
		if !HO_Decision {
			return
		}
	}

	//等待RRC ACK失败，则向新RAN发起注册；
	command = ueid + ",UE Reregistration"
	println(command)
	ue_list[ueid_int].ran_now, _ = MAX_index(RSRP_ALL_NOW)
	//ue_list[ueid_int].ran_to = next_ran(ue_list[ueid_int].ran_now)
	go send_message(conn_list[ue_list[ueid_int].ran_now], command)

}
