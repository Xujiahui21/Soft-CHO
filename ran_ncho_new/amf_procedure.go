package main

import (
	"encoding/hex"
	"fmt"
	"strings"
	"test"
	"test/nasTestpacket"

	"bytes"
	// "encoding/base64"
	// "encoding/binary"

	// "os/exec"

	"strconv"

	// "testing"
	"time"

	// "test/ngapTestpacket"
	// "golang.org/x/net/html/charset"
	// "github.com/axgle/mahonia"

	// "github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require"

	// ausf_context "github.com/free5gc/ausf/context"

	"github.com/free5gc/nas"
	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/nas/nasType"

	// "github.com/free5gc/util/milenage"

	"github.com/free5gc/ngap"
	"github.com/free5gc/ngap/ngapType"
	"github.com/free5gc/openapi/models"
)

func registration(id int) {
	// 这里的id就是amf_ue_id，和所有ue_list的index、ue.id一致
	// var n int
	var sendMsg []byte
	var recvMsg = make([]byte, 2048)
	var InitialContextSetupResponse []byte
	var UplinkNASTransport1 []byte
	var UplinkNASTransport2 []byte

	ue := ue_list[id].ue
	// 生成amf_ueid_map记录
	// amf_ueid_map.Store(int(ue.AmfUeNgapId), id)
	ran_ueid_now += 1
	// insert UE data to MongoDB

	servingPlmnId := "20893"
	test.InsertAuthSubscriptionToMongoDB(ue.Supi, ue.AuthenticationSubs)
	test.GetAuthSubscriptionFromMongoDB(ue.Supi)
	{
		amData := test.GetAccessAndMobilitySubscriptionData()
		test.InsertAccessAndMobilitySubscriptionDataToMongoDB(ue.Supi, amData, servingPlmnId)
		test.GetAccessAndMobilitySubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
	}
	{
		smfSelData := test.GetSmfSelectionSubscriptionData()
		test.InsertSmfSelectionSubscriptionDataToMongoDB(ue.Supi, smfSelData, servingPlmnId)
		test.GetSmfSelectionSubscriptionDataFromMongoDB(ue.Supi, servingPlmnId)
	}
	{
		smSelData := test.GetSessionManagementSubscriptionData()
		test.InsertSessionManagementSubscriptionDataToMongoDB(ue.Supi, servingPlmnId, smSelData)
		test.GetSessionManagementDataFromMongoDB(ue.Supi, servingPlmnId)
	}
	{
		amPolicyData := test.GetAmPolicyData()
		test.InsertAmPolicyDataToMongoDB(ue.Supi, amPolicyData)
		test.GetAmPolicyDataFromMongoDB(ue.Supi)
	}
	{
		smPolicyData := test.GetSmPolicyData()
		test.InsertSmPolicyDataToMongoDB(ue.Supi, smPolicyData)
		test.GetSmPolicyDataFromMongoDB(ue.Supi)
	}
	i := id + id/15
	// %256的动作可以省去了
	l1 := uint8(i)
	l1 = l1>>4 | (l1&0x0f)<<4
	i /= 256
	// %256的动作可以省去了
	l2 := uint8(i)
	l2 = l2>>4 | (l2&0x0f)<<4
	// send InitialUeMessage(Registration Request)(imsi-2089300007487)
	mobileIdentity5GS := nasType.MobileIdentity5GS{
		Len:    12, // suci
		Buffer: []uint8{0x01, 0x02, 0xf8, 0x39, 0xf0, 0xff, 0x00, 0x00, 0x00, 0x00, l2, l1},
	}

	ueSecurityCapability := ue.GetUESecurityCapability()
	registrationRequest := nasTestpacket.GetRegistrationRequest(
		nasMessage.RegistrationType5GSInitialRegistration, mobileIdentity5GS, nil, ueSecurityCapability, nil, nil, nil)
	sendMsg, _ = test.GetInitialUEMessage(ue.RanUeNgapId, registrationRequest, "")
	amf_conn.Write(sendMsg)
	ue_list[id].state = w_authentication_request

	t := time.Now().Format("2006-01-02 15:04:05.000000")
	content := strconv.Itoa(id) + ",Registration Request,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
	ControlRecorder[id] = append(ControlRecorder[id], content)
	ch_log <- content + "\n"

	// 为了配合amf的丢包重传机制，加入循环
	for Continue {
		// fmt.Printf("%v\n", recvMsg)
		recvMsg = <-ue_list[id].ch
		switch int64(recvMsg[1]) {
		case ngapType.ProcedureCodeDownlinkNASTransport:
			if len(recvMsg) >= 54 {
				// 如果是重发的，就直接回复。再走一次流程可能会导致ue状态的改变
				if ue_list[id].state == w_security_mode_command {
					amf_conn.Write(sendMsg)
					t = time.Now().Format("2006-01-02 15:04:05.000000")
					content = strconv.Itoa(id) + ",Authentication Response,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
					ControlRecorder[id] = append(ControlRecorder[id], content)
					ch_log <- content + "\n"
					continue
				}
				// receive NAS Authentication Request Msg
				// 发给controller让其他ran更新ue DeriveRESstarAndSetKey信息
				if id > ran_number {
					send_Authentication_Request := strconv.Itoa(id) + ",Authentication Request," + string(recvMsg)
					controller_conn.Write(add_tcp_length(send_Authentication_Request))
				}

				ngapPdu, err := ngap.Decoder(recvMsg)
				if err != nil {
					println("\n=============================================================")
					println("ngapPdu decode fail, ueid:", id, "amf_ueid:", ue.AmfUeNgapId, "ran_ueid:", ue.RanUeNgapId)
					fmt.Printf("\nAuthentication Request: %v\n", recvMsg)
					println("=============================================================\n")
				}
				// ngapPdu, _ := ngap.Decoder(recvMsg[:n])
				// assert.True(t, ngapPdu.Present == ngapType.NGAPPDUPresentInitiatingMessage, "No NGAP Initiating Message received.")

				// Calculate for RES*
				nasPdu := test.GetNasPdu(ue, ngapPdu.InitiatingMessage.Value.DownlinkNASTransport)
				// require.Equal(t, nasPdu.GmmHeader.GetMessageType(), nas.MsgTypeAuthenticationRequest,
				// 	"Received wrong GMM message. Expected Authentication Request.")
				rand := nasPdu.AuthenticationRequest.GetRANDValue()
				resStat := ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")

				// 更新NH
				Kgnb := DerivateAnKey(ue.Kamf, ue.ULCount.Get())
				// fmt.Printf("Kgnb: %v\n", Kgnb)
				ue_list[id].NH = DerivateNH(ue.Kamf, Kgnb)

				// send NAS Authentication Response
				pdu := nasTestpacket.GetAuthenticationResponse(resStat, "")
				sendMsg, _ = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
				amf_conn.Write(sendMsg)
				ue_list[id].state = w_security_mode_command

				t = time.Now().Format("2006-01-02 15:04:05.000000")
				content = strconv.Itoa(id) + ",Authentication Response,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
				ControlRecorder[id] = append(ControlRecorder[id], content)
				ch_log <- content + "\n"
			} else {
				if ue_list[id].state == w_initial_context_setup_request {
					amf_conn.Write(sendMsg)
					t = time.Now().Format("2006-01-02 15:04:05.000000")
					content = strconv.Itoa(id) + ",Security Mode Complete,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
					ControlRecorder[id] = append(ControlRecorder[id], content)
					ch_log <- content + "\n"
					continue
				}
				// receive NAS Security Mode Command Msg
				// n, _ = upf_conn.Read(recvMsg)
				// ngapPdu, _ = ngap.Decoder(recvMsg)
				// ngapPdu, _ = ngap.Decoder(recvMsg[:n])
				// nasPdu = test.GetNasPdu(ue, ngapPdu.InitiatingMessage.Value.DownlinkNASTransport)
				// require.Equal(t, nasPdu.GmmHeader.GetMessageType(), nas.MsgTypeSecurityModeCommand,
				// 	"Received wrong GMM message. Expected Security Mode Command.")

				// send NAS Security Mode Complete Msg
				registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
					mobileIdentity5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, nil)
				pdu := nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)
				pdu, _ = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, true, true)
				sendMsg, _ = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
				amf_conn.Write(sendMsg)
				ue_list[id].state = w_initial_context_setup_request

				t = time.Now().Format("2006-01-02 15:04:05.000000")
				content = strconv.Itoa(id) + ",Security Mode Complete,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
				ControlRecorder[id] = append(ControlRecorder[id], content)
				ch_log <- content + "\n"
			}
		case ngapType.ProcedureCodeInitialContextSetup:
			if ue_list[id].state == w_PDU_session_resource_setup_request {
				amf_conn.Write(InitialContextSetupResponse)
				t = time.Now().Format("2006-01-02 15:04:05.000000")
				content = strconv.Itoa(id) + ",Initial Context Setup Response,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
				ControlRecorder[id] = append(ControlRecorder[id], content)
				ch_log <- content + "\n"

				amf_conn.Write(UplinkNASTransport1)
				t = time.Now().Format("2006-01-02 15:04:05.000000")
				content = strconv.Itoa(id) + ",Registration Complete,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
				ControlRecorder[id] = append(ControlRecorder[id], content)
				ch_log <- content + "\n"

				amf_conn.Write(UplinkNASTransport2)
				t = time.Now().Format("2006-01-02 15:04:05.000000")
				content = strconv.Itoa(id) + ",Get Pdu Session Establishment Request,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
				ControlRecorder[id] = append(ControlRecorder[id], content)
				ch_log <- content + "\n"
				continue
			}
			// receive ngap Initial Context Setup Request Msg
			// ngapPdu, _ = ngap.Decoder(recvMsg)
			// ngapPdu, _ = ngap.Decoder(recvMsg[:n])
			// assert.True(t, ngapPdu.Present == ngapType.NGAPPDUPresentInitiatingMessage &&
			// 	ngapPdu.InitiatingMessage.ProcedureCode.Value == ngapType.ProcedureCodeInitialContextSetup,
			// 	"No InitialContextSetup received.")

			// send ngap Initial Context Setup Response Msg
			InitialContextSetupResponse, _ = test.GetInitialContextSetupResponse(ue.AmfUeNgapId, ue.RanUeNgapId)
			amf_conn.Write(InitialContextSetupResponse)

			t = time.Now().Format("2006-01-02 15:04:05.000000")
			content = strconv.Itoa(id) + ",Initial Context Setup Response,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
			ControlRecorder[id] = append(ControlRecorder[id], content)
			ch_log <- content + "\n"

			// send NAS Registration Complete Msg
			pdu := nasTestpacket.GetRegistrationComplete(nil)
			pdu, _ = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
			UplinkNASTransport1, _ = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
			amf_conn.Write(UplinkNASTransport1)

			t = time.Now().Format("2006-01-02 15:04:05.000000")
			content = strconv.Itoa(id) + ",Registration Complete,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
			ControlRecorder[id] = append(ControlRecorder[id], content)
			ch_log <- content + "\n"

			time.Sleep(100 * time.Millisecond)

			// send GetPduSessionEstablishmentRequest Msg
			sNssai := models.Snssai{
				Sst: 1,
				Sd:  "010203",
			}
			pdu = nasTestpacket.GetUlNasTransport_PduSessionEstablishmentRequest(10, nasMessage.ULNASTransportRequestTypeInitialRequest, "internet", &sNssai)
			pdu, _ = test.EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
			UplinkNASTransport2, _ = test.GetUplinkNASTransport(ue.AmfUeNgapId, ue.RanUeNgapId, pdu)
			amf_conn.Write(UplinkNASTransport2)
			ue_list[id].state = w_PDU_session_resource_setup_request

			t = time.Now().Format("2006-01-02 15:04:05.000000")
			content = strconv.Itoa(id) + ",Get Pdu Session Establishment Request,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
			ControlRecorder[id] = append(ControlRecorder[id], content)
			ch_log <- content + "\n"
		case ngapType.ProcedureCodePDUSessionResourceSetup:
			// receive 12. NGAP-PDU Session Resource Setup Request(DL nas transport((NAS msg-PDU session setup Accept)))
			// ngapPdu, _ = ngap.Decoder(recvMsg)
			// n, _ = upf_conn.Read(recvMsg)
			// ngapPdu, _ = ngap.Decoder(recvMsg[:n])
			// assert.True(t, ngapPdu.Present == ngapType.NGAPPDUPresentInitiatingMessage &&
			// 	ngapPdu.InitiatingMessage.ProcedureCode.Value == ngapType.ProcedureCodePDUSessionResourceSetup,
			// 	"No PDUSessionResourceSetup received.")

			// 获取5gc分配给ue的ip并广播，最早应该是从recvMsg[69]开始的
			fmt.Printf("PDU Session Resource Setup: %v\n", recvMsg)
			var ip string
			var teid string
			for i := 67; i < 77; i++ {
				if bytes.Equal(recvMsg[i:i+2], []byte{10, 60}) {
					ip = "10.60." + strconv.Itoa(int(recvMsg[i+2])) + "." + strconv.Itoa(int(recvMsg[i+3]))
					ip_ueid_map.Store(string(recvMsg[i:i+4]), id)
					break
				}
			}
			// 获取5gc分配给ue的teid并广播，最早应该是从recvMsg[144]开始的
			for i := 136; i < (len(recvMsg) - 8); i++ {
				if bytes.Equal(recvMsg[i:i+4], []byte{172, 17, 0, 1}) {
					teid = hex.EncodeToString(recvMsg[i+4 : i+8])
					break
				}
			}
			ue_list[id].IP = ip
			ue_list[id].TEID = teid
			send_allocate_ip := strconv.Itoa(id) + ",Allocate IP & TEID," + ip + "," + teid
			controller_conn.Write(add_tcp_length(send_allocate_ip))

			// send 14. NGAP-PDU Session Resource Setup Response
			sendMsg, _ = test.GetPDUSessionResourceSetupResponse(10, ue.AmfUeNgapId, ue.RanUeNgapId, ranN3Ipv4Addr)
			amf_conn.Write(sendMsg)
			ue_list[id].state = connected

			t = time.Now().Format("2006-01-02 15:04:05.000000")
			content = strconv.Itoa(id) + ",PDU Session Resource Setup Response,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
			ControlRecorder[id] = append(ControlRecorder[id], content)
			ch_log <- content + "\n"

			t = time.Now().Format("2006-01-02 15:04:05.000000")
			content = strconv.Itoa(id) + ",Registration Complete,ran" + ran_id + " send to ue," + strings.Split(t, " ")[1]
			controller_conn.Write(add_tcp_length(content))
			ControlRecorder[id] = append(ControlRecorder[id], content)
			ch_log <- content + "\n"
			if id > ran_number {
				content = strconv.Itoa(id) + ",RSRP_Interval,ran" + ran_id + ",100," + strings.Split(t, " ")[1]
				controller_conn.Write(add_tcp_length(content))
				ch_log <- content + "\n"
			}  // hand out the RSRP reporting interval 
			if id > ran_number {
				content = strconv.Itoa(id) + ",RSRP_Meas_Period,ran" + ran_id + ",4," + strings.Split(t, " ")[1]
				controller_conn.Write(add_tcp_length(content))
				ch_log <- content + "\n"
			}  //handout the RSRP measurement period of UE
			if id > ran_number {
				content = strconv.Itoa(id) + ",MR_Trigger_Event_Para,ran" + ran_id + ",Type,A2,Threshold,-100,TTT,80," + strings.Split(t, " ")[1]
				controller_conn.Write(add_tcp_length(content))
				ch_log <- content + "\n"
			}  //handout the trigger event of HO
			return
		default:
			return
		}
	}
}

