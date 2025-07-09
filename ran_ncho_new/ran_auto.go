package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"test"
	"test/consumerTestdata/UDM/TestGenAuthData"

	// "bytes"
	// "encoding/base64"
	// "encoding/binary"
	"bufio"
	"net"
	"os"
	"sync"

	"math"
	"strconv"

	// "math/rand"

	// "testing"
	"time"

	// "test/ngapTestpacket"
	// "golang.org/x/net/html/charset"
	// "github.com/axgle/mahonia"

	// "github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require"

	// ausf_context "github.com/free5gc/ausf/context"

	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapType"

	// "github.com/free5gc/util/milenage"
	"git.cs.nctu.edu.tw/calee/sctp"
	"github.com/free5gc/nas/security"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/mongoapi"
)

var (
	Continue     = true
	Logname      = "/root/ran_ncho_new/log/" + time.Now().Format("2006-01-02_15-04-05")[5:] + ".log"
	ue_list      []UE
	ran_ueid_now = 1
	// 收到信息时由ran_ueid映射到ue_list id，由注册流程和HO流程添加
	// amf_ueid_map 		 map[int]int
	amf_ueid_map sync.Map
	ip_ueid_map  sync.Map
	// Recorder记录往返upf的信息对照。ControlRecorder记录每个ue的控制信息，最后按照每个ue的顺序记录
	// Recorder             []map[string]string
	Recorder             []sync.Map
	ControlRecorder      [][]string
	ranN2Ipv4Addr        string = "172.17.0.2"
	amfN2Ipv4Addr        string = "172.17.0.1"
	ranN3Ipv4Addr        string = "172.17.0.2"
	upfN3Ipv4Addr        string = "172.17.0.1"
	listen_controller_ip        = "172.17.0.2:9000"
	listen_ran_ip        []string
	listen_ue_ip         = "172.17.0.2:9003"
	listen_ue_port       = 9003
	pre_ranip            []string
	next_ranip           []string
	pran_id              []string
	nran_id              []string
	ran_id               string
	handover_type        string
	RAN_IDINT            int
	ran_number           int
	RAN_NUM              int = 3
	ue_number            int
	ch_log               = make(chan string, 4096)
	ch_quit              = make(chan struct{}, 1)
	amf_conn             *sctp.SCTPConn
	upf_conn             *net.UDPConn
	pran_conn            []net.Conn
	nran_conn            []net.Conn
	ue_conn              *net.UDPConn
	controller_conn      net.Conn
	timer                int = 2000
	n2_NH_map            sync.Map
	ran_N                int = 3
	RSRP_Interval        int = 1000
	RSRP                 [][]float64
	Packet_Loss_Rate     float64 = 0.0
	PLR_switch           int     = 1
	CHO_threshold        float64 = -100.0
	During_CHO           bool    = false
	CHO_Target           int     = 3
	First_RRC_ACK        bool    = true
	First_RACH           bool    = true
	PLR                  int     = 1
	ran_5g_dura          int     = 10
	ran_ran_dura         int     = 2
	STOP                 bool    = false
	package_count        int     = 0
)

func init() {
	mongoapi.SetMongoDB("free5gc", "mongodb://172.17.0.1:27017")
	if !exists("/root/ran_ncho_new/log/") {
		os.Mkdir("/root/ran_ncho_new/log/", os.ModePerm)
	}
	if !exists(Logname) {
		os.Create(Logname)
	}
	if !exists(Logname + ".control.log") {
		os.Create(Logname + ".control.log")
	}

	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() && ip.IP.To4() != nil {
			ranN2Ipv4Addr = ip.IP.String()
			ranN3Ipv4Addr = ip.IP.String()
			ran_id_int, _ := strconv.Atoi(strings.Split(ranN3Ipv4Addr, ".")[3])
			ran_id = strconv.Itoa(ran_id_int - 2)
			listen_controller_ip = ip.IP.String() + ":9000"
			for i := 1; i < RAN_NUM; i++ {
				println(ip.IP.String() + ":900" + strconv.Itoa(i))
				listen_ran_ip = append(listen_ran_ip, ip.IP.String()+":900"+strconv.Itoa(i))
			}
			println(listen_ran_ip)
			// listen_ran_ip = ip.IP.String() + ":9001"
			listen_ue_ip = ip.IP.String() + ":9003"
		}
	}
	go write_log(ch_log, ch_quit)
}

