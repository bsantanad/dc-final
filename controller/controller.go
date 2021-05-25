package controller

import (
	"fmt"
	"encoding/json"
	"os"

	"go.nanomsg.org/mangos"
	"go.nanomsg.org/mangos/protocol/pull"

	_ "go.nanomsg.org/mangos/transport/all"
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
var Workloads []Workload

var workloadsUrl = "tcp://localhost:40899"

func receiveWorkloads() {
	var sock mangos.Socket
	var err error
	var msg []byte

	if sock, err = pull.NewSocket(); err != nil {
		die("can't get new pull socket: %s", err)
	}
	if err = sock.Listen(workloadsUrl); err != nil {
		die("can't listen on pull socket: %s", err.Error())
	}
	for {
		// Could also use sock.RecvMsg to get header
		msg, err = sock.Recv()
		if err != nil {
			die("cannot receive from mangos Socket: %s", err.Error())
		}

        // after getting json string, convert it to workload struct
        // and add it to the fake database
		var workload Workload
		json.Unmarshal(msg, &workload)
		Workloads = append(Workloads, workload)
		fmt.Println(Workloads)
	}
}

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func Start() {
	receiveWorkloads()
}
