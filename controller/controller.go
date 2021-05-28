package controller

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	"go.nanomsg.org/mangos"
	"go.nanomsg.org/mangos/protocol/pull"
	"go.nanomsg.org/mangos/protocol/push"
	"go.nanomsg.org/mangos/protocol/rep"

	_ "go.nanomsg.org/mangos/transport/all"
	//"github.com/bsantanad/dc-final/scheduler"
)

// shared structs
type Workload struct {
	Id          uint64   `json:"workload_id"`
	Filter      string   `json:"filter"`
	Name        string   `json:"workload_name"`
	Status      string   `json:"status"`
	RunningJobs int      `json:"running_jobs"`
	Images      []uint64 `json:"filtered_images"`
}

type Image struct {
	WorkloadId uint64 `json:"workload_id"`
	Id         uint64 `json:"image_id"`
	Type       string `json:"type"`
	Data       []byte `json:"data"`
	Size       int    `json:"size"`
}
type Worker struct {
	Name  string `json:"name"`
	Token string `json:"token"`
	Cpu   uint64 `json:"cpu"`
	Id    uint64 `json:"id"`
	Url   string `json:"url"`
	Api   string `json:"api"`
}

type LoginResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
}

type Job struct {
	Filter  string   `json:"filter"`
	ImageId uint64   `json:"image_id"`
	Workers []Worker `json:"workers"`
}

// end shared structs

// fake database
var Workloads []Workload
var Images []Image
var Workers []Worker

// id manager
var workersIds uint64

var apiUrl = "http://localhost:8080"
var workloadsUrl = "tcp://localhost:40899"
var imagesUrl = "tcp://localhost:40900"
var workersUrl = "tcp://localhost:40901"
var schedulerUrl = "tcp://localhost:40902"

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

		job := checkForWork(workload)
		job.Workers = Workers
		jobStr, err := json.Marshal(job)
		if err != nil {
			die("cannot parse job to json string: %s", err.Error())
		}
		pushJob(schedulerUrl, string(jobStr))
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

func listenWorkers() {
	// REQREP
	var sock mangos.Socket
	var err error
	var msg []byte
	if sock, err = rep.NewSocket(); err != nil {
		die("can't get new rep socket: %s", err)
	}
	if err = sock.Listen(workersUrl); err != nil {
		die("can't listen on rep socket: %s", err.Error())
	}
	for {
		// Could also use sock.RecvMsg to get header
		msg, err = sock.Recv()
		if err != nil {
			die("cannot receive on rep socket: %s", err.Error())
		}
		var worker Worker
		err = json.Unmarshal(msg, &worker)
		if err != nil {
			fmt.Println("[ERROR] controller couldnt parse worker\n" +
				"bad json sent")
			continue
		}
		if worker.Name == "" {
			fmt.Println("[ERROR] worker is missing info\n")
			continue
		}
		fmt.Println("[INFO] worker: " + worker.Name + " has requested a token")
		worker.Token = getCredentials(worker.Name)
		worker.Id = workersIds
		worker.Api = apiUrl
		workersIds++

		Workers = append(Workers, worker)

		workerStr, err := json.Marshal(worker)
		if err != nil {
			fmt.Println("[ERROR] worker is missing info\n")
			continue
		}
		err = sock.Send([]byte(workerStr))
		if err != nil {
			die("can't send reply: %s", err.Error())
		}
	}
}

func pushJob(url string, msg string) {
	var sock mangos.Socket
	var err error

	if sock, err = push.NewSocket(); err != nil {
		die("can't get new push socket: %s", err.Error())
	}
	if err = sock.Dial(url); err != nil {
		die("can't dial on push socket: %s", err.Error())
	}
	if err = sock.Send([]byte(msg)); err != nil {
		die("can't send message on push socket: %s", err.Error())
	}
	time.Sleep(time.Second / 10)
	sock.Close()
}

// make POST /login endpoint
func getCredentials(name string) string {
	psswd := generatePassword(10)
	client := &http.Client{}
	req, err := http.NewRequest("POST", apiUrl+"/login", nil)
	req.Header.Add("Authorization", "Basic "+basicAuth(name, psswd))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	bodyText, err := ioutil.ReadAll(resp.Body)
	//jsony := string(bodyText)
	var login LoginResponse
	err = json.Unmarshal(bodyText, &login)
	if err != nil {
		fmt.Println("[ERROR] controller couldnt parse login response\n")
		return ""
	}
	return login.Token
}

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func instertWorkload(workload Workload) {

	if len(Workloads) == 0 {
		Workloads = append(Workloads, workload)
		return
	}
	id := workload.Id
	if id > uint64(len(Workloads))-1 {
		Workloads = append(Workloads, workload)
		return
	}

	Workloads[id] = workload
	return
}

/********* http requests helper *******************/
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generatePassword(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// sends info for the creation of a job in main.go
func checkForWork(load Workload) Job {
	if len(load.Images) < 1 {
		return Job{}
	}

	var job Job
	job.Filter = load.Filter
	job.ImageId = load.Images[len(load.Images)-1]
	return job
}

func Start() {
	rand.Seed(time.Now().UnixNano())
	//Jobs := make(chan scheduler.Job)
	go receiveWorkloads()
	go receiveImages()
	go listenWorkers()

}
