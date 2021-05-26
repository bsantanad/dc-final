// SPDX-License-Identifier: GPL-3.0-or-later
// authors: bsantanad & renataaparicio

//FIXME DRY - do not repeat yourself - the workloads array is repeated
// in both the controller and the API, this can be fixed changing the
// scalability protocol from PIPELINE to PAIR
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"go.nanomsg.org/mangos"
	"go.nanomsg.org/mangos/protocol/push"
)

type LoginResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
}

/*
type Image struct {
	Name string `json:"name"`
	Size int    `json:"size"`
	Data []byte `json:"data"`
}
*/

type Image struct {
	WorkloadId uint64 `json:"workload_id"`
	Id         uint64 `json:"image_id"`
	Type       string `json:"type"`
	Data       []byte `json:"data"`
	Size       int    `json:"size"`
}

type User struct {
	Username string  `json:"user"`
	Token    string  `json:"token"`
	Images   []Image `json:"image"`
	Time     string  `json:"time"`
}

type Status struct {
	SystemName string     `json:"system_name"`
	ServerTime string     `json:"server_time"`
	Workloads  []Workload `json:"active_workloads"`
}

type ImageMsg struct {
	Message    string `json:"message"`
	WorkloadId uint64 `json:"workload_id"`
	ImageId    uint64 `json:"image_id"`
	Type       string `json:"type"`
	Size       int    `json:"size"`
}

type Message struct {
	Message string `json:"message"`
}

type WorkloadReq struct {
	Filter       string `json:"filter"`
	WorkloadName string `json:"workload_name"`
}

type Workload struct {
	Id          uint64   `json:"workload_id"`
	Filter      string   `json:"filter"`
	Name        string   `json:"workload_name"`
	Status      string   `json:"status"`
	RunningJobs int      `json:"running_jobs"`
	Images      []uint64 `json:"filtered_images"`
}

type ImageReq struct {
	WorkloadId string `json:"workload_id"`
}

var Users []User /* this will act as our DB */
var Workloads []Workload
var Images []Image
var workloadsIds uint64
var imagesIds uint64

/***************** send msg via pipeline ****/
var workloadsUrl = "tcp://localhost:40899"
var imagesUrl = "tcp://localhost:40900"

func pushMsg(url string, msg string) {
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
func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

/********************* Endpoint Functions ***************************/

func homePage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	returnMsg(w, "DPIP REST API index. Invalid enpoints will redirect here")
	fmt.Println("[INFO]: / requested")
}

// postLogin will get the hash that's generated by default
// by the header "Authorization", then it will use it
// as the token for this particular user.
// It will also add the user to the "DB" of users, along
// with it's token
func postLogin(w http.ResponseWriter, r *http.Request) {
	var token string
	var user string
	var tmp string

	fmt.Println("[INFO]: POST /login requested")
	user, _, _ = r.BasicAuth() //get username
	tmp = r.Header.Get("Authorization")
	token = strings.Fields(tmp)[1] // get the hash from header

	//Build response
	var login LoginResponse
	login = LoginResponse{
		Message: "Hi " + user + ", welcome to the DPIP System",
		Token:   token,
	}

	var userInfo User
	userInfo = User{
		Username: user,
		Token:    token,
		Time:     time.Now().UTC().String(),
	}
	Users = append(Users, userInfo)

	json.NewEncoder(w).Encode(login)
}

// delLogout function will revoke a token from being usable.
// first it checks if the headers are sent in the correct
// format, then it will search the token in the Users "DB"
// if found it will remove it, if not, it will return 400
func delLogout(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO]: DELETE /logout requested")
	tmp := r.Header.Get("Authorization")
	if strings.Fields(tmp)[0] != "Bearer" {
		w.WriteHeader(400)
		returnMsg(w, "bad request, check headers "+
			"you must send a Bearer token")
		return
	}
	token := strings.Fields(tmp)[1] // get the token from header
	index, user, exists := searchToken(token)
	if !exists {
		w.WriteHeader(400)
		returnMsg(w, "token not found, "+
			"please provide a valid one")
		return
	}

	Users = removeUser(Users, index)
	returnMsg(w, "Bye "+user.Username+", your token has been revoked")
}

