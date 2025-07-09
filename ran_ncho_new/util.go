package main

import (
	"encoding/binary"
	"encoding/hex"
	"test"

	// "bytes"
	// "encoding/base64"
	// "encoding/binary"

	"net"
	"os"

	// "os/exec"
	"strconv"
	// "testing"
	"time"

	"github.com/free5gc/util/ueauth"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	// "test/ngapTestpacket"
	// "golang.org/x/net/html/charset"
	// "github.com/axgle/mahonia"
	// "github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require"
	// ausf_context "github.com/free5gc/ausf/context"
	// "github.com/free5gc/util/milenage"
)

const (
	initial                  int = iota
	w_authentication_request     // w代表wait，也就是此时等待收到authentication_request这条消息
	w_security_mode_command
	w_initial_context_setup_request
	w_PDU_session_resource_setup_request
	connected
	duringXn_P
	duringXn_E
	duringN2_P
	duringN2_E
	// w_handover_request						// 这里开始进入N2准备阶段
	// w_handover_command
	// w_ue_context_release_command			// 这里开始进入N2执行阶段
	// w_initial_context_setup_request_afterN2
)

type UE struct {
	state int
	IP    string
	TEID  string
	NH    []uint8
	ch    chan []byte
	ue    *test.RanUeContext
	conn  net.Conn
	addr  *net.UDPAddr
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

func message_timer(ueid int, command string, old_record string) {
	time.Sleep(time.Duration(timer) * time.Millisecond)
	if record, ok := Recorder[ueid].Load(command); !ok || record == old_record {
		println("ueid: ", ueid, ", seq: ", command, ",  TIME OUT !!!")
		// } else {
		// 	println("ueid: ", ueid, ", command: ", command, ",  received from upf.")
	}
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

func release() {
	Continue = false
	select {
	case _, recv := <-ch_quit:
		if recv {
			close(ch_quit)
		}
	default:
		close(ch_quit)
	}
	for _, ue := range ue_list {
		if ue.conn != nil {
			ue.conn.Close()
		}
	}
	amf_conn.Close()
	upf_conn.Close()
	for i := 0; i < ran_number-1; i++ {
		pran_conn[i].Close()
		nran_conn[i].Close()
	}

	controller_conn.Close()
	delete_database()
}

func delete_database() {
	if ranN2Ipv4Addr != "172.17.0.2" {
		return
	}
	for Continue {
		time.Sleep(1 * time.Second)
	}
	for index := 1; index < len(ue_list); index++ {
		test.DelAuthSubscriptionToMongoDB(ue_list[index].ue.Supi)
		test.DelAccessAndMobilitySubscriptionDataFromMongoDB(ue_list[index].ue.Supi, "20893")
		test.DelSmfSelectionSubscriptionDataFromMongoDB(ue_list[index].ue.Supi, "20893")
	}
}

func createDummyPackage(id int, seq int) []byte {
	// gtpHdr, _ := hex.DecodeString("32ff00340000000100000000")
	//println(ue_list[id].TEID)
	gtpHdr, _ := hex.DecodeString("32ff0034" + ue_list[id].TEID + "00000000")
	println("id:", id, "gtpHdr:", "32ff0034"+ue_list[id].TEID+"00000000", "ip:", ue_list[id].IP)

	ipv4hdr := ipv4.Header{
		Version:  4,
		Len:      20,
		Protocol: 1,
		Flags:    0,
		TotalLen: 48,
		TTL:      64,
		// Src:      net.ParseIP("10.60.0." + strconv.Itoa(id)).To4(),
		Src: net.ParseIP(ue_list[id].IP).To4(),
		Dst: net.ParseIP("110.242.68.66").To4(),
		// Dst: net.ParseIP("159.226.227.86").To4(),
		ID: 1,
	}
	//println(ue_list[id].IP)
	checksum := test.CalculateIpv4HeaderChecksum(&ipv4hdr)
	ipv4hdr.Checksum = int(checksum)
	v4HdrBuf, _ := ipv4hdr.Marshal()
	tt := append(gtpHdr, v4HdrBuf...)

	icmpData, _ := hex.DecodeString("8c870d0000000000101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f3031323334353637")

	m := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			// ID: 12394, Seq: 1,
			ID: id, Seq: seq,
			Data: icmpData,
		},
	}

	b, _ := m.Marshal(nil)
	// b[2] = byte(icmp_checksum >> 8)
	// b[3] = byte(icmp_checksum % 256)
	// fmt.Printf("icmp_checksum %% 256: %d\n", byte(icmp_checksum % 256))
	// fmt.Printf("icmp_checksum >> 8: %d\n", byte(icmp_checksum >> 8))

	// icmp checksum
	/*
		校验和比较奇怪，按照icmp标准计算校验和，b[2]应该是111，b[3]应该是58，这两个数字在上面b, _ := m.Marshal(nil)中就已经计算完成了，
		但是这里还特意改为b[2] = 0xaf，b[3] = 0x88。不知道是怎么计算的，可能是因为icmp校验和也需要计算后面的icmp data，但是这个例子是错误的，所以直接修改校验和了。
		有意思的是，用标准的校验和无法成功发送数据包，必须用这两个。因此我倒推了一下校验和的数字，以后对应于不同ID和Seq的icmp数据包可以根据这个来计算。

		对于b[2] = 0xaf，b[3] = 0x88，计算0xaf * 256 + 0x88 = 44936，取反码就是20599，可以推算出原本按照icmp规则得到的校验和总数是217204，
		也就是说217204 >> 16 + 217204 % 65536 = 3 + 20596 = 20599
		减去默认的ID 12394和默认的Seq 1就是8204

		！！因此之后计算校验和的步骤如下！！:
		checksum := 8204 + ID + Seq
		checksum = ^checksum
		b[2] = byte(checksum >> 8)
		b[3] = byte(checksum % 256)

		这种方式计算出的ID=1, Seq=1的情况下b[2] = 0xdf，b[3] = 0xf1。证明可以成功跑通。
		由于校验和奇怪的计算方法，只能根据可以跑的数字倒推，而不要修改icmpdata，否则从头算起无法保证结果
	*/

	// 原始校验和的奇怪数字
	// b[2] = 0xaf
	// b[3] = 0x88

	// 对应ID=1，Seq=1的校验和
	// b[2] = 0xdf
	// b[3] = 0xf1

	// 真正的校验和计算，固定id为1
	icmp_cs := (8204+id+seq)%65536 + (8204+id+seq)/65536
	icmp_cs = ^icmp_cs
	b[2] = byte(icmp_cs >> 8)
	b[3] = byte(icmp_cs % 256)

	return append(tt, b...)
}