func main() {
	ch_upf := make(chan []byte, 1024)
	ch_amf := make(chan string, 1024)
	ch_controller := make(chan string, 1024)
	ch_pran := make(chan []byte, 1024)
	// 这个暂时还没用上
	ch_nran := make(chan string, 1024)
	ch_between_1 := make(chan string, 1024)
	ch_between_2 := make(chan string, 1024)

	connect_amf(ch_amf)
	go listen_controller(ch_controller)
	// 上一步确定pran和nran之后再继续
	<-ch_controller //阻塞的作用吗？
	RAN_IDINT, _ = strconv.Atoi(ran_id)
	for i := 1; i < ran_N; i++ {
		pre_ranip = append(pre_ranip, "172.17.0."+strconv.Itoa((RAN_IDINT-i+ran_N)%ran_N+2)+":900"+strconv.Itoa(i))
		pran_id = append(pran_id, strconv.Itoa((RAN_IDINT-i+ran_N)%ran_N))
		next_ranip = append(next_ranip, "172.17.0."+strconv.Itoa((RAN_IDINT+i)%ran_N+2)+":900"+strconv.Itoa(i))
		nran_id = append(nran_id, strconv.Itoa((RAN_IDINT+i)%ran_N)) // "0"
	}
	for i := 0; i < RAN_NUM-1; i++ {
		go listen_pre_ran(ch_pran, i, ch_between_1)
		go dail_next_ran(ch_nran, i, ch_between_2)
		<-ch_between_1
		<-ch_between_2
	}
	//go listen_pre_ran(ch_pran,0)
	//go dail_next_ran(ch_nran,0)
	//go listen_pre_ran(ch_pran,1)
	//go dail_next_ran(ch_nran,1)
	// go listen_pre_ran(ch_pran)
	// go dail_next_ran(ch_nran)
	go listen_ue() //暂时应该被controller代替，不需要了！

	go listen_amf(ch_pran)
	go listen_upf(ch_upf)

	for Continue {
		time.Sleep(2 * time.Second)
	}

	file, _ := os.OpenFile(Logname+".control.log", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	for i := 0; i < len(ControlRecorder); i++ {
		for j := 0; j < len(ControlRecorder[i]); j++ {
			file.WriteString(ControlRecorder[i][j] + "\n")
		}
		file.WriteString("\n")
	}
	file.Close()

	err := exec.Command("cp", "-f", Logname+".control.log", "/root/ran/latest.log").Run()
	if err != nil {
		println("error copy control log to latest log.")
	}
}

func listen_pre_ran(ch_pran chan []byte, ran_index int, ch_between_1 chan string) {
	listen, err := net.Listen("tcp", listen_ran_ip[ran_index])
	if err != nil {
		fmt.Printf("listen failed, err:%v\n", err)
		return
	}
	fmt.Printf("listen %v start\n", listen_ran_ip[ran_index])
	pran_conn_i, err := listen.Accept()
	if err != nil {
		fmt.Printf("accept failed, err:%v\n", err)
		return
	}
	fmt.Printf("listen %v accept\n", listen_ran_ip[ran_index])
	listen.Close()

	pran_conn = append(pran_conn, pran_conn_i)
	println(ran_index, pran_conn)

	ch_between_1 <- "ok"

	reader := bufio.NewReader(pran_conn[ran_index])
	// var buf [1024]byte

	if handover_type == "xn" {
		// 第0步，建立xn setup
		raw_length, _ := reader.Peek(2)
		length := uint16(raw_length[0])<<8 + uint16(raw_length[1])
		buf := make([]byte, length+2)
		reader.Read(buf)
		recv := string(buf[2:])
		// n, _ := reader.Read(buf[:])
		// recv := string(buf[:n])
		// ** senario * trans_delay /ran to ran/
		time.Sleep(time.Duration(ran_ran_dura) * time.Millisecond)
		t := time.Now().Format("2006-01-02 15:04:05.000000")
		XnSetupRequest := recv + ",ran" + ran_id + " recv from ran" + pran_id[ran_index] + "," + strings.Split(t, " ")[1]
		ControlRecorder[0] = append(ControlRecorder[0], XnSetupRequest)
		ch_log <- XnSetupRequest + "\n"

		t = time.Now().Format("2006-01-02 15:04:05.000000")
		XnSetupResponse := "0,Xn Setup Response,ran" + ran_id + " send to ran" + pran_id[ran_index] + "," + strings.Split(t, " ")[1]

		pran_conn[ran_index].Write(add_tcp_length(XnSetupResponse))
		ControlRecorder[0] = append(ControlRecorder[0], XnSetupResponse)
		ch_log <- XnSetupResponse + "\n"

		for Continue {
			raw_length, err := reader.Peek(2)
			// n, err := reader.Read(buf[:])
			if err != nil {
				if err == io.EOF {
					println("pran close the connection.")
					pran_conn[ran_index].Close()
					return
				}
				select {
				case <-ch_quit:
					pran_conn[ran_index].Close()
					return
				default:
					fmt.Printf("Read from pran_conn failed, err:%v\n", err)
				}
			}
			length := uint16(raw_length[0])<<8 + uint16(raw_length[1])
			buf := make([]byte, length+2)
			reader.Read(buf)
			recv := string(buf[2:])
			// ** senario * trans_delay /ran to ran/
			time.Sleep(time.Duration(ran_ran_dura) * time.Millisecond)
			// recv := string(buf[:n])
			t := time.Now().Format("2006-01-02 15:04:05.000000")
			content := recv + ",ran" + ran_id + " recv from ran" + pran_id[ran_index] + "," + strings.Split(t, " ")[1]

			l := strings.Split(recv, ",")
			ueid, _ := strconv.Atoi(l[0])
			command := l[1]
			ControlRecorder[ueid] = append(ControlRecorder[ueid], content)
			ch_log <- content + "\n"

			switch command {
			case "Condition Handover Request":
				go excuteXnHandover_tRAN(ueid, ran_index)
			case "Early Status Transfer":
				ue_list[ueid].ch <- []byte{}
			case "SN Status Transfer":
				//ue_list[ueid].ch <- []byte{}
			case "Configuration Delete":
				ue_list[ueid].ch <- []byte("Deleted")
				CHO_Target = 3
				During_CHO = false
			}
		}
	} else if handover_type == "n2" {
		// 需要处理一个开启excuteN2Handover2的任务
		var amf_ueid int
		var NH []uint8 = make([]uint8, 1)
		var teid string = ""
		for Continue {
			HandoverRequest := <-ch_pran
			if HandoverRequest[16] == 29 {
				amf_ueid = int(HandoverRequest[13])*256 + int(HandoverRequest[14])
			} else if HandoverRequest[15] == 29 {
				amf_ueid = int(HandoverRequest[13])
			}
			// NH的标志[93 0 33 16]最少是从51下标开始的，暂且读过20个循环
			for i := 50; i < 70; i++ {
				// fmt.Printf("%v\n", HandoverRequest[i:i+4])
				if bytes.Equal(HandoverRequest[i:i+4], []byte{93, 0, 33, 16}) {
					NH = HandoverRequest[i+4 : i+36]
					fmt.Printf("NH recv from amf: %v\n", NH)
					break
				}
			}
			if len(NH) != 32 {
				println("read NH fail, abort! amf_ueid:", amf_ueid)
				continue
			}
			ch_n2 := make(chan string, 1024)
			// []byte无法作为哈希表的键，使用string
			n2_NH_map.Store(string(NH), ch_n2)

			// teid的标志最少是从111下标开始的，暂且读过10个循环
			for i := 103; i < 113; i++ {
				if bytes.Equal(HandoverRequest[i:i+4], []byte{172, 17, 0, 1}) {
					teid = hex.EncodeToString(HandoverRequest[i+4 : i+8])
					break
				}
			}
			go excuteN2Handover_tRAN(amf_ueid, teid, ch_n2, ran_index)
			NH = make([]uint8, 1)
			teid = ""
		}
	}
}

func dail_next_ran(ch_nran chan string, ran_index int, ch_between_2 chan string) {
	var err error
	var nran_conn_i net.Conn
	nran_conn_i, err = net.Dial("tcp", next_ranip[ran_index])
	for err != nil {
		time.Sleep(1 * time.Second)
		nran_conn_i, err = net.Dial("tcp", next_ranip[ran_index])
	}
	fmt.Printf("dail %v success\n", next_ranip[ran_index])
	nran_conn = append(nran_conn, nran_conn_i)
	println(ran_index, nran_conn)

	ch_between_2 <- "ok"

	reader := bufio.NewReader(nran_conn[ran_index])
	// var buf [1024]byte

	if handover_type == "xn" {
		// 第0步，建立xn setup
		t := time.Now().Format("2006-01-02 15:04:05.000000")
		XnSetupRequest := "0,Xn Setup Request,ran" + ran_id + " send to ran" + nran_id[ran_index] + "," + strings.Split(t, " ")[1]

		_, err = nran_conn[ran_index].Write(add_tcp_length(XnSetupRequest))
		if err != nil {
			fmt.Printf("send failed, err:%v\n", err)
			return
		}

		ControlRecorder[0] = append(ControlRecorder[0], XnSetupRequest)
		ch_log <- XnSetupRequest + "\n"

		time.Sleep(10 * time.Millisecond)

		raw_length, _ := reader.Peek(2)
		length := uint16(raw_length[0])<<8 + uint16(raw_length[1])
		buf := make([]byte, length+2)
		reader.Read(buf)
		recv := string(buf[2:])
		// n, _ := reader.Read(buf[:])
		// recv := string(buf[:n])
		t = time.Now().Format("2006-01-02 15:04:05.000000")
		XnSetupResponse := recv + ",ran" + ran_id + " recv from ran" + nran_id[ran_index] + "," + strings.Split(t, " ")[1]
		ControlRecorder[0] = append(ControlRecorder[0], XnSetupResponse)
		ch_log <- XnSetupResponse + "\n"

		for Continue {
			raw_length, err := reader.Peek(2)
			// n, err := reader.Read(buf[:])
			if err != nil {
				if err == io.EOF {
					println("nran close the connection.")
					nran_conn[ran_index].Close()
					return
				}
				select {
				case <-ch_quit:
					nran_conn[ran_index].Close()
					return
				default:
					fmt.Printf("Read from nran_conn failed, err:%v\n", err)
				}
			}
			length := uint16(raw_length[0])<<8 + uint16(raw_length[1])
			buf := make([]byte, length+2)
			reader.Read(buf)
			recv := string(buf[2:])
			// recv := string(buf[:n])
			// ** senario * trans_delay /ran to ran/
			time.Sleep(time.Duration(ran_ran_dura) * time.Millisecond)

			t := time.Now().Format("2006-01-02 15:04:05.000000")
			content := recv + ",ran" + ran_id + " recv from ran" + nran_id[ran_index] + "," + strings.Split(t, " ")[1]

			l := strings.Split(recv, ",")
			ueid, _ := strconv.Atoi(l[0])
			command := l[1]
			ControlRecorder[ueid] = append(ControlRecorder[ueid], content)
			ch_log <- content + "\n"

			switch command {
			case "Condition Handover Request Acknowledge":
				ue_list[ueid].ch <- []byte{}
			case "UE Context Release(HO Success)":
				ue_list[ueid].state = initial
				ue_list[ueid].ch <- []byte{}
				First_RRC_ACK = true
				First_RACH = true
				PLR = 1
				CHO_Target = 3
				During_CHO = false
			case "CHO Configuration Deleted":
				ue_list[ueid].ch <- []byte{}
				First_RRC_ACK = true
				First_RACH = true
				//PLR = 1
				CHO_Target = 3
				During_CHO = false
			}
		}
	} else if handover_type == "n2" {
		// 似乎没有需要处理的呢
	}
}

func connect_amf(ch_amf chan string) {
	// var n int
	var err error
	var sendMsg []byte
	var recvMsg = make([]byte, 2048)
	amf_conn, err = test.ConnectToAmf(amfN2Ipv4Addr, ranN2Ipv4Addr, 38412, 9487)
	if err != nil {
		fmt.Printf("create amf_conn failed, err:%v\n", err)
	}
	upf_conn, err = test.ConnectToUpf(ranN3Ipv4Addr, upfN3Ipv4Addr, 2152, 2152)
	if err != nil {
		fmt.Printf("create upf_conn failed, err:%v\n", err)
	}
	// 经过测试，gnb_id的字段可以不需要更改，只需要唯一的ran name即可完成注册
	// 但是这种注册方式5gc会认为所有基站挂在同一个gnb下，会由5gc分配未知（找不到地方获取）的cell_id，从而给n2 handover环节带来困难
	// 因此使用递增的gnb_id进行注册，ran name则无需变动。经过测试，这种方式对于n2和xn都可以执行，且handover required命令中cell_id固定为"\x01\x20"即可
	// 另外在注册和HO的命令中，会使用[]byte("\x00\x01\x02")和[]byte{0x00, 0x0b, 0x0f}等形式，似乎结果是一致的，即"\x2f"和0x2f是相同的
	// RAN2 send Second NGSetupRequest Msg
	// sendMsg, err = test.GetNGSetupRequest([]byte("\x00\x01\x02"), 24, "ran"+ran_id)
	gnbid, _ := strconv.Atoi(ran_id)
	sendMsg, _ = test.GetNGSetupRequest([]byte{0x00, 0x01, uint8(gnbid + 1)}, 24, "ran"+ran_id)
	amf_conn.Write(sendMsg)

	// RAN2 receive Second NGSetupResponse Msg
	amf_conn.Read(recvMsg)
}

// listen conn程序只做基本的收包，尽量不阻塞。然后通过chan发送给对应函数推进下一步
func listen_amf(ch_pran chan []byte) {
	// defer amf_conn.Close()
	var recvMsg = make([]byte, 2048)
	for Continue {
		n, err := amf_conn.Read(recvMsg)
		if err != nil {
			if err == io.EOF {
				println("AMF close the connection.")
				amf_conn.Close()
				return
			}
			select {
			case <-ch_quit:
				amf_conn.Close()
				return
			default:
				fmt.Printf("Read from amf_conn failed, err:%v\n", err)
			}
		}
		// fmt.Printf("recv command from amf: %v\n", recvMsg[:n])
		// println("len amf msg:", n, "procedure code:", recvMsg[1])
		go handle_amf_message(string(recvMsg[:n]), ch_pran)
	}
}

func handle_amf_message(recv_s string, ch_pran chan []byte) {
	var amf_ueid int
	var ran_ueid int
	var title string
	recv := []byte(recv_s)
	// recv := []byte(recv_str)
	t := time.Now().Format("2006-01-02 15:04:05.000000")
	// ngapPdu, _ := ngap.Decoder(recv)
	// fmt.Printf("%v\n", recv)
	// println("code:", ngapPdu.InitiatingMessage.ProcedureCode.Value)
	// 也可以直接查recv[1]的值确定ProcedureCode
	switch int64(recv[1]) {
	// switch ngapPdu.InitiatingMessage.ProcedureCode.Value {
	// NAS Authentication Request Msg
	case ngapType.ProcedureCodeDownlinkNASTransport:
		if recv[15] == 85 {
			amf_ueid = int(recv[12])*256 + int(recv[13])
			if recv[18] == 0 {
				ran_ueid = int(recv[19])
			} else {
				ran_ueid = int(recv[19])<<8 | int(recv[20])
			}
		} else if recv[14] == 85 {
			amf_ueid = int(recv[12])
			if recv[17] == 0 {
				ran_ueid = int(recv[18])
			} else {
				ran_ueid = int(recv[18])<<8 | int(recv[19])
			}
		}
		if len(recv) < 54 {
			title = ",Security Mode Command,"
		} else {
			title = ",Authentication Request,"
			ran_id_int, _ := strconv.Atoi(ran_id)
			ueid := 1 + ran_id_int
			if ran_ueid != 1 {
				ueid = ran_ueid + ran_number - 1
			}
			amf_ueid_map.Store(amf_ueid, ueid)
			ue_list[ueid].ue.AmfUeNgapId = int64(amf_ueid)
		}
	// Initial Context Setup Request Msg
	case ngapType.ProcedureCodeInitialContextSetup:
		if recv[16] == 85 {
			amf_ueid = int(recv[13])*256 + int(recv[14])
		} else if recv[15] == 85 {
			amf_ueid = int(recv[13])
		}
		title = ",Initial Context Setup Request,"
	// PDU Session Resource Setup Request
	case ngapType.ProcedureCodePDUSessionResourceSetup:
		if recv[16] == 85 {
			amf_ueid = int(recv[13])*256 + int(recv[14])
		} else if recv[15] == 85 {
			amf_ueid = int(recv[13])
		}
		title = ",PDU Session Resource Setup Request,"
	// Handover Request
	case ngapType.ProcedureCodeHandoverResourceAllocation:
		// fmt.Printf("Handover Request: %v\n", recv)
		if recv[16] == 29 {
			amf_ueid = int(recv[13])*256 + int(recv[14])
		} else if recv[15] == 29 {
			amf_ueid = int(recv[13])
		}
		title = ",Handover Request,"
	// Handover Command
	case ngapType.ProcedureCodeHandoverPreparation:
		if recv[15] == 85 {
			amf_ueid = int(recv[12])*256 + int(recv[13])
		} else if recv[14] == 85 {
			amf_ueid = int(recv[12])
		}
		title = ",Handover Command,"
	// UE Context Release
	case ngapType.ProcedureCodeUEContextRelease:
		if recv[13] == 0 {
			amf_ueid = int(recv[12])
		} else {
			amf_ueid = int(recv[12])*256 + int(recv[13])
		}
		title = ",UE Context Release,"
	// Path Switch Request Acknowledge
	case ngapType.ProcedureCodePathSwitchRequest:
		if recv[15] == 85 {
			amf_ueid = int(recv[12])*256 + int(recv[13])
		} else if recv[14] == 85 {
			amf_ueid = int(recv[12])
		}
		// amf_ueid_map[amf_ueid] = amf_ueid
		amf_ueid_map.Store(amf_ueid, amf_ueid)
		title = ",Path Switch Request Acknowledge,"
	default:
		fmt.Printf("invalid recv command from amf: %v\n", recv)
		return
	}

	// println("amf_ueid:", amf_ueid)
	// println("title:", title)
	// ueid, ok := amf_ueid_map[amf_ueid]
	ueid_interface, ok := amf_ueid_map.Load(amf_ueid)
	if !ok {
		// if ngapPdu.InitiatingMessage.ProcedureCode.Value == ngapType.ProcedureCodeHandoverResourceAllocation {
		if int64(recv[1]) == ngapType.ProcedureCodeHandoverResourceAllocation {
			ch_pran <- recv
			// 因为此时还没有对应ueid，日志记录在excuteN2handover中单独解决
		} else {
			println("invalid amf_ueid from amf:", amf_ueid)
		}
		return
	}
	ueid := ueid_interface.(int)
	// println("ueid:", ueid)
	ue_list[ueid].ch <- recv
	// println("ue_list[ueid].ch <- recv")
	content := strconv.Itoa(ueid) + title + "ran" + ran_id + " recv from amf," + strings.Split(t, " ")[1]
	ControlRecorder[ueid] = append(ControlRecorder[ueid], content)
	ch_log <- content + "\n"
}

// 发出ping的gtp头部有12字节，返回ping的gtp头部主要有2种：
// 给ueid=1的信息，头部16字节：[52 255 0 56 0 0 0 1 0 0 0 133 1 0 9 0]
// 给ueid!=1的信息，头部8字节：[48 255 0 48 0 0 0 1]
// gtp头部的构造和解析参考：
// https://blog.csdn.net/yeyiqun/article/details/102548114
// https://blog.csdn.net/u010178611/article/details/81909857
// 需要先去除gtp头部
func listen_upf(ch_upf chan []byte) {
	// var index int
	var recvMsg = make([]byte, 2048)
	for Continue {
		// 读gtp header固定头部
		n, err := upf_conn.Read(recvMsg)
		if err != nil {
			if err == io.EOF {
				println("end of upf")
				upf_conn.Close()
				return
			}
			select {
			case <-ch_quit:
				upf_conn.Close()
				return
			default:
				fmt.Printf("Read from upf_conn failed, err:%v\n", err)
			}
		}
		// index++
		go dail_controller_and_ue(bytes.Clone(recvMsg[:n]))
	}
}

// listen conn程序只做基本的收包，尽量不阻塞。然后通过chan发送给对应函数推进下一步
// 只有从upf收到的消息需要经过特定解析发给controller，因此这部分作为listen_upf的线程
func dail_controller_and_ue(raw []byte) {
	t := time.Now().Format("2006-01-02 15:04:05.000000")
	// 数据包去掉gtp固定头部后的总长
	// fmt.Printf("fixed gtp header: %v\n", raw[:8])
	length := int(raw[2])<<8 + int(raw[3])
	if length == 0 {
		fmt.Println("receive N3 end marker:", raw)
		// [48 254 0 0 0 0 0 1] N3 end marker
		return
	}
	// fmt.Printf("upf package without fixed gtp header: %v\n", raw[8:])
	// 去除gtp拓展头部
	var ex_gtp byte = 8
	// 标志位只要有一个，就有扩展字段
	if (raw[0] & 0b00000111) != 0 {
		ex_gtp += 4
		if (raw[0] & 0b00000100) == 4 {
			for ; raw[ex_gtp-1] != 0; ex_gtp += (raw[ex_gtp] << 2) {
			}
		}
	}

	ip_msg := raw[ex_gtp:]
	// fmt.Printf("ip part of upf msg: %v\n", ip_msg)

	// ping消息固定使用百度的一个cdn ip 110.242.68.66。这样可以区分，非这个地址的就是其他app的消息
	if bytes.Equal(ip_msg[12:16], []byte{110, 242, 68, 66}) {
		ueid := int(ip_msg[25]) | int(ip_msg[24])<<8
		seq := fmt.Sprintf("%d", int(ip_msg[27])+int(ip_msg[26])<<8)
		// fmt.Printf("recv message from upf. ueid: %v , seq: %v\n", ueid, seq)
		println("ueid: ", ueid, ", seq: ", seq, ",  received from upf.")

		// 检查记录是否存在
		source_reocrd, ok := Recorder[ueid].Load(seq)
		if !ok {
			println("No record found! skip.")
			return
		}
		update_record := source_reocrd.(string) + ",ran" + ran_id + " recv from upf," + strings.Split(t, " ")[1]
		Recorder[ueid].Store(seq, update_record)
		t = time.Now().Format("2006-01-02 15:04:05.000000")
		content := update_record + ",ran" + ran_id + " send to controller," + strings.Split(t, " ")[1]
		// fmt.Printf("send content：%v\n", content)

		controller_conn.Write(add_tcp_length(content))
		// controller_conn.WriteToUDP([]byte(content), controller_addr)
		ch_log <- content + "\n"
		return
		// ueid_interface, ok := ip_ueid_map.Load(string(ip_msg[16:20]))
		// if !ok {
		// 	println("No ueid found! skip.")
		// 	return
		// }
		// ueid := ueid_interface.(int)
		// // uevm能够为每个网卡设置ip，这里也就不需要改IP了
		// // length := uint16(len(ip_msg))
		// // println("ueid: ", ueid, "length:", length)
		// // header := []byte{uint8(length >> 8), uint8(length)}
		// // ue_list[ueid].conn.Write(append(header, ip_msg...))
		// // _, err := ue_list[ueid].conn.Write(ip_msg)
		// _, err := ue_conn.WriteToUDP(ip_msg, ue_list[ueid].addr)
		// if err != nil {
		// 	fmt.Printf("ue conn close. ueid:%v, err:%v\n", ueid, err)
		// }
		// // content := fmt.Sprintf("index: %v, ip: %v", index, ip_msg)
		// // ch_log <- content + "\n"
		// return
	}
	ueid_interface, ok := ip_ueid_map.Load(string(ip_msg[16:20]))
	if !ok {
		println("No ueid found! skip.")
		return
	}
	ueid := ueid_interface.(int)
	// ueid := ip_msg[19]
	// uevm能够为每个网卡设置ip，这里也就不需要改IP了
	// length := uint16(len(ip_msg))
	// println("ueid: ", ueid, "length:", length)
	// header := []byte{uint8(length >> 8), uint8(length)}
	// ue_list[ueid].conn.Write(append(header, ip_msg...))
	if PLR == 1 {
		_, err := ue_conn.WriteToUDP(ip_msg, ue_list[ueid].addr)
		// _, err := ue_list[ueid].conn.Write(ip_msg)
		if err != nil {
			fmt.Printf("ue conn close. ueid:%v, err:%v\n", ueid, err)
		}
	} else {
		return
	}
	// _, err := ue_conn.WriteToUDP(ip_msg, ue_list[ueid].addr)
	// // _, err := ue_list[ueid].conn.Write(ip_msg)
	// if err != nil {
	// 	fmt.Printf("ue conn close. ueid:%v, err:%v\n", ueid, err)
	// }
	// var ueid int
	// var seq string
	// if raw[0] == 52 {
	// 	ueid = int(raw[41]) | int(raw[40])<<8
	// 	seq = fmt.Sprintf("%d", int(raw[43])+int(raw[42])<<8)
	// } else {
	// 	// if ueid == 0 || seq == "0" {
	// 	ueid = int(raw[33]) | int(raw[32])<<8
	// 	seq = fmt.Sprintf("%d", int(raw[35])+int(raw[34])<<8)
	// }
	// ueid := int(ip_msg[25]) | int(ip_msg[24])<<8
	// seq := fmt.Sprintf("%d", int(ip_msg[27])+int(ip_msg[26])<<8)
	// // fmt.Printf("recv message from upf. ueid: %v , seq: %v\n", ueid, seq)
	// println("ueid: ", ueid, ", seq: ", seq, ",  received from upf.")

	// // 检查记录是否存在
	// source_reocrd, ok := Recorder[ueid].Load(seq)
	// if !ok {
	// 	println("No record found! skip.")
	// 	return
	// }
	// update_record := source_reocrd.(string) + ",ran" + ran_id + " recv from upf," + strings.Split(t, " ")[1]
	// Recorder[ueid].Store(seq, update_record)
	// t = time.Now().Format("2006-01-02 15:04:05.000000")
	// content := update_record + ",ran" + ran_id + " send to controller," + strings.Split(t, " ")[1]
	// // fmt.Printf("send content：%v\n", content)
	// // 发包做丢包处理吗？？ 有重传机制吗，ue会重新注册吗？
	// controller_conn.Write(add_tcp_length(content))
	// ch_log <- content + "\n"
}

// listen conn程序只做基本的收包，尽量不阻塞。然后通过chan发送给对应函数推进下一步
func listen_controller(ch_controller chan string) {
	// 建立 tcp 服务
	listen, err := net.Listen("tcp", listen_controller_ip)
	if err != nil {
		fmt.Printf("listen failed, err:%v\n", err)
		return
	}
	fmt.Printf("listen %v start\n", listen_controller_ip)
	// 等待客户端建立连接
	controller_conn, err = listen.Accept()
	if err != nil {
		fmt.Printf("accept failed, err:%v\n", err)
		return
	}
	fmt.Printf("listen controller accept\n")
	listen.Close()

	reader := bufio.NewReader(controller_conn)
	// var buf [1024]byte

	// controller发送第一个命令，给出ran总数，handover方式，ue数量。
	// n, _ := reader.Read(buf[:])
	// recv := string(buf[:n])
	raw_length, _ := reader.Peek(2)
	length := uint16(raw_length[0])<<8 + uint16(raw_length[1])
	buf := make([]byte, length+2)
	reader.Read(buf)
	recv := string(buf[2:])
	recv_split := strings.Split(recv, ",")
	handover_type = recv_split[1]
	println(handover_type)
	// 假设5个ran，则最后ran的ip是172.17.0.6
	ran_number, _ = strconv.Atoi(recv_split[0])
	ue_number, _ = strconv.Atoi(recv_split[2])

	// 这俩recorder需要初始化
	Recorder = make([]sync.Map, ue_number+ran_number+1)
	ControlRecorder = make([][]string, ue_number+ran_number+1)

	// last_ranip := strconv.Itoa(ran_number + 1)
	// RAN_IDINT, _  = strconv.Atoi(ran_id)
	// for i := 1; i < ran_number; i++ {
	// 	pre_ranip = append(pre_ranip, "172.17.0." + strconv.Itoa((RAN_IDINT - i + ran_number)%ran_number + 2) + ":900" + strconv.Itoa(i))
	// 	pran_id = append(pran_id, strconv.Itoa((RAN_IDINT - i + ran_number)%ran_number))
	// 	next_ranip = append(next_ranip, "172.17.0." + strconv.Itoa((RAN_IDINT + i)%ran_number + 2) + ":900" + strconv.Itoa(i) )
	// 	nran_id = append(nran_id, strconv.Itoa((RAN_IDINT + i)%ran_number))   // "0"
	// }
	// if ranN2Ipv4Addr == "172.17.0."+last_ranip {
	// 	// 其实pre_ranip没用到，因为目前是监听pre ran，不需要指定ip和端口
	// 	for i := 1; i < ran_number; i++ {
	// 		pre_ranip[i-1] = "172.17.0." + strconv.Itoa(ran_number) + ":9001"
	// 		pran_id[i-1] = strconv.Itoa(ran_number - 2)
	// 		next_ranip[i-1] = "172.17.0.2:9001"
	// 		nran_id[i-1] = "0"
	// 	}

	// } else if ranN2Ipv4Addr == "172.17.0.2" {
	// 	pre_ranip = "172.17.0." + last_ranip + ":9001"
	// 	pran_id = strconv.Itoa(ran_number - 1)
	// 	next_ranip = "172.17.0.3:9001"   //需要调整的地方，不能固定死下一个ran的ip地址
	// 	nran_id = "1"
	// } else {
	// 	this_ip, _ := strconv.Atoi(strings.Split(ranN2Ipv4Addr, ".")[3])
	// 	pre_ranip = "172.17.0." + strconv.Itoa(this_ip-1) + ":9001"
	// 	pran_id = strconv.Itoa(this_ip - 3)
	// 	next_ranip = "172.17.0." + strconv.Itoa(this_ip+1) + ":9001"
	// 	nran_id = strconv.Itoa(this_ip - 1)
	// }
	// 这里OK以后再开始连接其他ran  上面阻塞等的是这个
	ch_controller <- "ok"

	// ue_list的index就是ue的唯一标识，ran与controller维护相同长度的ue_list。同时ran需要做这个index与amf_ueid/ran_ueid的转译。
	ue_list = make([]UE, ran_number+ue_number+1)
	// amf_ueid_map = make(map[int]int, ran_number+ue_number+1)
	for id := 0; id < ran_number+ue_number+1; id++ {
		// 需要在这里注册完成ue *test.RanUeContext实体，后面handover时tRAN会直接用到，按照顺序handover的话里面的数值都不需要变。
		// 这里0号是占位ue，不使用
		// 1 ~ ran_number 号是各ran为自己注册的背景流量ue，也只会根据controller命令注册属于自己的那个ue
		// ran_number+1 ~ ran_number+ue_number 号是参与handover的ue。通过controller命令，一开始只在ran0上注册，后续再移动到其他ran。但是所有ran维护相同结构的ue_list表
		imsi, _ := strconv.ParseInt("2089300000000", 16, 64)
		imsi += int64(id + id/15)
		ran_ue_id := 1
		if id > ran_number {
			ran_ue_id = id - (ran_number - 1)
		}
		ue := test.NewRanUeContext("imsi-"+strconv.FormatInt(imsi, 16), int64(ran_ue_id), security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2,
			models.AccessType__3_GPP_ACCESS)
		ue.AmfUeNgapId = int64(id)
		ue.AuthenticationSubs = test.GetAuthSubscription(TestGenAuthData.MilenageTestSet19.K,
			TestGenAuthData.MilenageTestSet19.OPC,
			TestGenAuthData.MilenageTestSet19.OP)
		ue_list[id] = UE{
			state: initial,
			IP:    "10.60.0.0",
			TEID:  "00000000",
			NH:    []byte{0},
			ch:    make(chan []byte, 1024),
			ue:    ue,
		}
		// if id < len(Recorder) {
		// 	Recorder[id] = make(map[string]string, 100)
		// } else {
		// 	Recorder = append(Recorder, make(map[string]string, 100))
		// }
	}

	for Continue {
		raw_length, err := reader.Peek(2)
		// n, err := reader.Read(buf[:])
		if err != nil {
			if err == io.EOF {
				println("Controller close the connection.")
				release()
				// controller_conn.Close()
				return
			}
			select {
			case <-ch_quit:
				controller_conn.Close()
				return
			default:
				fmt.Printf("Read from controller_conn failed, err:%v\n", err)
			}
		}
		length := uint16(raw_length[0])<<8 + uint16(raw_length[1])
		buf := make([]byte, length+2)
		reader.Read(buf)

		if strings.Split(string(buf[2:]), ":")[0] == "PLR" {
			go handle_controller_package(string(buf[2:]))
		} else if PLR == 1  {
			go handle_controller_package(string(buf[2:]))
		} else {
			//fmt.Printf("Packet Loss!\n")
			continue
		}
		// if rand.Float64() >= Packet_Loss_Rate && PLR_switch == 1 {   //收包丢包处理
		// 	go handle_controller_package(string(buf[2:]))
		// } else {
		// 	fmt.Printf("Packet Loss:",Packet_Loss_Rate,"\n")
		// 	continue
		// }
		//go handle_controller_package(string(buf[2:]))
		// go handle_controller_package(string(buf[:n]))
		// ch_controller <- content
	}
}

func handle_controller_package(recv string) {
	// println(recv)
	l := strings.Split(recv, ",")
	ll := strings.Split(recv, ":")
	ueid, _ := strconv.Atoi(l[0])
	full_command := l[1]
	RSRP_command := ll[0]
	command := strings.Split(full_command, "||")[0]
	var rsrp_temp []float64

	// 非5gc标准流程过程放在这里，不计入日志 那是什么过程？？
	if command == "Authentication Request" {
		// println("更新ue:", ueid)
		// 不能直接使用l[2]，因为这一串乱码可能会变成","符号导致意外截断命令
		new_recv_split := strings.Split(recv, "Authentication Request,")
		AuthenticationRequest := []byte(new_recv_split[1])
		// fmt.Printf("%v\n", AuthenticationRequest)
		ngapPdu, err := ngap.Decoder(AuthenticationRequest)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		nasPdu := test.GetNasPdu(ue_list[ueid].ue, ngapPdu.InitiatingMessage.Value.DownlinkNASTransport)
		rand := nasPdu.AuthenticationRequest.GetRANDValue()
		ue_list[ueid].ue.DeriveRESstarAndSetKey(ue_list[ueid].ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")
		return
	} else if command == "Allocate IP & TEID" {
		fmt.Printf("UE_LIST: %v\n", l)
		ue_list[ueid].IP = l[2]
		ip_ueid_map.Store(string(net.ParseIP(l[2]).To4()), ueid)
		ue_list[ueid].TEID = l[3]
		return
	}

	t := time.Now().Format("2006-01-02 15:04:05.000000")
	content := recv + ",ran" + ran_id + " recv from controller," + strings.Split(t, " ")[1]
	// fmt.Printf("recv content：%v\n", content)
	ch_log <- content + "\n"
	// println(command)
	//println(recv)

	// handover confirm比较特殊，单独处理
	// if found, _ := regexp.MatchString("Handover Confirm", command); found {
	if command == "Handover Confirm" {
		// 日志放在n2流程里记录
		// hash表的键就是用的string，这里就不用转[]byte了
		NH := strings.Split(full_command, "||")[2]
		fmt.Printf("NH recv from controller: %v\n", []byte(NH))
		ch_interface, ok := n2_NH_map.Load(NH)
		if !ok {
			fmt.Printf("failed to map ue NH to amf NH, ueid: %v , NH:\n%v\n", ueid, NH)
			return
		}
		ch_interface.(chan string) <- content
		n2_NH_map.Delete(NH)
		ue_list[ueid].NH = []byte(NH)
		return
	}

	fmt.Println(command)

	switch command {
	case "Quit":
		// Continue = false
		release()

	case "Connect UE":
		// println(command)
		go registration(ueid)

	case "UE Reregistration":
		STOP = true //initial ?
		go registration(ueid)

	case "HO to next ran":
		if handover_type == "xn" {
			//go excuteXnHandover_sRAN(ueid)
		} else if handover_type == "n2" {
			go excuteN2Handover_sRAN(ueid, 0)
		}

	case "RRC Reconfiguration Complete":
		if First_RRC_ACK {
			ControlRecorder[ueid] = append(ControlRecorder[ueid], content)
			ch_log <- content + "\n" //计入日志的管道
			ue_list[ueid].ch <- []byte(strings.Split(full_command, "||")[1])
			First_RRC_ACK = false
		}

	case "RACH Target_RAN":
		if First_RACH {
			ch_log <- content + "\n" //计入日志的管道
			ue_list[ueid].ch <- []byte(strings.Split(full_command, "||")[1])
			First_RACH = false
		}

	case "APP Send": //是哪个功能？？测试iperf等？
		// 去掉头部ueid和",APP Send,"之后就是完整数据包
		fmt.Printf("recv: %v\n", []byte(recv[len(l[0])+10:]))
		upf_conn.Write([]byte(recv[len(l[0])+10:]))

	case "CHO Configuration Delete":
		ue_list[ueid].ch <- []byte("Delete")

	case "RRC Reestablish Request":
		//ue_list[ueid].ch <- []byte("Reestablish")
		t = time.Now().Format("2006-01-02 15:04:05.000000")
		content := l[0] + ",RRC Reestablished,ran" + ran_id + " send to controller," + strings.Split(t, " ")[1]
		fmt.Printf("send content:%v\n", content)

		controller_conn.Write(add_tcp_length(content))

	case "Measurement Report":
		index_mr := 0 
		for index_mr < ran_N {
			BS_rsrp, _ := strconv.ParseFloat(l[3+2*index_mr],64)
			rsrp_temp = append(rsrp_temp,BS_rsrp)
			index_mr = index_mr + 1
		}
		index_mr = 1
		ran_id_int, _ := strconv.Atoi(ran_id)
		// if package_count < 1000 {
		// 	CHO_threshold = -140
		// }else{
		// 	CHO_threshold = -80
		// }
		for index_mr < ran_N {
		    if (rsrp_temp[(ran_id_int+index_mr)%ran_N] >= CHO_threshold && !During_CHO) && (ran_id_int+index_mr)%ran_N != CHO_Target {
			// if (rsrp_temp[(ran_id_int+index_mr)%ran_N] >= CHO_threshold + rsrp_temp[ran_id_int] && !During_CHO) && (ran_id_int+index_mr)%ran_N != CHO_Target { //&& rsrp_temp[(ran_id_int+index_mr)%ran_N] >= -110 
			    // 触发Handover并break； 要更改nran_id的
			    CHO_Target = (ran_id_int + index_mr) % ran_N
			    if index_mr == 1 && rsrp_temp[(ran_id_int+index_mr+1)%ran_N] > rsrp_temp[(ran_id_int+index_mr)%ran_N] {
				    CHO_Target = (ran_id_int + index_mr + 1) % ran_N
			    }
			    println(CHO_Target)
			    go excuteXnHandover_sRAN(ueid, CHO_Target, index_mr)

			    During_CHO = true
				break
		    }
			index_mr = index_mr + 1
		}

	default:
		seq, err := strconv.Atoi(command)
		if RSRP_command == "RSRP" {
			go analyse_rsrp(ll[1], 4)
		} else if RSRP_command == "PLR" {
			PLR, _ = strconv.Atoi(strings.Split(l[0], ":")[1])
			package_count = package_count + 1
			println(PLR)
		} else if err != nil {
			println("Undefined Command:", command)
		} else {
			//println(Recorder)
			// println(Recorder[ueid])
			dummyPackage := createDummyPackage(ueid, seq)
			//fmt.Printf("DUMMPY: %v\n", dummyPackage)
			t = time.Now().Format("2006-01-02 15:04:05.000000")
			content += ",ran" + ran_id + " send to upf," + strings.Split(t, " ")[1]
			Recorder[ueid].Store(command, content)
			ch_log <- content + "\n"
			//fmt.Println("command: ", command)
			upf_conn.Write(dummyPackage)
			go message_timer(ueid, command, content)
		}
	}
}

func listen_ue() {
	// listen, err := net.Listen("tcp", listen_ue_ip)
	// if err != nil {
	// 	fmt.Printf("listen failed, err:%v\n", err)
	// 	return
	// }
	// fmt.Printf("listen %v start\n", listen_ue_ip)
	// for Continue {
	// 	conn, err := listen.Accept()
	// 	if err != nil {
	// 		fmt.Printf("accept failed, err:%v\n", err)
	// 		return
	// 	}
	// 	go listen_ue_process(conn)
	// }
	// println("listen udp 172.17.0.2:9002")
	var err error
	addr := &net.UDPAddr{IP: net.ParseIP(ranN2Ipv4Addr), Port: listen_ue_port}
	ue_conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("listen failed, err:%v\n", err)
		return
	}
	buf := make([]byte, 1500)
	for Continue {
		n, addr, err := ue_conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("accept ue package failed, err:%v\n", err)
			return
		}
		// fmt.Printf("%v", buf[:n])
		if n < 5 {
			ueid, err := strconv.Atoi(string(buf[:n]))
			if err == nil {
				ue_list[ueid].addr = addr
			}
		} else if PLR == 1 {
			upf_conn.Write(buf[:n])
		} else {
			continue
		}
	}
}

