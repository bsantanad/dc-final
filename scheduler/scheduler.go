package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	pb "github.com/bsantanad/dc-final/proto"
	"google.golang.org/grpc"

	"go.nanomsg.org/mangos"
	"go.nanomsg.org/mangos/protocol/pull"
	_ "go.nanomsg.org/mangos/transport/all"
)

//const (
//	address     = "localhost:50051"
//	defaultName = "world"
//)
/*
type Job struct {
	Address string
	RPCName string
}
*/
var schedulerUrl = "tcp://localhost:40902"

type Worker struct {
	Name  string `json:"name"`
	Token string `json:"token"`
	Cpu   uint64 `json:"cpu"`
	Id    uint64 `json:"id"`
	Url   string `json:"url"`
	Api   string `json:"api"`
}

type Job struct {
	Filter  string   `json:"filter"`
	ImageId uint64   `json:"image_id"`
	Workers []Worker `json:"workers"`
}

func schedule(job Job) {

	fmt.Println("im entered job to scheduler")
	fmt.Println(job)
	if job.Filter == "" {
		return
	}

	// sort array of workers by cpu usage
	sort.Slice(job.Workers[:], func(i, j int) bool {
		return job.Workers[i].Cpu < job.Workers[j].Cpu
	})

	url := job.Workers[0].Url
	filter := job.Filter
	imageId := strconv.FormatUint(job.ImageId, 10)

	fmt.Println("im bfore dial to grpc in job to scheduler")
	// Set up a connection to the server.
	conn, err := grpc.Dial(url, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewFiltersClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	fmt.Println("im bfore grayscale request grpc in job to scheduler")
	r, err := c.GrayScale(ctx, &pb.FilterRequest{Filter: filter, Id: imageId})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	fmt.Println(r.GetMessage())
}

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func Start() {
	var sock mangos.Socket
	var err error
	var msg []byte

	if sock, err = pull.NewSocket(); err != nil {
		die("can't get new pull socket: %s", err)
	}
	if err = sock.Listen(schedulerUrl); err != nil {
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
		var job Job
		err = json.Unmarshal(msg, &job)
		if err != nil {
			fmt.Println("[ERROR] controller couldnt parse to image\n" +
				"bad json sent")
		}
		schedule(job)
	}
}
