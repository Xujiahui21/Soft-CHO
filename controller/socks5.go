package main

// =======================================================
// Need maintain IP header and TCP header by ourselves
// Not implement yet
// =======================================================

// import (
// 	"bufio"
// 	"encoding/binary"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"net"
// 	"strconv"

// 	"github.com/google/gopacket/layers"
// )

// func socks5_start(ueid int) {
// 	src_port := strconv.Itoa(10000 + ueid)
// 	server, err := net.Listen("tcp", ":"+src_port)
// 	if err != nil {
// 		fmt.Printf("Listen failed: %v\n", err)
// 		return
// 	}

// 	for {
// 		client, err := server.Accept()
// 		println("connect accept")
// 		if err != nil {
// 			fmt.Printf("Accept failed: %v", err)
// 			continue
// 		}
// 		ue_list[ueid].client_conn = client
// 		go process(ueid, client, src_port)
// 	}
// }

// func process(ueid int, client net.Conn, src_port string) {
// 	defer client.Close()

// 	if err := Socks5Auth(client); err != nil {
// 		fmt.Println("auth error:", err)
// 		client.Close()
// 		return
// 	}

// 	err := Socks5Connect(ueid, client, src_port)
// 	if err != nil {
// 		fmt.Println("connect error:", err)
// 		client.Close()
// 		return
// 	}

// 	// listen
// 	reader := bufio.NewReader(client)
// 	buf := make([]byte, 1500)
// 	ip := net.ParseIP(ue_list[ueid].ip).To4()
// 	for {
// 		// only get tcp data. no tcp header and ip header
// 		n, err := reader.Read(buf)
// 		if err != nil {
// 			return
// 		}
// 		go socks5_send_package(buf[:n], ueid, ip)
// 	}
// }

// func Socks5Auth(client net.Conn) (err error) {
// 	buf := make([]byte, 256)

// 	// 读取 VER 和 NMETHODS
// 	n, err := io.ReadFull(client, buf[:2])
// 	if n != 2 {
// 		return errors.New("reading header: " + err.Error())
// 	}

// 	ver, nMethods := int(buf[0]), int(buf[1])
// 	if ver != 5 {
// 		return errors.New("invalid version")
// 	}

// 	// 读取 METHODS 列表
// 	n, err = io.ReadFull(client, buf[:nMethods])
// 	if n != nMethods {
// 		return errors.New("reading methods: " + err.Error())
// 	}

// 	//无需认证
// 	n, err = client.Write([]byte{0x05, 0x00})
// 	if n != 2 || err != nil {
// 		return errors.New("write rsp: " + err.Error())
// 	}

// 	return nil
// }

// func Socks5Connect(ueid int, client net.Conn, src_port string) error {
// 	buf := make([]byte, 256)

// 	n, err := io.ReadFull(client, buf[:4])
// 	if n != 4 {
// 		return errors.New("read header: " + err.Error())
// 	}

// 	ver, cmd, _, atyp := buf[0], buf[1], buf[2], buf[3]
// 	if ver != 5 || cmd != 1 {
// 		return errors.New("invalid ver/cmd")
// 	}

// 	addr := ""
// 	switch atyp {
// 	case 1:
// 		n, err = io.ReadFull(client, buf[:4])
// 		if n != 4 {
// 			return errors.New("invalid IPv4: " + err.Error())
// 		}
// 		addr = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])

// 	case 3:
// 		n, err = io.ReadFull(client, buf[:1])
// 		if n != 1 {
// 			return errors.New("invalid hostname: " + err.Error())
// 		}
// 		addrLen := int(buf[0])

// 		n, err = io.ReadFull(client, buf[:addrLen])
// 		if n != addrLen {
// 			return errors.New("invalid hostname: " + err.Error())
// 		}
// 		addr = string(buf[:addrLen])

// 	case 4:
// 		return errors.New("IPv6: no supported yet")

// 	default:
// 		return errors.New("invalid atyp")
// 	}

// 	n, err = io.ReadFull(client, buf[:2])
// 	if n != 2 {
// 		return errors.New("read port: " + err.Error())
// 	}
// 	des_port := binary.BigEndian.Uint16(buf[:2])

// 	// destAddrPort := fmt.Sprintf("%s:%d", addr, port)
// 	// dest, err := net.Dial("tcp", destAddrPort)
// 	// if err != nil {
// 	// 	return errors.New("dial dst: " + err.Error())
// 	// }

// 	ip_layer := layers.IPv4{
// 		SrcIP:    net.ParseIP(ue_list[ueid].ip).To4(),
// 		DstIP:    net.ParseIP(addr).To4(),
// 		Version:  4,
// 		TTL:      64,
// 		Protocol: layers.IPProtocolTCP,
// 	}
// 	tcp_layer := layers.TCP{}

// 	_, err = client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
// 	if err != nil {
// 		return errors.New("write rsp: " + err.Error())
// 	}

// 	return nil
// }

// func socks5_send_package(data []byte, ueid int, ip []byte) {
// 	ip_layer := layers.IPv4{
// 		SrcIP:    net.ParseIP(ue_list[ueid].ip).To4(),
// 		DstIP:    net.ParseIP(addr).To4(),
// 		Version:  4,
// 		TTL:      64,
// 		Protocol: layers.IPProtocolTCP,
// 	}
// 	tcp_layer := layers.TCP{}

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
