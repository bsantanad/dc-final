package controller

import (
	"encoding/json"
	"fmt"
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

type Image struct {
	WorkloadId uint64 `json:"workload_id"`
	ImageId    uint64 `json:"image_id"`
	Type       string `json:"type"`
	Data       []byte `json:"data"`
}

// end shared structs

// fake database
var Workloads []Workload
var Images []Image

var workloadsUrl = "tcp://localhost:40899"
var imagesUrl = "tcp://localhost:40900"

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
		err = json.Unmarshal(msg, &workload)
		if err != nil {
			fmt.Println("[ERROR] controller couldnt parse to image\n" +
				"bad json sent")
		}
		instertWorkload(workload)
		fmt.Println(Workloads)
	}
}
func receiveImages() {
	var sock mangos.Socket
	var err error
	var msg []byte

	if sock, err = pull.NewSocket(); err != nil {
		die("can't get new pull socket: %s", err)
	}
	if err = sock.Listen(imagesUrl); err != nil {
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
		var image Image
		err = json.Unmarshal(msg, &image)
		if err != nil {
			fmt.Println("[ERROR] controller couldnt parse to image\n" +
				"bad json sent")
		}

		// add image to fake db
		Images = append(Images, image)
		//fmt.Println(Images)
	}
}

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func instertWorkload(workload Workload) {
	id := workload.Id
	if id > uint64(len(Workloads))-1 || id == 0 {
		Workloads = append(Workloads, workload)
		return
	}

	Workloads[id] = workload
	return
}

func Start() {
	go receiveWorkloads()
	go receiveImages() //TODO create dir for images and store the image there
	// every workload received create dir
	// every image received store in dir
}
