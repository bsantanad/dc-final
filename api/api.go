// SPDX-License-Identifier: GPL-3.0-or-later
// authors: bsantanad & renataaparicio
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
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
	WorkloadId string `json:"workload_id"`
	ImageId    string `json:"image_id"`
	Type       string `json:"type"`
	Data       []byte `json:"data"`
}

type User struct {
	Username string  `json:"user"`
	Token    string  `json:"token"`
	Images   []Image `json:"image"`
	Time     string  `json:"time"`
}

type Status struct {
	Message string `json:"message"`
	Time    string `json:"time"`
}

type ImageMsg struct {
	Message    string `json:"message"`
	WorkloadId string `json:"workload_id"`
	ImageId    string `json:"image_id"`
	Type       string `json:"type"`
}

type Message struct {
	Message string `json:"message"`
}

type WorkloadReq struct {
	Filter       string `json:"filter"`
	WorkloadName string `json:"workload_name"`
}

type Workload struct {
	WorkloadId     string `json:"workload_id"`
	Filter         string `json:"filter"`
	WorkloadName   string `json:"workload_name"`
	Status         string `json:"status"`
	RunningJobs    int    `json:"running_jobs"`
	FilteredImages string `json:"filtered_images"`
}

type ImageReq struct {
	WorkloadId string `json:"workload_id"`
}

var Users []User /* this will act as our DB */

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
	var buf bytes.Buffer
	file, _, err := r.FormFile("data")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer file.Close()
	// Copy the image data to my buffer
	io.Copy(&buf, file)

	//FIXME real info
	// Fill the image struct
	var image Image
	image.WorkloadId = "tmp"
	image.ImageId = "tmp"
	image.Type = "tmp"
	image.Data, err = buf.ReadBytes(254)
	if err != nil {
		w.WriteHeader(409)
		returnMsg(w, "Image couldn't be uploaded :(. Please try again")
		return
	}
	Users[index].Images = append(user.Images, image)

	buf.Reset()

	var msg ImageMsg
	msg = ImageMsg{
		Message:    "An image has been successfully uploaded :)",
		WorkloadId: image.WorkloadId,
		ImageId:    image.ImageId,
		Type:       image.Type,
	}

	json.NewEncoder(w).Encode(msg)
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
	_, user, exists := searchToken(token)
	if !exists {
		w.WriteHeader(400)
		returnMsg(w, "token not found, "+
			"please provide a valid one")
		return
	}

	var status Status
	status = Status{
		Message: "Hi " + user.Username + ", the DPIP System is Up and Running",
		Time:    user.Time,
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
	_, user, exists := searchToken(token)
	fmt.Println(user)
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

	json.NewEncoder(w).Encode(workloadreq)
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

	//FIXME real info
	json.NewEncoder(w).Encode("hola")
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
		w.WriteHeader(404)
		returnMsg(w, "page not found")
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
	router.HandleFunc("/workloads", handleWorkloads)               // POST
	router.HandleFunc("/workloads/{workload_id}", handleWorkloads) // GET
	router.HandleFunc("/images", handleImages)                     // POST cp upload
	//router.HandleFunc("/images/{image_id}", handleImagesId) // POST cp upload

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