// based on https://stackoverflow.com/a/40699578
// postImages, upload a file (image).
// It first checks the headers and find the token,
// validates it and finds the user.
// Then creates a buffer, copy the bytes of the image
// to it and fills the Image struct.
// Finally it append the image to the Image slice
// the user has.
func postImages(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO]: POST /images requested")
	tmp := r.Header.Get("Authorization")
	if strings.Fields(tmp)[0] != "Bearer" {
		w.WriteHeader(400)
		returnMsg(w, "bad request, check headers "+
			"you must send a Bearer token")
		return
	}
	token := strings.Fields(tmp)[1] // get the token from header
	index, user, exists := searchToken(token)
	if !exists {
		w.WriteHeader(400)
		returnMsg(w, "token not found, "+
			"please provide a valid one")
		return
	}

	// uploading the file part
	r.ParseMultipartForm(32 << 20) // limit your max input length!
	file, _, err := r.FormFile("data")
	if err != nil {
		w.WriteHeader(400)
		returnMsg(w, err.Error())
		return
	}
	defer file.Close()

	// validate workloads id
	wrkId := r.FormValue("workload_id")
	imgType := r.FormValue("type")
	if wrkId == "" ||
		imgType == "" {
		w.WriteHeader(400)
		returnMsg(w, "the form sent is missing workload_id or type")
		return
	}
	// validate id
	workloadId, err := strconv.ParseUint(wrkId, 10, 64)
	if workloadId >= workloadsIds || workloadsIds == 0 {
		w.WriteHeader(400)
		returnMsg(w, "the workload id doesnt exists, "+
			"please check again, you may have to create a workload first."+
			" If you have, then check that the id you sent is in fact correct")
		return
	}
	// validate type
	fmt.Println(imgType)
	if imgType != "original" && imgType != "filtered" {
		w.WriteHeader(400)
		returnMsg(w, "the type sent isnt valid, "+
			"try with original or filtered")
		return
	}

	// Copy the image data to my buffer
	var buf bytes.Buffer
	io.Copy(&buf, file)

	// Fill the image struct
	var image Image
	image.WorkloadId = workloadId
	image.Id = imagesIds
	imagesIds += 1
	image.Type = imgType
	image.Data = buf.Bytes()
	image.Size = len(image.Data)
	/*
		if err != nil {
			w.WriteHeader(409)
			returnMsg(w, "Image couldn't be uploaded :(. Please try again")
			return
		}
	*/
	Users[index].Images = append(user.Images, image)

	// add image to workload's image array
	Workloads[workloadId].Images = append(Workloads[workloadId].Images,
		image.Id)
	workload := Workloads[workloadId]
	// transform to string
	wrkStr, err := json.Marshal(workload)
	if err != nil {
		w.WriteHeader(500)
		returnMsg(w, "server internal error, "+
			"couldnt marshal json")
		return
	}
	pushMsg(workloadsUrl, string(wrkStr))

	// transform to string
	imgStr, err := json.Marshal(image)
	if err != nil {
		w.WriteHeader(500)
		returnMsg(w, "server internal error, "+
			"couldnt marshal json")
		return
	}
	// push image to controller
	pushMsg(imagesUrl, string(imgStr))

	// add image to fake db
	Images = append(Images, image)

	var msg ImageMsg
	msg = ImageMsg{
		Message:    "An image has been successfully uploaded :)",
		WorkloadId: image.WorkloadId,
		ImageId:    image.Id,
		Type:       image.Type,
		Size:       image.Size,
	}

	buf.Reset()
	json.NewEncoder(w).Encode(msg)
}

