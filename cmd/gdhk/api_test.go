package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type state bool

const open state = false
const closed state = true

func (s state) String() string {
	if s {
		return "closed"
	}
	return "open"
}
func (s state) Int() int {
	if s {
		return 1
	}
	return 0
}

// api is a mock implementation of the API that runs
// on the ESP8266 device
type api struct {
	port   int
	status state

	Close   func() error
	pressed int
}

func newAPI() (*api, error) {
	// Create tcp listener on ephemeral port
	listener, err := net.Listen("tcp4", ":0")
	if err != nil {
		return nil, fmt.Errorf("could not create listener: %v", err)
	}

	// Capture the allocated port
	a := &api{
		port: listener.Addr().(*net.TCPAddr).Port,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", a.respond)

	server := &http.Server{
		Handler: mux,
	}

	go func() {
		err := server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			fmt.Printf("http server exited with error: %v\n", err)
		}
	}()

	a.Close = server.Close
	return a, nil
}

func (a *api) respond(w http.ResponseWriter, r *http.Request) {
	resp := APIResponse{Success: true}

	if strings.HasPrefix(r.URL.Path, "/open") {
		a.status = open
	}

	if strings.HasPrefix(r.URL.Path, "/close") {
		a.status = closed
	}

	if strings.HasPrefix(r.URL.Path, "/press") {
		a.pressed++
	}

	resp.Status = a.status.Int()
	resp.Message = fmt.Sprintf("The Garage Door is %s", a.status)

	// Hack to test response values
	if nums, ok := r.URL.Query()["num"]; ok {
		if i, err := strconv.Atoi(nums[0]); err == nil {
			log.Printf("num override detected - returning: %d\n", i)
			resp.Status = i
		}
	}

	b, _ := json.Marshal(resp)
	w.Write(b)
}