func listen_ue_process(conn net.Conn) {
	reader := bufio.NewReaderSize(conn, 16384)
	raw_length, _ := reader.Peek(2)
	length := uint16(raw_length[0])<<8 + uint16(raw_length[1])
	buf := make([]byte, length+2)
	reader.Read(buf)
	recv := string(buf[2:])
	ueid, _ := strconv.Atoi(recv)
	// if ue_list[ueid].conn != nil {
	// 	println("close")
	// 	ue_list[ueid].conn.Close()
	// }
	ue_list[ueid].conn = conn

	buf = make([]byte, 16384)
	for Continue {
		n, err := reader.Read(buf)
		// raw_length, err := reader.Peek(2)
		if err != nil {
			fmt.Println("ueid:", ueid, "err:", err)
			if err == io.EOF {
				conn.Close()
				return
			}
			select {
			case <-ch_quit:
				conn.Close()
				return
			default:
				fmt.Printf("Read from ue_conn failed, err:%v\n", err)
				return
			}
		}
		// length := uint16(raw_length[0])<<8 + uint16(raw_length[1])
		// buf := make([]byte, length+2)
		// reader.Read(buf)

		upf_conn.Write(buf[:n])
		fmt.Printf("ue%v package: %v\n", ueid, buf[:n])
		// upf_conn.Write(bytes.Clone(buf[2:]))
		// fmt.Printf("ue%v package: %v\n", ueid, buf[2:])
	}
}

