package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-ping/ping"
	"golang.org/x/crypto/ssh"
)

const (
	initial   int = iota // ue在5gc注册之前
	connected            // ue在5gc注册之后或者handover之后，此时仅向当前ran发包
	duringXn             // 向两个ran同时发包
)

type UE struct {
	state, pkg_index, ran_now, ran_to int
	// gtp_header                        []byte
	// ip                                string
	// nic_fd                            int
	// client_conn                       net.Conn
	allow_send bool
}

func exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

func IsNum(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func MAX_index(arr []float64) (int, float64) {
	max_index := 0 // 假设第一个元素为最大值

	// 遍历数组，寻找最大值
	for index, value := range arr {
		if value > arr[max_index] {
			max_index = index
		}
	}

	return max_index, arr[max_index]
}

func release() {
	close(ch_quit)
	// 需要对list内部元素进行更改时，range方式直接获取元素有风险
	for i := 0; i < len(conn_list); i++ {
		conn_list[i].Close()
	}
	if open_uevm {
		uevm_conn.Close()
	}
	// if auto_run {
	// 	time.Sleep(time.Second)
	// 	cmd := exec.Command("nohup", "/root/free5gc/force_kill.sh", "&")
	// 	cmd.Run()
	// }
	// for _, ue := range ue_list {
	// 	if ue.nic_fd != 0 {
	// 		syscall.Close(ue.nic_fd)
	// 	}
	// }
}

// func send_message(conn net.Conn, command string, packetlossrate float64) {
func send_message(conn net.Conn, command string) {
	// if (rand.Float64() >= packetlossrate && PLR_switch == 1) || PLR_switch == 0 {
	// 	t := time.Now().Format("2006-01-02 15:04:05.000000")
	// 	command = command + ",controller send to ran," + strings.Split(t, " ")[1]
	// 	_, err := conn.Write(add_tcp_length(command))
	// 	if err != nil {
	// 		fmt.Printf("send failed, err:%v\n", err)
	// 		return
	// 	}
	// 	// t := time.Now().Format("2006-01-02 15:04:05.000000")
	// 	// content := strings.Split(t, " ")[1]+",send to ran,"+command
	// 	// fmt.Printf("send content：%v\n", content)
	// 	ch_log <- command + "\n"
	// }
	t := time.Now().Format("2006-01-02 15:04:05.000000")
	command = command + ",controller send to ran," + strings.Split(t, " ")[1]
	_, err := conn.Write(add_tcp_length(command))
	if err != nil {
		fmt.Printf("send failed, err:%v\n", err)
		return
	}
	// t := time.Now().Format("2006-01-02 15:04:05.000000")
	// content := strings.Split(t, " ")[1]+",send to ran,"+command
	// fmt.Printf("send content：%v\n", content)
	ch_log <- command + "\n"
}

func write_log(ch_log chan string, ch_quit chan struct{}) {
	var log string
	file, _ := os.OpenFile(Logname, os.O_APPEND|os.O_WRONLY, os.ModeAppend)

	for {
		select {
		case <-ch_quit:
			file.Close()
			return
		case log = <-ch_log:
			file.WriteString(log)
		}
	}
}

func autorun(ran_ip string, name string) {
	sshHost := ran_ip
	sshUser := "root"
	sshPassword := "zz"
	sshType := "password"
	sshPort := 22

	//创建sshp登陆配置
	config := &ssh.ClientConfig{
		Timeout:         time.Second, //ssh 连接time out 时间一秒钟, 如果ssh验证错误 会在一秒内返回
		User:            sshUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //这个可以, 但是不够安全
		//HostKeyCallback: hostKeyCallBackFunc(h.Host),
	}
	if sshType == "password" {
		config.Auth = []ssh.AuthMethod{ssh.Password(sshPassword)}
	}

	//dial 获取ssh client
	addr := fmt.Sprintf("%s:%d", sshHost, sshPort)
	sshClient, _ := ssh.Dial("tcp", addr, config)
	defer sshClient.Close()

	//创建ssh-session
	session, _ := sshClient.NewSession()
	defer session.Close()
	//执行远程命令
	output, _ := session.CombinedOutput("cd /root/" + name + " && rm output && nohup ./" + name + " > output 2>&1 &")
	println(output)
}

func check_access(url string, ch_check chan struct{}, status *struct {
	count    int
	fail_url string
}) {
	pinger, _ := ping.NewPinger(url)
	pinger.Count = 1
	pinger.Run() // Blocks until finished.
	// get send/receive/duplicate/rtt stats
	if pinger.Statistics().PacketsRecv != pinger.Count {
		status.fail_url = url
	}
	status.count += 1
	ch_check <- struct{}{}
}

// 解决粘包问题
func add_tcp_length(str string) []byte {
	length := uint16(len(str))
	header := []byte{uint8(length >> 8), uint8((length << 8) >> 8)}
	return append(header, []byte(str)...)
}

// ip头部校验和只计算20字节
func modifyCheckSum(data []byte) (uint8, uint8) {
	var (
		sum    uint32
		length int = len(data)
		index  int
	)
	// 先给校验和位置0
	data[10] = 0
	data[11] = 0
	// 以每16位为单位进行求和，直到所有的字节全部求完或者只剩下一个8位字节（如果剩余一个8位字节说明字节数为奇数个）
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		index += 2
		length -= 2
	}
	// 如果字节数为奇数个，要加上最后剩下的那个8位字节
	if length > 0 {
		sum += uint32(data[index])
	}
	// 加上高16位进位的部分
	sum += (sum >> 16)
	// 别忘了返回的时候先求反
	rsum := uint16(^sum)
	return uint8(rsum >> 8), uint8(rsum)
}
