package main

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

func next_ran(ran_now int) int {
	if _, ok := next_list[scenario]; !ok {
		var ran_list []int
		for i := 0; i < ran_number; i++ {
			ran_list = append(ran_list, i+1)
		}

		switch scenario {
		case "loop":
			ran_list[ran_number-1] = 1
		default:
			ran_list[ran_number-1] = 0
		}
		next_list[scenario] = ran_list
	}

	if ran_now == -1 {
		return -1
	}
	return next_list[scenario][ran_now]
}

func init_scenario() bool {
	// 5gc
	// if auto_run {
	// 	println("Starting 5gc...")
	// 	cmd := exec.Command("nohup", "/root/free5gc/run.sh", "&")
	// 	cmd.Run()
	// 	time.Sleep(3 * time.Second)
	// }

	ue_number = viper.GetInt("scenario." + scenario + ".ue")
	ran_number = viper.GetInt("scenario." + scenario + ".ran")
	var err error
	// TODO: setup ran automatically

	// imsi和amf-ueid必须从1开始，因此方便起见加入一个空ue放入list占位  这个无用，空ue；
	// 目标是保证controller的ue_list和每个ran的ue_list中：ue_list的index、id保持一致
	new_ue := UE{
		state:      initial,
		pkg_index:  0,
		ran_now:    -1,
		ran_to:     -1,
		allow_send: false,
	}
	ue_list = append(ue_list, new_ue)

	base_ranip := 2

	// check access to all ran and baidu
	println("Checking online...")
	check_status := struct {
		count    int
		fail_url string
	}{0, ""}
	ch_check := make(chan struct{})
	go check_access("baidu.com", ch_check, &check_status)
	for i := 0; i < ran_number; i++ {
		ran_ip := "172.17.0." + strconv.Itoa(base_ranip+i)
		go check_access(ran_ip, ch_check, &check_status)
	}
	for i := 0; i < ran_number+1; i++ {
		select {
		case <-ch_check:
			if check_status.fail_url != "" {
				println("Check access to", check_status.fail_url, "failed. Please check network.")
				return false
			}
		case <-time.After(5 * time.Second):
			println("Time out. Please check network.")
			return false
		}
		if check_status.count == ran_number+1 {
			break
		}
	}

	// ssh to all ran and start services
	// TODO 自动启动
	if auto_run {
		println("Remote starting ran services...")
		for i := 0; i < ran_number; i++ {
			ran_ip := "172.17.0." + strconv.Itoa(base_ranip+i)
			go autorun(ran_ip, "ran")
		}
		time.Sleep(1 * time.Second)
	}

	// ue_list的index就是ue的唯一标识，ran与controller维护相同长度的ue_list。同时ran需要做这个index与amf_ueid/ran_ueid的转译。**
	println("Creating ran connections...")
	for i := 0; i < ran_number; i++ {
		ran_ip := "172.17.0." + strconv.Itoa(base_ranip+i) + ":9000"
		conn, err := net.Dial("tcp", ran_ip)
		for err != nil {
			time.Sleep(1 * time.Second)
			conn, err = net.Dial("tcp", ran_ip)
		}
		conn_list = append(conn_list, conn) //把所有链接放到一个list中，用于维护；
		go listen_ran(conn)                 //在controller.go中；
		fmt.Printf("connect %v success\n", ran_ip)

		// for j:=0; j<background_ue_number; j++ {   这个是真正的有效UE；
		new_ue := UE{
			state:      initial,
			pkg_index:  0,
			ran_now:    i,
			ran_to:     -1,
			allow_send: true,
		}
		ue_list = append(ue_list, new_ue)
	}

	for i := 0; i < ue_number; i++ {
		new_ue := UE{
			state:      initial,
			pkg_index:  0,
			ran_now:    FIRST_RAN,
			ran_to:     1,
			allow_send: true,
		}
		ue_list = append(ue_list, new_ue)
	}

	if open_uevm {
		if auto_run {
			go autorun("10.156.168.52", "ue")
			time.Sleep(time.Second)
		}
		println("Creating UE VM connection...")
		uevm_conn, err = net.Dial("tcp", uevm_ip)
		for err != nil {
			time.Sleep(1 * time.Second)
			uevm_conn, err = net.Dial("tcp", uevm_ip)
		}
		// ran数量，HO ue数量
		uevm_conn.Write(add_tcp_length(viper.GetString("scenario."+scenario+".ran") + "," + viper.GetString("scenario."+scenario+".ue") + "," + ue_nic_limit))
		fmt.Printf("connect UE VM success\n")
	}

	println("initializing scenario...")
	switch scenario {
	case "single":
		init_single()
	case "loop":
		init_loop()
	case "batch":
		init_batch()
	case "multi_connect":
		init_multi_connect()
	case "test_run":
		init_test_run()
	}
	return true
}

