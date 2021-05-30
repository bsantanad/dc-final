package main

import (
	"log"

	"github.com/bsantanad/dc-final/api"
	"github.com/bsantanad/dc-final/controller"
	"github.com/bsantanad/dc-final/scheduler"
)

func main() {
	log.Println("Welcome to the Distributed and " +
		"Parallel Image Processing System")

	// Start Controller
	go controller.Start()
	go scheduler.Start()

	// API
	api.Start()

}