func getImages(w http.ResponseWriter, r *http.Request) {
	// handle token
	tmp := r.Header.Get("Authorization")
	if strings.Fields(tmp)[0] != "Bearer" {
		w.WriteHeader(400)
		returnMsg(w, "bad request, check headers "+
			"you must send a Bearer token")
		return
	}
	token := strings.Fields(tmp)[1] // get the token from header
	_, _, exists := searchToken(token)
	//fmt.Println(user)
	if !exists {
		w.WriteHeader(400)
		returnMsg(w, "token not found, "+
			"please provide a valid one")
		return
	}

	// read path params
	vars := mux.Vars(r)
	id := vars["image_id"]
	if id == "" {
		w.WriteHeader(400)
		returnMsg(w, "id missing, "+
			"you should do smthg like images/{image_id}")
		return
	}
	fmt.Println("[INFO]: GET /images/" + id + " requested")

	intId, err := strconv.ParseUint(id, 10, 64)
	// validate id
	if intId >= imagesIds || imagesIds == 0 {
		w.WriteHeader(400)
		returnMsg(w, "the image id doesnt exists")
		return
	}

	permissions := 0775
	actualImg := Images[intId].Data
	err = ioutil.WriteFile(id, actualImg, os.FileMode(permissions))
	if err != nil {
		w.WriteHeader(500)
		returnMsg(w, "internal server error"+
			"couldn't get image")
		return
	}

	// download images
	w.WriteHeader(200)
	returnMsg(w, "image downloaded as "+id)
	return

}

// getStatus, show the status of the account related
// to the token sent in the header, proper validations
// are done, and then the creation time, and a msg is
// returned to the user
func getStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO]: GET /status requested")
	tmp := r.Header.Get("Authorization")
	if strings.Fields(tmp)[0] != "Bearer" {
		w.WriteHeader(400)
		returnMsg(w, "bad request, check headers "+
			"you must send a Bearer token")
		return
	}
	token := strings.Fields(tmp)[1] // get the token from header
	_, _, exists := searchToken(token)
	if !exists {
		w.WriteHeader(400)
		returnMsg(w, "token not found, "+
			"please provide a valid one")
		return
	}

	var status Status
	hostname, err := os.Hostname()
	if err != nil {
		w.WriteHeader(500)
		returnMsg(w, "internal server error"+
			"couldn't get server name")
		return
	}

	status = Status{
		SystemName: hostname,
		ServerTime: time.Now().String(),
		Workloads:  Workloads,
	}

	json.NewEncoder(w).Encode(status)
}

func postWorkloads(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO]: POST /workloads requested")

	// handle token
	tmp := r.Header.Get("Authorization")
	if strings.Fields(tmp)[0] != "Bearer" {
		w.WriteHeader(400)
		returnMsg(w, "bad request, check headers "+
			"you must send a Bearer token")
		return
	}
	token := strings.Fields(tmp)[1] // get the token from header
	_, _, exists := searchToken(token)
	//fmt.Println(user)
	if !exists {
		w.WriteHeader(400)
		returnMsg(w, "token not found, "+
			"please provide a valid one")
		return
	}

	// handle body request
	body, _ := ioutil.ReadAll(r.Body)
	var workloadreq WorkloadReq
	json.Unmarshal(body, &workloadreq)

	// check if json sent is correct
	if workloadreq.Filter == "" ||
		workloadreq.WorkloadName == "" {
		w.WriteHeader(400)
		returnMsg(w, "bad request, "+
			"json sent misspelled or missing field")
		return
	}

	// create workload struct
	var workload Workload
	workload.Id = workloadsIds
	workloadsIds += 1
	workload.Filter = workloadreq.Filter
	workload.Name = workloadreq.WorkloadName
	workload.Status = "completed"
	workload.RunningJobs = 0
	workload.Images = nil
	Workloads = append(Workloads, workload)

	// transform to string
	workloadStr, err := json.Marshal(workload)
	if err != nil {
		w.WriteHeader(500)
		returnMsg(w, "server internal error, "+
			"couldnt marshal json")
		return

	}

	// push workload to controller
	pushMsg(workloadsUrl, string(workloadStr))

	json.NewEncoder(w).Encode(workload)
}

