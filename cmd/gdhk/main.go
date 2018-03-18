package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

// CHANGE ME
const httpuser = "admin"
const httppass = "password"
const doorAPI = "192.168.0.100"

type GarageDoor struct {
	*accessory.Accessory
	Opener *service.GarageDoorOpener
	Button *service.Switch
	state  int
	guard  chan struct{}
}

func NewGarageDoor(info accessory.Info) *GarageDoor {
	acc := GarageDoor{
		Accessory: accessory.New(info, accessory.TypeGarageDoorOpener),
		Button:    service.NewSwitch(),
		Opener:    service.NewGarageDoorOpener(),
		guard:     make(chan struct{}, 1),
	}
	acc.AddService(acc.Opener.Service)
	acc.AddService(acc.Button.Service)

	acc.Opener.CurrentDoorState.OnValueRemoteGet(acc.getState)
	acc.Opener.TargetDoorState.OnValueRemoteGet(acc.getTargetState)
	acc.Opener.TargetDoorState.OnValueRemoteUpdate(acc.setState)
	acc.Opener.CurrentDoorState.SetEventsEnabled(true)
	acc.Button.On.OnValueRemoteUpdate(acc.pressButton)

	// Load a token into the guard channel
	acc.guard <- struct{}{}

	return &acc
}

func main() {
	info := accessory.Info{
		Name:         "GarageDoor",
		SerialNumber: "GDOOR-0001",
		Manufacturer: "forfuncsake",
		Model:        "GDHK",
	}
	door := NewGarageDoor(info)

	config := hc.Config{Pin: "12344321", StoragePath: "/usr/local/garagedoor/var/db"}
	t, err := hc.NewIPTransport(config, door.Accessory)
	if err != nil {
		log.Panic(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	http.HandleFunc("/refresh", func(http.ResponseWriter, *http.Request) {
		door.getState()
		door.Opener.CurrentDoorState.SetValue(door.state)
		door.Opener.TargetDoorState.SetValue(door.getTargetState())
	})

	go func() {
		http.ListenAndServe(":8180", nil)
	}()

	log.Println("Starting...")
	t.Start()
}

func (d *GarageDoor) getTargetState() (state int) {
	switch d.getState() {
	case characteristic.CurrentDoorStateClosed, characteristic.CurrentDoorStateClosing:
		return characteristic.TargetDoorStateClosed
	case characteristic.CurrentDoorStateOpen, characteristic.CurrentDoorStateOpening:
		return characteristic.TargetDoorStateOpen
	default:
		return characteristic.CurrentDoorStateStopped
	}
}

const press int = -5

var stateURL = map[int]string{
	characteristic.TargetDoorStateOpen:   "/open",
	characteristic.TargetDoorStateClosed: "/close",
	press: "/press",
}

func (d *GarageDoor) pressButton(bool) {
	d.setState(press)
}

func (d *GarageDoor) setState(to int) {
	log.Printf("setState called\n")

	path, _ := stateURL[to]

	req, err := http.NewRequest(http.MethodPost, "http://"+doorAPI+path, nil)
	if err != nil {
		log.Printf("failed to create POST request: %v", err)
		return
	}

	req.SetBasicAuth(httpuser, httppass)
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("failed to post to button: %v", err)
	}
}

func (d *GarageDoor) getState() (state int) {
	log.Printf("getState called\n")
	defer func() {
		if state >= 0 {
			d.state = state
		} else {
			state = characteristic.CurrentDoorStateStopped
		}
	}()
	state = -1

	// Guard the module from being probed more than once a second
	// by returning the last saved state
	select {
	case <-d.guard:
		go func() {
			<-time.After(1 * time.Second)
			d.guard <- struct{}{}
		}()
	default:
		log.Println("Guarding probe from excessive polls")
		return d.state
	}

	resp, err := http.Get("http://" + doorAPI)
	if err != nil {
		log.Printf("error getting status: %v\n", err)
		return state
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading status response: %v\n", err)
		return state
	}

	var a struct {
		Success bool   `json:"success"`
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	err = json.Unmarshal(b, &a)
	if err != nil {
		log.Printf("error marshalling status response: %v\n", err)
		return state
	}

	if !a.Success {
		log.Printf("got error from API: %s\n", a.Message)
		if a.Status < 1 {
			return state
		}
	}

	log.Printf("returning %d", a.Status)
	state = a.Status

	return state
}
