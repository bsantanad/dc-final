package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	pb "github.com/bsantanad/dc-final/proto"
	"github.com/anthonynsimon/bild/blur"
	"github.com/anthonynsimon/bild/imgio"
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
	Api   string `json:"api"`
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
	imageName := getImage(in.GetId())
	if len(imageName) == 0 {
		return &pb.FilterReply{Message: "bad image"}, nil
	}
	// blur it
	blury(imageName)
	// post image
	postImage(imageName)

	fmt.Println(in.GetFilter())
	return &pb.FilterReply{Message: "hello "}, nil
}
func (s *server) Blur(ctx context.Context,
	in *pb.FilterRequest) (*pb.FilterReply, error) {
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

func blury(name string) {
	//img, err := os.Open(name)
	//defer img.Close()
	img, err := imgio.Open(name)
	if err != nil {
		fmt.Println(err)
		return
	}

	result := blur.Gaussian(img, 10.0)

	if err := imgio.Save(name,
		result, imgio.PNGEncoder()); err != nil {
		fmt.Println(err)
		return
	}
}

func getImage(imageId string) string {
	url := WorkerInfo.Api + "/images/" + imageId
	//fmt.Println(url)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Bearer "+WorkerInfo.Token)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	if resp.StatusCode != 200 {
		fmt.Println("bad status: %s", resp.Status)
		return ""
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	// download image
	permissions := 0775
	name := "tmp_image" + imageId
	err = ioutil.WriteFile(name, body, os.FileMode(permissions))
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	return name
}

func postImage(name string) {
	url := WorkerInfo.Api + "/images"
	client := &http.Client{}
	//prepare the reader instances to encode
	values := map[string]io.Reader{
		"data": mustOpen(name), // lets assume its this file
		"type": strings.NewReader("filtered"),
	}
	err := Upload(client, url, values)
	if err != nil {
		panic(err)
	}
}

func mustOpen(f string) *os.File {
	r, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	return r
}

// code from https://stackoverflow.com/a/20397167
func Upload(client *http.Client, url string,
	values map[string]io.Reader) (err error) {

	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		// Add an image file
		if x, ok := r.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				return
			}
		} else {
			// Add other fields
			if fw, err = w.CreateFormField(key); err != nil {
				return
			}
		}
		if _, err = io.Copy(fw, r); err != nil {
			return err
		}

	}
	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Add("Authorization", "Bearer "+WorkerInfo.Token)
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Submit the request
	res, err := client.Do(req)
	if err != nil {
		return
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}
	return
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
