package scheduler

import (
	//"context"
	//"log"
	//"time"
	"encoding/json"
	"fmt"
	"os"

	//pb "github.com/CodersSquad/dc-final/proto"
	//"google.golang.org/grpc"

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
}

type Job struct {
	Filter  string `json:"filter"`
	ImageId uint64 `json:"image_id"`
}

func schedule(job Job) {

	fmt.Println("im in jobs")
	fmt.Println(job)
	/*
		// Set up a connection to the server.
		conn, err := grpc.Dial(job.Address, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := pb.NewGreeterClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		r, err := c.SayHello(ctx, &pb.HelloRequest{Name: job.RPCName})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}
		log.Printf("Scheduler: RPC respose from %s : %s", job.Address, r.GetMessage())
	*/
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
