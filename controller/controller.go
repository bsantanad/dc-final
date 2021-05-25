package controller

import (
	"fmt"
	//"log"
	"os"
	//"time"

	//"go.nanomsg.org/mangos"
	"nanomsg.org/go/mangos/v2"
	"nanomsg.org/go/mangos/v2/protocol/pull"
	//"go.nanomsg.org/mangos/protocol/sub"
	//"go.nanomsg.org/mangos/protocol/pub"

	// register transports
	_ "nanomsg.org/go/mangos/v2/transport/all"
	//_ "go.nanomsg.org/mangos/transport/all"
)

// shared structs
type Workload struct {
	Id             uint64   `json:"workload_id"`
	Filter         string   `json:"filter"`
	Name           string   `json:"workload_name"`
	Status         string   `json:"status"`
	RunningJobs    int      `json:"running_jobs"`
	FilteredImages []uint64 `json:"filtered_images"`
}

// end shared structs

// fake database
var workloads []Workload

var address = "tcp://localhost:40899"

func receiveWorkloads() {
	var sock mangos.Socket
	var err error
	var msg []byte

	sock, err = pull.NewSocket()
	if err != nil {
		die("can't get new pull socket: %s", err.Error())
	}
	err = sock.Listen(address)
	if err != nil {
		die("can't listen on pull socket: %s", err.Error())
	}
	for {
		fmt.Println("im listening")
		msg, err = sock.Recv()
		fmt.Printf("NODE0: RECEIVED \"%s\"\n", msg)
		fmt.Printf(err.Error())
	}
}

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func Start() {
	receiveWorkloads()
}