func analyse_rsrp(recv string, ueid int) {
	//fmt.Printf("Read RSRP\n")
	l := strings.Split(recv, ",")
	var RSRP [][]float64
	season := RSRP_Interval / 200
	for i := 0; i < ran_N; i++ {
		var rsrp []float64
		for j := 0; j < season; j++ {
			num, err := strconv.ParseFloat(l[i*season+j+i+1], 64)
			rsrp = append(rsrp, num)
			if err != nil {
				fmt.Printf("Convert error:%v\n", err)
				return
			}
		}
		RSRP = append(RSRP, rsrp)
		//fmt.Printf("%v\n", rsrp)
	}
	ran_id_int, _ := strconv.Atoi(ran_id)
	if RSRP[ran_id_int][season-1] > -85 { //按照取值设定丢包率
		Packet_Loss_Rate = 0.01
	} else if RSRP[ran_id_int][season-1] > -100 {
		Packet_Loss_Rate = 0.01 + 0.006*(-85-RSRP[ran_id_int][season-1]) //0.006是斜率，(0.1-0.01)/(100-85),0.01是截距；
	} else {
		Packet_Loss_Rate = 0.1 + ((math.Exp(-100-RSRP[ran_id_int][season-1])-1)*0.9)/(math.Exp(56)-1) // which 56 = -100-(-156)<max> ??
	}
	// index := 1
	// for index < ran_N { //遍历RAN_N,决策迁移到哪个RAN；  CHO下实现A4 A3迁移，先进行A4的CHO判断、再进行A3的HO判断；
	// 	for i := 0; i < season; i++ {
	// 		println(RSRP[(ran_id_int+index)%ran_N][i], CHO_threshold, During_CHO, (ran_id_int+index)%ran_N, CHO_Target)
	// 		//RSRP[(ran_id_int+index)%ran_N][i]-RSRP[ran_id_int][i] >= CHO_threshold
	// 		//RSRP[(ran_id_int+index)%ran_N][i] >= CHO_threshold
	// 		if (RSRP[(ran_id_int+index)%ran_N][i] >= CHO_threshold && !During_CHO) && (ran_id_int+index)%ran_N != CHO_Target {
	// 			// 触发Handover并break； 要更改nran_id的
	// 			CHO_Target = (ran_id_int + index) % ran_N
	// 			if index == 1 && RSRP[(ran_id_int+index+1)%ran_N][i] > RSRP[(ran_id_int+index)%ran_N][i] {
	// 				CHO_Target = (ran_id_int + index + 1) % ran_N
	// 			}
	// 			println(CHO_Target)
	// 			go excuteXnHandover_sRAN(ueid, CHO_Target, index)

	// 			During_CHO = true
	// 			break
	// 		}
	// 	}

	// 	index = index + 1
	// }
	//fmt.Printf("%v\n", RSRP)
	//fmt.Printf("Recieve!\n")
}
