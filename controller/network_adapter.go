package main

// ===========================================================
// Implement at VM ue side. Controller just for controll
// ===========================================================

// import (
// 	"bytes"
// 	"encoding/hex"
// 	"fmt"
// 	"net"
// 	"os"
// 	"os/exec"
// 	"strconv"
// 	"syscall"
// 	"unsafe"

// 	"golang.org/x/sys/unix"
// )

// const (
// 	MTU      = 1500
// 	IFPREFIX = "/dev/net/tun"
// )

// func start_nic(ueid int, ip string, teid string) {
// 	ue_list[ueid].ip = ip
// 	ue_list[ueid].gtp_header, _ = hex.DecodeString("32ff0034" + teid + "00000000")
// 	if ue_list[ueid].nic_fd != 0 {
// 		syscall.Close(ue_list[ueid].nic_fd)
// 	}
// 	ue_list[ueid].nic_fd = create_nic("ue"+strconv.Itoa(ueid), ip)
// 	read_nic(ueid)
// }

// // return fd
// func create_nic(name, ip string) int {
// 	// 打开tun设备
// 	fd, err := syscall.Open(IFPREFIX, os.O_RDWR, 0)
// 	if err != nil {
// 		fmt.Println("Open tun device failed:", err)
// 		return 0
// 	}

// 	// 设置tun设备
// 	// ifr := &syscall.Ifreq{}
// 	// copy(ifr.Name[:], IFNAME)
// 	ifr, _ := unix.NewIfreq(name)
// 	// ifr.Flags = syscall.IFF_TUN | syscall.IFF_NO_PI
// 	ifr.SetUint16(syscall.IFF_TUN | syscall.IFF_NO_PI)
// 	_, _, err1 := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TUNSETIFF, uintptr(unsafe.Pointer(ifr)))
// 	if err1 != 0 {
// 		fmt.Println("Set tun device failed:", err)
// 		return 0
// 	}

// 	cmd := exec.Command("ip", "link", "set", "dev", name, "up")
// 	if err := cmd.Run(); err != nil {
// 		fmt.Fprintf(os.Stderr, "Error starting TUN device %v: %v", name, err)
// 		return 0
// 	}
// 	// cmd = exec.Command("ip", "addr", "add", ip+"/24", "dev", name)
// 	// if err := cmd.Run(); err != nil {
// 	// 	fmt.Fprintf(os.Stderr, "Error configuring TUN device %v: %v", name, err)
// 	// 	return 0
// 	// }
// 	// cmd = exec.Command("route", "del", "-net", "10.60.0.0", "netmask", "255.255.255.0", "dev", name)
// 	// if err := cmd.Run(); err != nil {
// 	// 	fmt.Fprintf(os.Stderr, "Error configuring route %v: %v", name, err)
// 	// 	return 0
// 	// }
// 	return fd
// }

// func read_nic(ueid int) {
// 	fd := ue_list[ueid].nic_fd
// 	ip := net.ParseIP(ue_list[ueid].ip).To4()
// 	buf := make([]byte, MTU)
// 	for {
// 		n, err := syscall.Read(fd, buf)
// 		if err != nil {
// 			fmt.Println("Read from tun device failed:", err)
// 			return
// 		}
// 		// 创建了nic就会一直收到这个类似心跳的信息，不知道是什么
// 		if !bytes.Equal(buf[:16], []byte{96, 0, 0, 0, 0, 8, 58, 255, 254, 128, 0, 0, 0, 0, 0, 0}) {
// 			go nic_send_package(buf[:n], ueid, ip)
// 		}
// 	}
// }

// func nic_send_package(data []byte, ueid int, ip []byte) {
// 	for i := 0; i < 4; i++ {
// 		data[12+i] = ip[i]
// 	}
// 	data[10], data[11] = modifyIPv4CheckSum(data[:20])
// 	fmt.Println("Read:", data)
// 	full_package := append(ue_list[ueid].gtp_header, data...)
// 	command := strconv.Itoa(ueid) + ",APP Send," + string(full_package)
// 	_, err := conn_list[ue_list[ueid].ran_now].Write(add_tcp_length(command))
// 	if err != nil {
// 		fmt.Printf("send failed, err:%v\n", err)
// 		return
// 	}
// 	ch_log <- command + "\n"
// }

// func write_nic(fd int, data []byte) {
// 	_, err := syscall.Write(fd, data)
// 	if err != nil {
// 		fmt.Println("Write to tun device failed:", err)
// 		return
// 	}
// }