func init_single() {

}

func single() {
	// time.Sleep(1 * time.Second)

	fmt.Println("ue connect to 5gc")
	// 背景ue连接
	for i := 0; i < ran_number; i++ {
		command := strconv.Itoa(i+1) + ",Connect UE"
		go send_message(conn_list[i], command)
		time.Sleep(100 * time.Millisecond)
		ue_list[i+1].state = connected
	}

	// HO ue连接
	for i := 0; i < ue_number; i++ {
		command := strconv.Itoa(ran_number+i+1) + ",Connect UE"
		go send_message(conn_list[FIRST_RAN], command)
		// time.Sleep(50 * time.Millisecond)
		ue_list[ran_number+i+1].state = connected
	}

	select {
	case <-ch_scenario:
	case <-time.After(time.Second * 3000):
	}
	// time.Sleep(3 * time.Second)

	// fmt.Println("handle " + handover_type + " handover") //Controller端改写申请HO到下一个基站；
	// for i := 0; i < ue_number; i++ {
	// 	command := strconv.Itoa(ran_number+i+1) + ",HO to next ran"
	// 	go send_message(conn_list[0], command)
	// 	// time.Sleep(2 * time.Millisecond)
	// 	ue_list[ran_number+i+1].state = duringXn
	// }

	time.Sleep(3 * time.Second)
	ch_control <- "q"
	time.Sleep(2 * time.Second)
	// release()
}

func init_loop() {

}

func loop() {

}

func init_batch() {

}

func batch() {
	fmt.Println("ue connect to 5gc")
	// 背景ue连接
	for i := 0; i < ran_number; i++ {
		command := strconv.Itoa(i+1) + ",Connect UE"
		go send_message(conn_list[i], command)
		time.Sleep(100 * time.Millisecond)
		ue_list[i+1].state = connected
	}

	// HO ue连接
	for i := 0; i < ue_number; i++ {
		command := strconv.Itoa(ran_number+i+1) + ",Connect UE"
		go send_message(conn_list[0], command)
		time.Sleep(100 * time.Millisecond)
		ue_list[ran_number+i+1].state = connected
	}
	// time.Sleep(10 * time.Second)

	select {
	case <-ch_scenario:
	case <-time.After(time.Second * 3000):
	}
	// 做ran_num-1次HO
	for times := 0; times < ran_number-1; times++ {
		fmt.Println("handle " + handover_type + " handover")
		for i := ran_number + ue_number; i > ran_number; i-- {
			command := strconv.Itoa(i) + ",HO to next ran"
			go send_message(conn_list[ue_list[i].ran_now], command)
			time.Sleep(5 * time.Millisecond)
			ue_list[i].state = duringXn
		}
		time.Sleep(10 * time.Second)
	}

	ch_control <- "q"
	time.Sleep(2 * time.Second)
}

func init_multi_connect() {

}

func multi_connect() {

}

func init_test_run() {

}

func test_run() {
	// time.Sleep(1 * time.Second)

	fmt.Println("ue connect to 5gc")
	// 背景ue连接
	for i := 0; i < ran_number; i++ {
		command := strconv.Itoa(i+1) + ",Connect UE"
		go send_message(conn_list[i], command)
		time.Sleep(100 * time.Millisecond)
		ue_list[i+1].state = connected
	}

	// HO ue连接
	for i := 0; i < ue_number; i++ {
		command := strconv.Itoa(ran_number+i+1) + ",Connect UE"
		go send_message(conn_list[0], command)
		time.Sleep(150 * time.Millisecond)
		ue_list[ran_number+i+1].state = connected
	}

	select {
	case <-ch_scenario:
	case <-time.After(time.Second * 300000):
	}

	if viper.GetBool("scenario." + scenario + ".handover") {
		// time.Sleep(15 * time.Second)
		fmt.Println("handle " + handover_type + " handover")
		// for i := 0; i < ue_number; i++ {
		for i := 0; i < 10; i++ {
			command := strconv.Itoa(ran_number+i+1) + ",HO to next ran"
			go send_message(conn_list[0], command)
			time.Sleep(20 * time.Millisecond)
			ue_list[ran_number+i+1].state = duringXn
		}

		time.Sleep(10 * time.Second)
		select {
		case <-ch_scenario:
		case <-time.After(time.Second * 3000):
		}
		ch_control <- "q"
	}

	time.Sleep(2 * time.Second)
	// release()
}