// 解决粘包问题
func add_tcp_length(str string) []byte {
	length := uint16(len(str))
	header := []byte{uint8(length >> 8), uint8((length << 8) >> 8)}
	return append(header, []byte(str)...)
}

func DerivateNH(Kamf []uint8, syncInput []byte) []uint8 {
	P0 := syncInput
	L0 := ueauth.KDFLen(P0)

	NH, err := ueauth.GetKDFValue(Kamf, ueauth.FC_FOR_NH_DERIVATION, P0, L0)
	if err != nil {
		println("err: ", err)
		return make([]uint8, 1)
	}
	// fmt.Printf("NH: %v\n", NH)
	return NH
}

func DerivateAnKey(Kamf []uint8, ULCount uint32) []byte {
	accessType := uint8(0x01) // security.AccessType3GPP // Defalut 3gpp
	P0 := make([]byte, 4)
	binary.BigEndian.PutUint32(P0, ULCount)
	L0 := ueauth.KDFLen(P0)
	P1 := []byte{accessType}
	L1 := ueauth.KDFLen(P1)

	Kgnb, err := ueauth.GetKDFValue(Kamf, ueauth.FC_FOR_KGNB_KN3IWF_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		println("err: ", err)
		return make([]byte, 1)
	}
	return Kgnb
}

// ip头部校验和只计算20字节
func modifyIPv4CheckSum(data []byte) (uint8, uint8) {
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
