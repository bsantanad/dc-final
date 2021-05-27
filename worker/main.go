package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	pb "github.com/bsantanad/dc-final/proto"
	"github.com/esimov/stackblur-go"
	"google.golang.org/grpc"

	"go.nanomsg.org/mangos"
	"go.nanomsg.org/mangos/protocol/req"

	_ "go.nanomsg.org/mangos/transport/all"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

var (
	defaultRPCPort = 50051
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedFiltersServer
}

var (
	controllerAddress = ""
	workerName        = ""
	tags              = ""
)

// shared structs
type Worker struct {
	Name  string `json:"name"`
	Token string `json:"token"`
	Cpu   uint64 `json:"cpu"`
	Id    uint64 `json:"id"`
	Url   string `json:"url"`
}

var WorkerInfo Worker // stores worker name, token and cpu

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// SayHello implements helloworld.GreeterServer
func (s *server) GrayScale(ctx context.Context,
	in *pb.FilterRequest) (*pb.FilterReply, error) {
	// get image by id from api
	ioreader := getImage(in.GetId())
	// blur it
	blurImg := blur(ioreader)
	// post image

	fmt.Println(in.GetFilter())
	fmt.Println(in.GetId())
	return &pb.FilterReply{Message: "Hello "}, nil
}
func (s *server) Blur(ctx context.Context,
	in *pb.FilterRequest) (*pb.FilterReply, error) {

	fmt.Println("im in worker")
	fmt.Println(in.GetFilter())
	fmt.Println(in.GetId())
	return &pb.FilterReply{Message: "Hello "}, nil
}

func init() {
	flag.StringVar(&controllerAddress, "controller",
		"tcp://localhost:40899", "Controller address")
	flag.StringVar(&workerName, "worker-name",
		"hard-worker", "Worker Name")
	flag.StringVar(&tags, "tags", "gpu,superCPU,largeMemory",
		"Comma-separated worker tags")
}

func blur(img io.Reader) []byte {
	src, _, err := image.Decode(img)
	if err != nil {
		log.Fatal(err)
	}
	res := stackblur.Process(src, uint32(5))
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, res, nil)
	return buf.Bytes()
}

func getImage(imageId string) io.Reader {
	url := WorkerInfo.Url + "/image/" + imageId
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Bearer "+WorkerInfo.Token)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return resp.Body
}

// FIXME we were trying to post the image how to use form in http
// Content-Type: multipart/form-data
func postImage(imageId string) io.Reader {
	url := WorkerInfo.Url + "/image"
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, nil)
	req.Header.Add("Authorization", "Bearer "+WorkerInfo.Token)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return resp.Body
}

// joinCluster works with controller in a REQREP way. The worker
// tells the controller that he is up and running (sends name and cpu).
// The controller returns a token for him to use the api
func joinCluster(url string) {
	var sock mangos.Socket
	var err error
	var msg []byte

	// make the request
	if sock, err = req.NewSocket(); err != nil {
		die("can't get new req socket: %s", err.Error())
	}
	fmt.Println(controllerAddress)
	if err = sock.Dial(controllerAddress); err != nil {
		die("can't dial on req socket: %s\n%s", err.Error(), controllerAddress)
	}
	stat, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		fmt.Println("stat read fail")
	}

	var myInfo Worker
	myInfo.Name = workerName
	myInfo.Cpu = stat.CPUStatAll.User
	myInfo.Url = url
	infoStr, err := json.Marshal(myInfo)
	if err != nil {
		fmt.Println("worker coudn't get his info")
		return
	}

	// send worker info to controller
	if err = sock.Send([]byte(infoStr)); err != nil {
		die("can't send message on push socket: %s", err.Error())
	}

	// receive controller response (worker struct with token)
	if msg, err = sock.Recv(); err != nil {
		die("can't receive date: %s", err.Error())
	}
	var tmp Worker
	err = json.Unmarshal(msg, &tmp)
	if err != nil {
		fmt.Println("[ERROR] worker couldnt parse worker\n" +
			"bad json sent")
		return
	}

	WorkerInfo = tmp
	fmt.Println("[INFO] worker " + tmp.Name + " has been registered with " +
		"workers id: " + strconv.FormatUint(tmp.Id, 10))
	sock.Close()
}

func getAvailablePort() int {
	port := defaultRPCPort
	for {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
		if err != nil {
			port = port + 1
			continue
		}
		ln.Close()
		break
	}
	return port
}

func main() {
	flag.Parse()

	// Setup Worker RPC Server
	rpcPort := getAvailablePort()
	log.Printf("Starting RPC Service on localhost:%v", rpcPort)

	// Subscribe to Controller
	hostname := "localhost:" + strconv.Itoa(rpcPort)
	go joinCluster(hostname)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", rpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterFiltersServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