func getWorkloads(w http.ResponseWriter, r *http.Request) {

	// handle token
	tmp := r.Header.Get("Authorization")
	if strings.Fields(tmp)[0] != "Bearer" {
		w.WriteHeader(400)
		returnMsg(w, "bad request, check headers "+
			"you must send a Bearer token")
		return
	}
	token := strings.Fields(tmp)[1] // get the token from header
	_, user, exists := searchToken(token)
	fmt.Println(user)
	if !exists {
		w.WriteHeader(400)
		returnMsg(w, "token not found, "+
			"please provide a valid one")
		return
	}

	// read path params
	vars := mux.Vars(r)
	id := vars["workload_id"]
	if id == "" {
		w.WriteHeader(400)
		returnMsg(w, "id missing, "+
			"you should do smthg like workloads/{workload_id}")
		return
	}
	fmt.Println("[INFO]: GET /workloads/" + id + " requested")

	intId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		w.WriteHeader(400)
		returnMsg(w, "you didnt send a valid number, "+
			"please check again")
		return

	}

	if intId >= workloadsIds || workloadsIds == 0 {
		w.WriteHeader(400)
		returnMsg(w, "that id doesnt exists, "+
			"please check again")
		return

	}

	json.NewEncoder(w).Encode(Workloads[intId])
}

/********************* Handler Functions ***************************/

func handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	case http.MethodPost:
		postLogin(w, r) // post
	case http.MethodPut:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	case http.MethodDelete:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	default:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	}

}
func handleLogout(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	case http.MethodPost:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	case http.MethodPut:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	case http.MethodDelete:
		delLogout(w, r) // delete
	default:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	}

}
func handleImages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getImages(w, r) // get
	case http.MethodPost:
		postImages(w, r) // post
	case http.MethodPut:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	case http.MethodDelete:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	default:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	}

}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getStatus(w, r) //get
	case http.MethodPost:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	case http.MethodPut:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	case http.MethodDelete:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	default:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	}

}

func handleWorkloads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getWorkloads(w, r) //get
	case http.MethodPost:
		postWorkloads(w, r) //post
	case http.MethodPut:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	case http.MethodDelete:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	default:
		w.WriteHeader(404)
		returnMsg(w, "page not found")
	}

}

func handleRequests() {

	// create the gorilla/mux http router, this
	// will help us parsing the path params in
	// the endpoints
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", homePage)
	router.HandleFunc("/login", handleLogin)
	router.HandleFunc("/logout", handleLogout)
	router.HandleFunc("/status", handleStatus)
	//TODO
	router.HandleFunc("/workloads", handleWorkloads)
	router.HandleFunc("/workloads/{workload_id}", handleWorkloads)
	router.HandleFunc("/images", handleImages)
	router.HandleFunc("/images/{image_id}", handleImages)

	// no longer usefull
	//router.HandleFunc("/upload", handleUpload)

	log.Fatal(http.ListenAndServe(":8080", router))
}

/********************* Helper Functions ***************************/

// Search token in Users, returned index, user struct
// and boolean that tells us if it was found.
func searchToken(token string) (int, User, bool) {
	for i, user := range Users {
		if user.Token == token {
			return i, user, true
		}
	}
	var tmp User
	return -1, tmp, false
}

// swap the user you want to remove with the
// last item, return the slice without the last item
func removeUser(users []User, index int) []User {
	users[index] = users[len(users)-1]
	return users[:len(users)-1]
}

func returnMsg(w http.ResponseWriter, msg string) {
	var msgJSON Message
	msgJSON = Message{
		Message: msg,
	}
	json.NewEncoder(w).Encode(msgJSON)

}

func Start() {
	handleRequests()
}