func excuteXnHandover_sRAN(ueid int, ran_to_id int, index int) {
	for wait := 0; wait < 20; wait++ {
		if ue_list[ueid].state != connected {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
		if wait == 19 {
			println("ue id:", ueid, "not in connected state for 3s. Abort handover procedure!")
			t := time.Now().Format("2006-01-02 15:04:05.000000")
			content := strconv.Itoa(ueid) + ",Abort Handover Procedure,ran" + ran_id + " after recv HO for 3s," + strings.Split(t, " ")[1]
			ch_log <- content + "\n"
			return
		}
	}
	t := time.Now().Format("2006-01-02 15:04:05.000000")
	Ran_to_ID := strconv.Itoa(ran_to_id)
	nran_convert_id := (ran_to_id + 2 - RAN_IDINT) % ran_N
	HandoverRequest := strconv.Itoa(ueid) + ",Condition Handover Request,ran" + ran_id + " send to ran" + Ran_to_ID + "," + strings.Split(t, " ")[1]
	nran_conn[nran_convert_id].Write(add_tcp_length(HandoverRequest))
	ue_list[ueid].state = duringXn_P
	ControlRecorder[ueid] = append(ControlRecorder[ueid], HandoverRequest)
	ch_log <- HandoverRequest + "\n"

	<-ue_list[ueid].ch

	// t = time.Now().Format("2006-01-02 15:04:05.000000")
	// ran_to_id_ano := ran_to_id - 2
	// Ran_to_ID_another := strconv.Itoa(ran_to_id_ano)
	// nran_convert_id_ANO := 1
	// HandoverRequest = strconv.Itoa(ueid) + ",Condition Handover Request,ran" + ran_id + " send to ran" + Ran_to_ID_another + "," + strings.Split(t, " ")[1]
	// nran_conn[nran_convert_id_ANO].Write(add_tcp_length(HandoverRequest))
	// ue_list[ueid].state = duringXn_P
	// ControlRecorder[ueid] = append(ControlRecorder[ueid], HandoverRequest)
	// ch_log <- HandoverRequest + "\n"
	// //println(ue_list[ueid].ch)
	// // wait Condition Handover Request Acknowledge
	// <-ue_list[ueid].ch

	ULDL := []byte{uint8(ue_list[ueid].ue.ULCount.Overflow() >> 8), uint8(ue_list[ueid].ue.ULCount.Overflow()), ue_list[ueid].ue.ULCount.SQN(),
		uint8(ue_list[ueid].ue.DLCount.Overflow() >> 8), uint8(ue_list[ueid].ue.DLCount.Overflow()), ue_list[ueid].ue.DLCount.SQN()}
	t = time.Now().Format("2006-01-02 15:04:05.000000")
	// t1 := time.Now()
	RRCReconfiguration := strconv.Itoa(ueid) + ",RRC Reconfiguration (HO Command)||" + Ran_to_ID + "||" + string(ULDL) +"||A3||5||256" + ",ran" + ran_id + " send to ue," + strings.Split(t, " ")[1]
	controller_conn.Write(add_tcp_length(RRCReconfiguration))
	ControlRecorder[ueid] = append(ControlRecorder[ueid], RRCReconfiguration)
	ch_log <- RRCReconfiguration + "\n"

	//println(ue_list[ueid].ch)
	t1 := time.Now()
	// t2 := time.Now()
	// wait UE RRC Reconfigration complete

	T_target := t1.Add(time.Second * 3)
	// for !t2.Equal(T_target) {
	// 	if t2.Before(T_target) {
	// 		<-ue_list[ueid].ch
	// 	}

	// }
	recv_now := <-ue_list[ueid].ch
	t2 := time.Now()
	if !t2.Before(T_target) {
		ue_list[ueid].ch <- recv_now
		return
	}

	t = time.Now().Format("2006-01-02 15:04:05.000000")
	EarlyStatusTransfer := strconv.Itoa(ueid) + ",Early Status Transfer,ran" + ran_id + " send to ran" + Ran_to_ID + "," + strings.Split(t, " ")[1]
	nran_conn[nran_convert_id].Write(add_tcp_length(EarlyStatusTransfer))
	//ue_list[ueid].state = duringXn_E
	ControlRecorder[ueid] = append(ControlRecorder[ueid], EarlyStatusTransfer)
	ch_log <- EarlyStatusTransfer + "\n"

	// t = time.Now().Format("2006-01-02 15:04:05.000000")
	// EarlyStatusTransfer = strconv.Itoa(ueid) + ",Early Status Transfer,ran" + ran_id + " send to ran" + Ran_to_ID_another + "," + strings.Split(t, " ")[1]
	// nran_conn[nran_convert_id_ANO].Write(add_tcp_length(EarlyStatusTransfer))
	// //ue_list[ueid].state = duringXn_E
	// ControlRecorder[ueid] = append(ControlRecorder[ueid], EarlyStatusTransfer)
	// ch_log <- EarlyStatusTransfer + "\n"

	//println(ue_list[ueid].ch)
	// wait UE Context Release(HO Success)  except UE configration delete

	recv := <-ue_list[ueid].ch
	if bytes.Equal(recv, []byte("Delete")) {
		//finish delete excution & return ; else keep on ;
		t = time.Now().Format("2006-01-02 15:04:05.000000")
		ConfigurationDelete := strconv.Itoa(ueid) + ",Configuration Delete,ran" + ran_id + " send to ran" + Ran_to_ID + "," + strings.Split(t, " ")[1]
		nran_conn[nran_convert_id].Write(add_tcp_length(ConfigurationDelete))
		//ue_list[ueid].state = duringXn_E
		ControlRecorder[ueid] = append(ControlRecorder[ueid], ConfigurationDelete)
		ch_log <- ConfigurationDelete + "\n"

		//wait delete ack
		<-ue_list[ueid].ch
		t = time.Now().Format("2006-01-02 15:04:05.000000")
		// t1 := time.Now()
		DeleteACK := strconv.Itoa(ueid) + ",CHO Configuration Delete ACK,ran" + ran_id + " send to ue," + strings.Split(t, " ")[1]
		controller_conn.Write(add_tcp_length(DeleteACK))
		ControlRecorder[ueid] = append(ControlRecorder[ueid], DeleteACK)
		ch_log <- DeleteACK + "\n"
		During_CHO = false
		ue_list[ueid].state = connected

		return
	}
	During_CHO = false
	CHO_Target = 3
	t = time.Now().Format("2006-01-02 15:04:05.000000")
	SNStatusTransfer := strconv.Itoa(ueid) + ",SN Status Transfer,ran" + ran_id + " send to ran" + Ran_to_ID + "," + strings.Split(t, " ")[1]
	nran_conn[nran_convert_id].Write(add_tcp_length(SNStatusTransfer))
	ue_list[ueid].state = duringXn_E
	ControlRecorder[ueid] = append(ControlRecorder[ueid], SNStatusTransfer)
	ch_log <- SNStatusTransfer + "\n"
}

func excuteXnHandover_tRAN(ueid int, ran_index int) {
	ue_list[ueid].state = duringXn_P
	// 生成amf_ueid_map记录
	// amf_ueid_map[int(ue_list[ueid].ue.AmfUeNgapId)] = ueid
	t := time.Now().Format("2006-01-02 15:04:05.000000") //在preparation阶段执行；
	HandoverRequestAcknowledge := strconv.Itoa(ueid) + ",Condition Handover Request Acknowledge,ran" + ran_id + " send to ran" + pran_id[ran_index] + "," + strings.Split(t, " ")[1]
	pran_conn[ran_index].Write(add_tcp_length(HandoverRequestAcknowledge))
	ControlRecorder[ueid] = append(ControlRecorder[ueid], HandoverRequestAcknowledge)
	ch_log <- HandoverRequestAcknowledge + "\n"

	// 分别等待来自pran的EarlyStatusTransfer和 ue发送的RACH CHO请求开启excution；
	var ULDL []byte
	//println(ue_list[ueid].ch)
	recv := <-ue_list[ueid].ch
	if len(recv) == 6 {
		ULDL = recv
	}
	//println(ue_list[ueid].ch)
	recv = <-ue_list[ueid].ch
	if len(recv) == 6 {
		ULDL = recv
	} //是否核查ue以及pran身份是否合规，在进行下一步？？

	if STOP {
		STOP = false
		return
	}

	if bytes.Equal(recv, []byte("Deleted")) {
		t = time.Now().Format("2006-01-02 15:04:05.000000")
		CHO_Deleted := strconv.Itoa(ueid) + ",CHO Configuration Deleted,ran" + ran_id + " send to ran" + pran_id[ran_index] + "," + strings.Split(t, " ")[1]
		pran_conn[ran_index].Write(add_tcp_length(CHO_Deleted))
		ControlRecorder[ueid] = append(ControlRecorder[ueid], CHO_Deleted)
		ch_log <- CHO_Deleted + "\n"
		ue_list[ueid].state = initial
		return
	}

	ue_list[ueid].ue.ULCount.Set(uint16(ULDL[0])<<8+uint16(ULDL[1]), ULDL[2])
	ue_list[ueid].ue.DLCount.Set(uint16(ULDL[3])<<8+uint16(ULDL[4]), ULDL[5])

	// send Path Switch Request (XnHandover)
	// ** senario * trans_delay /ran to free5gc/
	time.Sleep(time.Duration(ran_5g_dura) * time.Millisecond)

	ue_list[ueid].state = duringXn_E
	sendMsg, _ := test.GetPathSwitchRequest(ue_list[ueid].ue.AmfUeNgapId, ue_list[ueid].ue.RanUeNgapId)
	amf_conn.Write(sendMsg)

	t = time.Now().Format("2006-01-02 15:04:05.000000")
	PathSwitchRequest := strconv.Itoa(ueid) + ",Path Switch Request,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
	ControlRecorder[ueid] = append(ControlRecorder[ueid], PathSwitchRequest)
	ch_log <- PathSwitchRequest + "\n"

	// wait Path Switch Request Acknowledge
	// println("wait Path Switch Request Acknowledge")
	//println(ue_list[ueid].ch)
	<-ue_list[ueid].ch
	// ** senario * trans_delay /ran to free5gc/
	time.Sleep(time.Duration(ran_5g_dura) * time.Millisecond)

	t = time.Now().Format("2006-01-02 15:04:05.000000")
	UEContextRelease := strconv.Itoa(ueid) + ",UE Context Release(HO Success),ran" + ran_id + " send to ran" + pran_id[ran_index] + "," + strings.Split(t, " ")[1]
	pran_conn[ran_index].Write(add_tcp_length(UEContextRelease))
	ue_list[ueid].state = connected
	ControlRecorder[ueid] = append(ControlRecorder[ueid], UEContextRelease)
	ch_log <- UEContextRelease + "\n"

	// 加入移动性注册测试
	// UE send NAS Registration Request(Mobility Registration Update) To Target AMF (2 AMF scenario not supportted yet)
	// ** senario * trans_delay /ran to free5gc/
	time.Sleep(time.Duration(ran_5g_dura) * time.Millisecond)

	id := ueid + ueid/15
	l1 := uint8(id)
	l1 = l1>>4 | (l1&0x0f)<<4
	id /= 256
	l2 := uint8(id)
	l2 = l2>>4 | (l2&0x0f)<<4
	mobileIdentity5GS := nasType.MobileIdentity5GS{
		Len:    11, // 5g-guti
		Buffer: []uint8{0xf2, 0x02, 0xf8, 0x39, 0xca, 0xfe, 0x00, 0x00, 0x00, l2, l1},
	}
	uplinkDataStatus := nasType.NewUplinkDataStatus(nasMessage.RegistrationRequestUplinkDataStatusType)
	uplinkDataStatus.SetLen(2)
	uplinkDataStatus.SetPSI10(1)
	ueSecurityCapability := ue_list[ueid].ue.GetUESecurityCapability()
	pdu := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSMobilityRegistrationUpdating,
		mobileIdentity5GS, nil, ueSecurityCapability, ue_list[ueid].ue.Get5GMMCapability(), nil, uplinkDataStatus)
	pdu, _ = test.EncodeNasPduWithSecurity(ue_list[ueid].ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	sendMsg, _ = test.GetInitialUEMessage(ue_list[ueid].ue.RanUeNgapId, pdu, "")
	amf_conn.Write(sendMsg)

	t = time.Now().Format("2006-01-02 15:04:05.000000")
	RegistrationRequest := strconv.Itoa(ueid) + ",Registration Request(Mobility Registration Update),ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
	ControlRecorder[ueid] = append(ControlRecorder[ueid], RegistrationRequest)
	ch_log <- RegistrationRequest + "\n"

	// 更新了ULDL之后应该是amf能认出这个UE了，所以不走Authentication Request而是和N2一样直接走到Initial Context Setup Request了
	// // Target RAN receive ngap Authentication Request Msg
	// recvMsg := <-ue_list[ueid].ch
	// // receive NAS Authentication Request Msg
	// // 发给controller让其他ran更新ue DeriveRESstarAndSetKey信息
	// // if id > ran_number {
	// // 	send_Authentication_Request := strconv.Itoa(id) + ",Authentication Request," + string(recvMsg)
	// // 	controller_conn.Write(add_tcp_length(send_Authentication_Request))
	// // }

	// ngapPdu, err := ngap.Decoder(recvMsg)
	// if err != nil {
	// 	println("\n=============================================================")
	// 	println("ngapPdu decode fail, ueid:", id, "amf_ueid:", ue_list[ueid].ue.AmfUeNgapId, "ran_ueid:", ue_list[ueid].ue.RanUeNgapId)
	// 	fmt.Printf("\nAuthentication Request: %v\n", recvMsg)
	// 	println("=============================================================\n")
	// }
	// // ngapPdu, _ := ngap.Decoder(recvMsg[:n])
	// // assert.True(t, ngapPdu.Present == ngapType.NGAPPDUPresentInitiatingMessage, "No NGAP Initiating Message received.")

	// // Calculate for RES*
	// nasPdu := test.GetNasPdu(ue_list[ueid].ue, ngapPdu.InitiatingMessage.Value.DownlinkNASTransport)
	// // require.Equal(t, nasPdu.GmmHeader.GetMessageType(), nas.MsgTypeAuthenticationRequest,
	// // 	"Received wrong GMM message. Expected Authentication Request.")
	// rand := nasPdu.AuthenticationRequest.GetRANDValue()
	// resStat := ue_list[ueid].ue.DeriveRESstarAndSetKey(ue_list[ueid].ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")

	// // send NAS Authentication Response
	// pdu = nasTestpacket.GetAuthenticationResponse(resStat, "")
	// sendMsg, _ = test.GetUplinkNASTransport(ue_list[ueid].ue.AmfUeNgapId, ue_list[ueid].ue.RanUeNgapId, pdu)
	// amf_conn.Write(sendMsg)

	// t = time.Now().Format("2006-01-02 15:04:05.000000")
	// content := strconv.Itoa(ueid) + ",Authentication Response,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
	// ControlRecorder[ueid] = append(ControlRecorder[ueid], content)
	// ch_log <- content + "\n"

	//println(ue_list[ueid].ch)
	// Target RAN receive ngap Initial Context Setup Request Msg
	<-ue_list[ueid].ch
	// ** senario * trans_delay /ran to free5gc/
	// ** senario * trans_delay /ran to free5gc/
	time.Sleep(2 * time.Duration(ran_5g_dura) * time.Millisecond)

	// Target RAN send ngap Initial Context Setup Response Msg
	sendMsg, _ = test.GetInitialContextSetupResponseForServiceRequest(ue_list[ueid].ue.AmfUeNgapId, ue_list[ueid].ue.RanUeNgapId, ranN2Ipv4Addr)
	amf_conn.Write(sendMsg)
	t = time.Now().Format("2006-01-02 15:04:05.000000")
	InitialContextSetupResponse := strconv.Itoa(ueid) + ",Initial Context Setup Response,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
	ControlRecorder[ueid] = append(ControlRecorder[ueid], InitialContextSetupResponse)
	ch_log <- InitialContextSetupResponse + "\n"

	// Target RAN send NAS Registration Complete Msg
	pdu = nasTestpacket.GetRegistrationComplete(nil)
	pdu, _ = test.EncodeNasPduWithSecurity(ue_list[ueid].ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	sendMsg, _ = test.GetUplinkNASTransport(ue_list[ueid].ue.AmfUeNgapId, ue_list[ueid].ue.RanUeNgapId, pdu)
	amf_conn.Write(sendMsg)
	ue_list[ueid].state = connected

	t = time.Now().Format("2006-01-02 15:04:05.000000")
	content := strconv.Itoa(ueid) + ",Xn Handover Complete,ran" + ran_id + " send to ue," + strings.Split(t, " ")[1]
	controller_conn.Write(add_tcp_length(content))
	ControlRecorder[ueid] = append(ControlRecorder[ueid], content)
	ch_log <- content + "\n"

	if ueid > ran_number {
		content = strconv.Itoa(ueid) + ",RSRP_Interval,ran" + ran_id + ",100," + strings.Split(t, " ")[1]
		controller_conn.Write(add_tcp_length(content))
		ch_log <- content + "\n"
	}  // hand out the RSRP reporting interval 
	if ueid > ran_number {
		content = strconv.Itoa(ueid) + ",RSRP_Meas_Period,ran" + ran_id + ",4," + strings.Split(t, " ")[1]
		controller_conn.Write(add_tcp_length(content))
		ch_log <- content + "\n"
	}  //handout the RSRP measurement period of UE
	if ueid > ran_number {
		content = strconv.Itoa(ueid) + ",MR_Trigger_Event_Para,ran" + ran_id + ",Type,A2,Threshold,-100,TTT,80," + strings.Split(t, " ")[1]
		controller_conn.Write(add_tcp_length(content))
		ch_log <- content + "\n"
	}  //handout the trigger event of HO

}

func excuteN2Handover_sRAN(ueid int, ran_index int) {
	for wait := 0; wait < 3; wait++ {
		if ue_list[ueid].state != connected {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
		if wait == 2 {
			println("ue id:", ueid, "not in connected state for 3s. Abort handover procedure!")
			t := time.Now().Format("2006-01-02 15:04:05.000000")
			content := strconv.Itoa(ueid) + ",Abort Handover Procedure,ran" + ran_id + " after recv HO for 3s," + strings.Split(t, " ")[1]
			ch_log <- content + "\n"
			return
		}
	}
	var recvMsg = make([]byte, 2048)
	ue := ue_list[ueid].ue
	ue_list[ueid].state = duringN2_P

	// 第1步，收到ue消息后给amf发送handover required
	// Source RAN send ngap Handover Required Msg
	// func GetHandoverRequired(amfUeNgapID int64, ranUeNgapID int64, targetGNBID []byte, targetCellID []byte)
	gnb_id, _ := strconv.Atoi(nran_id[ran_index])
	sendMsg, _ := test.GetHandoverRequired(ue.AmfUeNgapId, ue.RanUeNgapId, []byte{0x00, 0x01, uint8(gnb_id + 1)}, []byte{0x01, 0x20})
	amf_conn.Write(sendMsg)

	t := time.Now().Format("2006-01-02 15:04:05.000000")
	HandoverRequired := strconv.Itoa(ueid) + ",Handover Required,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
	ControlRecorder[ueid] = append(ControlRecorder[ueid], HandoverRequired)
	ch_log <- HandoverRequired + "\n"

	for Continue {
		// fmt.Printf("%v\n", recvMsg)
		recvMsg = <-ue_list[ueid].ch
		switch int64(recvMsg[1]) {
		case ngapType.ProcedureCodeHandoverPreparation:
			// Source RAN receive ngap Handover Command
			ue_list[ueid].state = duringN2_E

			t = time.Now().Format("2006-01-02 15:04:05.000000")
			HandoverCommand := ControlRecorder[ueid][len(ControlRecorder[ueid])-1]
			ULDL := []byte{uint8(ue.ULCount.Overflow() >> 8), uint8(ue.ULCount.Overflow()), ue.ULCount.SQN(), uint8(ue.DLCount.Overflow() >> 8), uint8(ue.DLCount.Overflow()), ue.DLCount.SQN()}
			ue_list[ueid].NH = DerivateNH(ue.Kamf, ue_list[ueid].NH)

			fmt.Printf("NH send to ue: %v\n", ue_list[ueid].NH)
			Kgnb := DerivateAnKey(ue.Kamf, ue.ULCount.Get())
			new_NH := DerivateNH(ue.Kamf, Kgnb)
			fmt.Printf("NH generate from new kgnb: %v\n", new_NH)
			fmt.Printf("NH new when NCC==2: %v\n", DerivateNH(ue.Kamf, new_NH))

			HandoverCommand = strings.Replace(HandoverCommand, "Handover Command", "Handover Command||"+string(ULDL)+"||"+string(ue_list[ueid].NH)+"||", 1)
			HandoverCommand += ",ran" + ran_id + " send to ue," + strings.Split(t, " ")[1]
			controller_conn.Write(add_tcp_length(HandoverCommand))
			ControlRecorder[ueid] = append(ControlRecorder[ueid], HandoverCommand)
			ch_log <- HandoverCommand + "\n"
		case ngapType.ProcedureCodeUEContextRelease:
			// Source RAN receive ngap UE Context Release Command
			// Source RAN send ngap UE Context Release Complete
			pduSessionIDList := []int64{10}
			sendMsg, _ = test.GetUEContextReleaseComplete(ue.AmfUeNgapId, ue.RanUeNgapId, pduSessionIDList)
			amf_conn.Write(sendMsg)
			ue_list[ueid].state = initial
			t = time.Now().Format("2006-01-02 15:04:05.000000")
			ueContextReleaseComplete := strconv.Itoa(ueid) + ",UE Context Release Complete,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
			ControlRecorder[ueid] = append(ControlRecorder[ueid], ueContextReleaseComplete)
			ch_log <- ueContextReleaseComplete + "\n"
			return
		default:
			return
		}
	}
}

func excuteN2Handover_tRAN(amf_ueid int, teid string, ch chan string, ran_index int) {
	// Target RAN receive ngap Handover Request
	ran_ueid := ran_ueid_now
	ran_ueid_now += 1

	t := time.Now().Format("2006-01-02 15:04:05.000000")
	// 欠一个最前面的ueid信息
	HandoverRequest := ",Handover Request,ran" + ran_id + " recv from amf," + strings.Split(t, " ")[1]

	// Target RAN send ngap Handover Request Acknowledge Msg
	sendMsg, _ := test.GetHandoverRequestAcknowledge(int64(amf_ueid), int64(ran_ueid))
	amf_conn.Write(sendMsg)

	t = time.Now().Format("2006-01-02 15:04:05.000000")
	// 欠一个最前面的ueid信息
	HandoverRequestAcknowledge := ",Handover Request Acknowledge,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]

	// Beginning of Execution

	// Target RAN receive Handover Confirm from UE
	// wait ue connect and activate ch
	HandoverConfirm := <-ch
	HandoverConfirm_split := strings.Split(HandoverConfirm, ",")
	ueid, _ := strconv.Atoi(HandoverConfirm_split[0])
	ULDL := []byte(strings.Split(HandoverConfirm, "||")[1])

	// 更新需要的所有信息
	ue_list[ueid].state = duringN2_E

	// ori_ue := ue_list[ueid].ue
	ue_list[ueid].ue.AmfUeNgapId = int64(amf_ueid)
	ue_list[ueid].ue.RanUeNgapId = int64(ran_ueid)
	// amf_ueid_map[amf_ueid] = ueid
	amf_ueid_map.Store(amf_ueid, ueid)
	// ue := ue_list[ueid].ue
	ue_list[ueid].ue.ULCount.Set(uint16(ULDL[0])<<8+uint16(ULDL[1]), ULDL[2])
	ue_list[ueid].ue.DLCount.Set(uint16(ULDL[3])<<8+uint16(ULDL[4]), ULDL[5])

	if teid != "" && teid != ue_list[ueid].TEID {
		ue_list[ueid].TEID = teid
		send_allocate_ip := strconv.Itoa(ueid) + ",Allocate IP & TEID," + ue_list[ueid].IP + "," + teid
		controller_conn.Write(add_tcp_length(send_allocate_ip))
	}

	// 记录这几条日志
	HandoverRequest = HandoverConfirm_split[0] + HandoverRequest
	HandoverRequestAcknowledge = HandoverConfirm_split[0] + HandoverRequestAcknowledge

	ControlRecorder[ueid] = append(ControlRecorder[ueid], HandoverRequest)
	ch_log <- HandoverRequest + "\n"
	ControlRecorder[ueid] = append(ControlRecorder[ueid], HandoverRequestAcknowledge)
	ch_log <- HandoverRequestAcknowledge + "\n"
	ControlRecorder[ueid] = append(ControlRecorder[ueid], HandoverConfirm)
	ch_log <- HandoverConfirm + "\n"

	// Target RAN send ngap Handover Notify
	sendMsg, _ = test.GetHandoverNotify(ue_list[ueid].ue.AmfUeNgapId, ue_list[ueid].ue.RanUeNgapId)
	amf_conn.Write(sendMsg)

	t = time.Now().Format("2006-01-02 15:04:05.000000")
	HandoverNotify := strconv.Itoa(ueid) + ",Handover Notify,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
	ControlRecorder[ueid] = append(ControlRecorder[ueid], HandoverNotify)
	ch_log <- HandoverNotify + "\n"

	// 需要等待sran的ue context release command结束。姑且等待10ms代替
	time.Sleep(10 * time.Millisecond)

	// UE send NAS Registration Request(Mobility Registration Update) To Target AMF (2 AMF scenario not supportted yet)
	id := ueid + ueid/15
	l1 := uint8(id)
	l1 = l1>>4 | (l1&0x0f)<<4
	id /= 256
	l2 := uint8(id)
	l2 = l2>>4 | (l2&0x0f)<<4
	mobileIdentity5GS := nasType.MobileIdentity5GS{
		Len:    11, // 5g-guti
		Buffer: []uint8{0xf2, 0x02, 0xf8, 0x39, 0xca, 0xfe, 0x00, 0x00, 0x00, l2, l1},
	}
	uplinkDataStatus := nasType.NewUplinkDataStatus(nasMessage.RegistrationRequestUplinkDataStatusType)
	uplinkDataStatus.SetLen(2)
	uplinkDataStatus.SetPSI10(1)
	ueSecurityCapability := ue_list[ueid].ue.GetUESecurityCapability()
	pdu := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSMobilityRegistrationUpdating,
		mobileIdentity5GS, nil, ueSecurityCapability, ue_list[ueid].ue.Get5GMMCapability(), nil, uplinkDataStatus)
	pdu, _ = test.EncodeNasPduWithSecurity(ue_list[ueid].ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	sendMsg, _ = test.GetInitialUEMessage(ue_list[ueid].ue.RanUeNgapId, pdu, "")
	amf_conn.Write(sendMsg)

	t = time.Now().Format("2006-01-02 15:04:05.000000")
	RegistrationRequest := strconv.Itoa(ueid) + ",Registration Request(Mobility Registration Update),ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
	ControlRecorder[ueid] = append(ControlRecorder[ueid], RegistrationRequest)
	ch_log <- RegistrationRequest + "\n"

	// Target RAN receive ngap Initial Context Setup Request Msg
	<-ue_list[ueid].ch

	// Target RAN send ngap Initial Context Setup Response Msg
	sendMsg, _ = test.GetInitialContextSetupResponseForServiceRequest(ue_list[ueid].ue.AmfUeNgapId, ue_list[ueid].ue.RanUeNgapId, ranN2Ipv4Addr)
	amf_conn.Write(sendMsg)
	t = time.Now().Format("2006-01-02 15:04:05.000000")
	InitialContextSetupResponse := strconv.Itoa(ueid) + ",Initial Context Setup Response,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
	ControlRecorder[ueid] = append(ControlRecorder[ueid], InitialContextSetupResponse)
	ch_log <- InitialContextSetupResponse + "\n"

	// Target RAN send NAS Registration Complete Msg
	pdu = nasTestpacket.GetRegistrationComplete(nil)
	pdu, _ = test.EncodeNasPduWithSecurity(ue_list[ueid].ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	sendMsg, _ = test.GetUplinkNASTransport(ue_list[ueid].ue.AmfUeNgapId, ue_list[ueid].ue.RanUeNgapId, pdu)
	amf_conn.Write(sendMsg)
	ue_list[ueid].state = connected

	t = time.Now().Format("2006-01-02 15:04:05.000000")
	RegistrationComplete := strconv.Itoa(ueid) + ",Registration Complete,ran" + ran_id + " send to amf," + strings.Split(t, " ")[1]
	ControlRecorder[ueid] = append(ControlRecorder[ueid], RegistrationComplete)
	ch_log <- RegistrationComplete + "\n"
	// ue_list[ueid].ue = ue

	t = time.Now().Format("2006-01-02 15:04:05.000000")
	content := strconv.Itoa(ueid) + ",N2 Handover Complete,ran" + ran_id + " send to ue," + strings.Split(t, " ")[1]
	controller_conn.Write(add_tcp_length(content))
	ControlRecorder[ueid] = append(ControlRecorder[ueid], content)
	ch_log <- content + "\n"
}
